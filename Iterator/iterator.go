package Iterator

import (
	"github.com/dgraph-io/badger"
	"github.com/koushamad/blockchain/Block"
	"github.com/koushamad/blockchain/Handler"
)

type Iterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (iter *Iterator) Next() *Block.Block {
	var block *Block.Block

	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		Handler.Handle(err)

		return item.Value(func(val []byte) error {
			block = Block.Deserialize(val)
			return nil
		})
	})

	Handler.Handle(err)

	iter.CurrentHash = block.PrevHash

	return block
}
