package Block

import (
	"bytes"
	"encoding/gob"
	"github.com/koushamad/blockchain/Handler"
)

type Block struct {
	Hash     []byte
	Data     []byte
	PrevHash []byte
	Nonce    int
}

func Genesis() *Block {
	return CreateBlock("Genesis", []byte{})
}

func CreateBlock(data string, prevHash []byte) *Block {
	block := &Block{[]byte{}, []byte(data), prevHash, 0}
	pow := NewProof(block)
	nonce, hash := pow.Run()

	block.Nonce = nonce
	block.Hash = hash

	return block
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
