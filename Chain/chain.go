package Chain

import (
	"fmt"
	"github.com/dgraph-io/badger"
	"github.com/koushamad/blockchain/Block"
	"github.com/koushamad/blockchain/Handler"
	"github.com/koushamad/blockchain/Iterator"
	"github.com/koushamad/blockchain/Transaction"
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
		gbtx := Transaction.CoinbaseTX(address, genesisData)
		genesis := Block.Genesis(gbtx)
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

func (chain *Chain) AddBlock(data string) {
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

	newBlock := Block.CreateBlock(data, lastHash)
	err = chain.Database.Update(func(txn *badger.Txn) error {
		err = txn.Set(newBlock.Hash, newBlock.Serialize())
		Handler.Handle(err)
		err = txn.Set([]byte("lh"), newBlock.Hash)

		chain.LastHash = newBlock.Hash

		return err
	})

	Handler.Handle(err)
}

func (chain *Chain) Iterator() *Iterator.Iterator {
	return &Iterator.Iterator{CurrentHash: chain.LastHash, Database: chain.Database}
}
