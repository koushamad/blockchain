package BlockChain

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/koushamad/blockchain/Handler"
	"github.com/koushamad/blockchain/Wallet"
	"os"
	"runtime"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

type Chain struct {
	LastHash []byte
	Database *badger.DB
}

func DBExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}

	return true
}

func InitBlockChain(address string) *Chain {
	if DBExists() {
		fmt.Println("Blockchain already exist")
		runtime.Goexit()
	}

	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handler.Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		gbtx := CoinbaseTX(address, genesisData)
		genesis := Genesis(gbtx)
		err = txn.Set(genesis.Hash, genesis.Serialize())
		Handler.Handle(err)
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err
	})

	Handler.Handle(err)
	chain := Chain{lastHash, db}
	return &chain
}

func ContinueBlockChain(address string) *Chain {
	if DBExists() == false {
		fmt.Println("No existing blockchain found, create one!")
		runtime.Goexit()
	}

	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handler.Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handler.Handle(err)
		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})

		return err
	})
	Handler.Handle(err)

	chain := Chain{lastHash, db}
	return &chain
}

func NewTransaction(from, to string, amount int, UTXO *UTXOSet) *Transaction {
	var inputs []TxInput
	var outputs []TXOutput

	wallets, err := Wallet.CreateWallets()
	Handler.Handle(err)
	w := wallets.GetWallet(from)
	pubKeyHash := Wallet.PublicKeyHash(w.PublicKey)

	acc, validOptions := UTXO.FindSpendableOutputs(pubKeyHash, amount)

	if acc < amount {
		Handler.Handle(errors.New("not enough funds"))
	}

	for txid, outs := range validOptions {
		txID, err := hex.DecodeString(txid)
		Handler.Handle(err)

		for _, out := range outs {
			input := TxInput{ID: txID, Out: out, PubKey: w.PublicKey}
			inputs = append(inputs, input)
		}
	}

	outputs = append(outputs, *NewTXOutput(amount, to))

	if acc > amount {
		outputs = append(outputs, *NewTXOutput(acc-amount, from))
	}

	tx := Transaction{Inputs: inputs, Outputs: outputs}
	tx.ID = tx.Hash()
	UTXO.Chain.SignTransaction(&tx, w.PrivateKey)

	return &tx
}

func (chain *Chain) AddBlock(transactions []*Transaction) *Block {
	var lastHash []byte

	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		Handler.Handle(err)

		err = item.Value(func(val []byte) error {
			lastHash = val
			return nil
		})

		return err
	})

	Handler.Handle(err)

	newBlock := CreateBlock(transactions, lastHash)
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		Handler.Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handler.Handle(err)

	return newBlock
}

func (chain *Chain) FindUTXO() map[string]TxOutputs {
	UTXO := make(map[string]TxOutputs)
	spentTXOs := make(map[string][]int)

	iter := chain.Iterator()

	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)

		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTXOs[txID] != nil {
					for _, spentOut := range spentTXOs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}

				outs := UTXO[txID]
				outs.Outputs = append(outs.Outputs, out)
				UTXO[txID] = outs
			}
			if tx.IsCoinbase() == false {
				for _, in := range tx.Inputs {
					inTXID := hex.EncodeToString(in.ID)
					spentTXOs[inTXID] = append(spentTXOs[inTXID], in.Out)
				}
			}
			if len(block.PrevHash) == 0 {
				break
			}
		}

		return UTXO
	}
}

func (chain *Chain) FindTransaction(ID []byte) (Transaction, error) {
	iter := chain.Iterator()

	for {
		block := iter.Next()

		for _, tx := range block.Transactions {
			if bytes.Compare(tx.ID, ID) == 0 {
				return *tx, nil
			}
		}

		if len(block.PrevHash) == 0 {
			break
		}
	}

	return Transaction{}, errors.New("transaction does not exist")
}

func (chain Chain) SignTransaction(tx *Transaction, priKey ecdsa.PrivateKey) {
	preTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		preTx, err := chain.FindTransaction(in.ID)
		Handler.Handle(err)
		preTXs[hex.EncodeToString(preTx.ID)] = preTx
	}

	tx.Sigh(priKey, preTXs)
}

func (chain Chain) VerifyTransaction(tx *Transaction) bool {

	if tx.IsCoinbase() {
		return true
	}

	preTXs := make(map[string]Transaction)

	for _, in := range tx.Inputs {
		preTx, err := chain.FindTransaction(in.ID)
		Handler.Handle(err)
		preTXs[hex.EncodeToString(preTx.ID)] = preTx
	}

	return tx.Verify(preTXs)
}

func (chain *Chain) Iterator() *Iterator {
	return &Iterator{CurrentHash: chain.LastHash, Database: chain.Database}
}
