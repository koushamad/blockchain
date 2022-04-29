package Block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"github.com/koushamad/blockchain/Handler"
	"github.com/koushamad/blockchain/Transaction"
)

type Block struct {
	Hash         []byte
	Transactions []*Transaction.Transaction
	PrevHash     []byte
	Nonce        int
}

func Genesis(coinbase *Transaction.Transaction) *Block {
	return CreateBlock([]*Transaction.Transaction{coinbase}, []byte{})
}

func CreateBlock(txs []*Transaction.Transaction, prevHash []byte) *Block {
	block := &Block{[]byte{}, txs, prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	return block
}

func (b Block) HashTransactions() []byte {
	var txHashes [][]byte
	var txHash [32]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.ID)
	}

	txHash = sha256.Sum256(bytes.Join(txHashes, []byte{}))

	return txHash[:]
}

func (b *Block) Serialize() []byte {
	var res bytes.Buffer
	encoder := gob.NewEncoder(&res)

	err := encoder.Encode(b)
	Handler.Handle(err)

	return res.Bytes()
}

func Deserialize(data []byte) *Block {
	var block Block
	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&block)
	Handler.Handle(err)

	return &block
}

//func (b *Block) DeriveHash() {
//	info := bytes.Join([][]byte{b.Data, b.PrevHash}, []byte{})
//	hash := sha256.Sum256(info)
//	b.Hash = hash[:]
//}
