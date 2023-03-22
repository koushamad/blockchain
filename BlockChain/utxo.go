package BlockChain

import (
	"bytes"
	"encoding/hex"
	"github.com/dgraph-io/badger"
	"github.com/koushamad/blockchain/Handler"
)

var (
	UTXOPrefix   = []byte("utxo-")
	PrefixLength = len(UTXOPrefix)
)

type UTXOSet struct {
	Chain *Chain
}

func (u UTXOSet) Reindex() {
	db := u.Chain.Database

	u.DeleteByPrefix(UTXOPrefix)
	UTXO := u.Chain.FindUTXO()

	err := db.Update(func(txn *badger.Txn) error {
		for txId, outs := range UTXO {
			key, err := hex.DecodeString(txId)
			if err != nil {
				return err
			}

			key = append(UTXOPrefix, key...)
			err = txn.Set(key, outs.Serialize())
			return err
		}
		return nil
	})
	Handler.Handle(err)
}

func (u *UTXOSet) Update(block *Block) {
	db := u.Chain.Database
	err := db.Update(func(txn *badger.Txn) error {
		for _, tx := range block.Transactions {
			if tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					updateOuts := TxOutputs{}
					inID := append(UTXOPrefix, in.ID...)
					item, err := txn.Get(inID)
					if err != nil {
						return err
					}

					var outs TxOutputs
					_ = item.Value(func(val []byte) error {
						outs = DeserializeOutputs(val)
						return nil
					})

					for outIdx, out := range outs.Outputs {
						if outIdx != in.Out {
							updateOuts.Outputs = append(updateOuts.Outputs, out)
						}
					}
					if len(updateOuts.Outputs) == 0 {
						if err := txn.Delete(inID); err != nil {
							return err
						}
					} else {
						if err := txn.Set(inID, updateOuts.Serialize()); err != nil {
							return err
						}
					}
				}
			}

			newOutputs := TxOutputs{}
			for _, out := range tx.Outputs {
				newOutputs.Outputs = append(newOutputs.Outputs, out)
			}

			txID := append(UTXOPrefix, tx.ID...)
			if err := txn.Set(txID, newOutputs.Serialize()); err != nil {
				return err
			}
		}

		return nil
	})

	Handler.Handle(err)
}

func (u UTXOSet) FindUnspentTransactions(publicKeyHash []byte) []TXOutput {
	var UTXOs []TXOutput

	db := u.Chain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			item := it.Item()
			var opts TxOutputs

			if err := item.Value(func(val []byte) error {
				opts = DeserializeOutputs(val)
				return nil
			}); err != nil {
				return err
			}

			for _, out := range opts.Outputs {
				if out.IsLockedWithKey(publicKeyHash) {
					UTXOs = append(UTXOs, out)
				}
			}
		}

		return nil
	})

	Handler.Handle(err)

	return UTXOs
}

func (u UTXOSet) FindSpendableOutputs(publicKeyHash []byte, amount int) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.Chain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			var opts TxOutputs

			if err := item.Value(func(val []byte) error {
				opts = DeserializeOutputs(val)
				return nil
			}); err != nil {
				return err
			}

			k = bytes.TrimPrefix(k, UTXOPrefix)
			txId := hex.EncodeToString(k)

			for outIdx, out := range opts.Outputs {
				if out.IsLockedWithKey(publicKeyHash) && accumulated < amount {
					accumulated += out.Value
					unspentOuts[txId] = append(unspentOuts[txId], outIdx)
				}
			}
		}

		return nil
	})

	Handler.Handle(err)

	return accumulated, unspentOuts
}

func (u UTXOSet) FindAllSpendableOutputs(publicKeyHash []byte) (int, map[string][]int) {
	unspentOuts := make(map[string][]int)
	accumulated := 0
	db := u.Chain.Database

	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			item := it.Item()
			k := item.Key()
			var outs TxOutputs

			if err := item.Value(func(val []byte) error {
				outs = DeserializeOutputs(val)
				return nil
			}); err != nil {
				return err
			}

			k = bytes.TrimPrefix(k, UTXOPrefix)
			txId := hex.EncodeToString(k)

			for outIdx, out := range outs.Outputs {
				if out.IsLockedWithKey(publicKeyHash) {
					accumulated += out.Value
					unspentOuts[txId] = append(unspentOuts[txId], outIdx)
				}
			}
		}

		return nil
	})

	Handler.Handle(err)

	return accumulated, unspentOuts
}

func (u UTXOSet) CountTransactions() int {
	db := u.Chain.Database
	counter := 0

	err := db.View(func(txn *badger.Txn) error {
		outs := badger.DefaultIteratorOptions

		it := txn.NewIterator(outs)
		defer it.Close()

		for it.Seek(UTXOPrefix); it.ValidForPrefix(UTXOPrefix); it.Next() {
			counter++
		}

		return nil
	})

	Handler.Handle(err)

	return counter
}

func (u UTXOSet) DeleteByPrefix(prefix []byte) {
	deleteKeys := func(keysForDelete [][]byte) error {
		if err := u.Chain.Database.Update(func(txn *badger.Txn) error {
			for _, key := range keysForDelete {
				if err := txn.Delete(key); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return err
		}
		return nil
	}

	collectSize := 10000
	err := u.Chain.Database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		keysForDelete := make([][]byte, 0, collectSize)
		keysCollected := 0
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			key := it.Item().KeyCopy(nil)
			keysForDelete = append(keysForDelete, key)
			if keysCollected == collectSize {
				if err := deleteKeys(keysForDelete); err != nil {
					return err
				}
				keysForDelete = make([][]byte, 0, collectSize)
				keysCollected = 0
			}
		}

		if keysCollected > 0 {
			if err := deleteKeys(keysForDelete); err != nil {
				return err
			}
		}

		return nil
	})

	Handler.Handle(err)
}
