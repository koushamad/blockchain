package BlockChain

import (
	"github.com/dgraph-io/badger"
	"github.com/koushamad/blockchain/Handler"
)

type Iterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (iter *Iterator) Next() *Block {
	var block *Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handler.Handle(err)

		return item.Value(func(val []byte) error {
			block = Deserialize(val)
			return nil
		})
	})

	Handler.Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}
