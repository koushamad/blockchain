package Transaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"github.com/koushamad/blockchain/Handler"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	OutPuts []TXOutput
}

type TXOutput struct {
	Value  int
	PupKey string
}

type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

func CoinbaseTX(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}

	txin := TxInput{[]byte{}, -1, data}
	txout := TXOutput{100, to}
	tx := Transaction{nil, []TxInput{txin}, []TXOutput{txout}}
	tx.SetID()

	return &tx
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte

	encode := gob.NewEncoder(&encoded)
	err := encode.Encode(tx)
	Handler.Handle(err)

	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func (tx Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

func (out *TXOutput) CanBeUnlocked(data string) bool {
	return out.PupKey == data
}
