package BlockChain

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/koushamad/blockchain/Handler"
	"github.com/koushamad/blockchain/Wallet"
	"math/big"
	"strings"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TXOutput
}

type TXOutput struct {
	Value      int
	PupKeyHash []byte
}

type TxOutputs struct {
	Outputs []TXOutput
}

type TxInput struct {
	ID        []byte
	Out       int
	Signature []byte
	PubKey    []byte
}

func NewTXOutput(value int, address string) *TXOutput {
	txo := &TXOutput{value, nil}
	txo.Lock([]byte(address))

	return txo
}

func CoinbaseTX(to, data string) *Transaction {
	if data == "" {
		randData := make([]byte, 24)

		if _, err := rand.Read(randData); err != nil {
			Handler.Handle(err)
		}
		data = fmt.Sprintf("%x", randData)
	}

	txin := TxInput{[]byte{}, -1, nil, []byte(data)}
	txout := NewTXOutput(20, to)
	tx := Transaction{nil, []TxInput{txin}, []TXOutput{*txout}}
	tx.ID = tx.Hash()

	return &tx
}

func (tx *Transaction) Hash() []byte {
	var hash [32]byte

	txCopy := *tx
	txCopy.ID = []byte{}

	hash = sha256.Sum256(txCopy.Serialize())
	return hash[:]
}

func (tx *Transaction) TrimmedCopy() Transaction {
	var inputs []TxInput
	var outputs []TXOutput

	for _, in := range tx.Inputs {
		inputs = append(inputs, TxInput{in.ID, in.Out, nil, nil})
	}

	for _, out := range tx.Outputs {
		outputs = append(outputs, TXOutput{out.Value, out.PupKeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer

	enc := gob.NewEncoder(&encoded)
	err := enc.Encode(tx)
	Handler.Handle(err)

	return encoded.Bytes()
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

func (tx *Transaction) Sigh(priKey ecdsa.PrivateKey, preTXs map[string]Transaction) {
	if tx.IsCoinbase() {
		return
	}

	for _, in := range tx.Inputs {
		if preTXs[hex.EncodeToString(in.ID)].ID == nil {
			Handler.Handle(errors.New("previous transaction dons not exist"))
		}
	}

	txCopy := tx.TrimmedCopy()

	for inId, in := range txCopy.Inputs {
		preTX := preTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = preTX.Outputs[in.Out].PupKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r, s, err := ecdsa.Sign(rand.Reader, &priKey, txCopy.ID)
		Handler.Handle(err)
		signature := append(r.Bytes(), s.Bytes()...)
		tx.Inputs[inId].Signature = signature
	}
}

func (tx *Transaction) Verify(preTXs map[string]Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	for _, in := range tx.Inputs {
		if preTXs[hex.EncodeToString(in.ID)].ID == nil {
			Handler.Handle(errors.New("previous transaction does not exist"))
		}
	}

	txCopy := tx.TrimmedCopy()
	curve := elliptic.P256()

	for inId, in := range tx.Inputs {
		preTx := preTXs[hex.EncodeToString(in.ID)]
		txCopy.Inputs[inId].Signature = nil
		txCopy.Inputs[inId].PubKey = preTx.Outputs[in.Out].PupKeyHash
		txCopy.ID = txCopy.Hash()
		txCopy.Inputs[inId].PubKey = nil

		r := big.Int{}
		s := big.Int{}
		sigLen := len(in.Signature)

		r.SetBytes(in.Signature[:(sigLen / 2)])
		s.SetBytes(in.Signature[(sigLen / 2):])

		x := big.Int{}
		y := big.Int{}

		keyLen := len(in.PubKey)
		x.SetBytes(in.PubKey[:(keyLen / 2)])
		y.SetBytes(in.PubKey[(keyLen / 2):])

		rawPubKey := ecdsa.PublicKey{curve, &x, &y}
		if ecdsa.Verify(&rawPubKey, txCopy.ID, &r, &s) == false {
			return false
		}
	}

	return true
}

func (tx Transaction) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("	------	Transaction:	%x\n", tx.ID))

	for i, input := range tx.Inputs {
		lines = append(lines, fmt.Sprintf("		Input:	 	%d", i))
		lines = append(lines, fmt.Sprintf("		TXID: 		%x", input.ID))
		lines = append(lines, fmt.Sprintf("		Out: 		%d", input.Out))
		lines = append(lines, fmt.Sprintf("		Sixnature:	%x", input.Signature))
		lines = append(lines, fmt.Sprintf("		PublicKey:	%x", input.PubKey))
	}

	for i, output := range tx.Outputs {
		lines = append(lines, fmt.Sprintf("		Output:		%d", i))
		lines = append(lines, fmt.Sprintf("		Value:		%d", output.Value))
		lines = append(lines, fmt.Sprintf("		Script:		%x\n", output.PupKeyHash))
	}

	return strings.Join(lines, "\n")
}

func (in *TxInput) UsesKey(publicKeyHash []byte) bool {
	lockingHash := Wallet.PublicKeyHash(in.PubKey)

	return bytes.Compare(lockingHash, publicKeyHash) == 0
}

func (out *TXOutput) Lock(address []byte) {
	pubKeyHash := Wallet.Base58Decode(address)
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-Wallet.ChecksumLength]
	out.PupKeyHash = pubKeyHash
}

func (out *TXOutput) IsLockedWithKey(pubKeyHash []byte) bool {
	return bytes.Compare(out.PupKeyHash, pubKeyHash) == 0
}

func (outs TxOutputs) Serialize() []byte {
	var buffer bytes.Buffer
	encode := gob.NewEncoder(&buffer)
	err := encode.Encode(outs)
	Handler.Handle(err)
	return buffer.Bytes()
}

func DeserializeOutputs(data []byte) TxOutputs {
	var outputs TxOutputs
	decode := gob.NewDecoder(bytes.NewReader(data))
	err := decode.Decode(&outputs)
	Handler.Handle(err)
	return outputs
}
