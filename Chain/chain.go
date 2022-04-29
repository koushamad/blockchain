package Chain

import (
	"github.com/dgraph-io/badger"
	"github.com/koushamad/blockchain/Block"
	"github.com/koushamad/blockchain/Handler"
	"github.com/koushamad/blockchain/Iterator"
)

const (
	dbPath = "./tmp/blocks"
)

type Chain struct {
	LastHash []byte
	Database *badger.DB
}

func InitBlockChain() *Chain {
	var lastHash []byte

	opts := badger.DefaultOptions(dbPath)
	db, err := badger.Open(opts)
	Handler.Handle(err)

	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			genesis := Block.Genesis()
			err = txn.Set(genesis.Hash, genesis.Serialize())
			Handler.Handle(err)
			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		} else {
			item, err := txn.Get([]byte("lh"))
			Handler.Handle(err)

			err = item.Value(func(val []byte) error {
				lastHash = val
				return nil
			})

			return err
		}
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
