package CommandLine

import (
	"errors"
	"flag"
	"fmt"
	"github.com/koushamad/blockchain/BlockChain"
	"github.com/koushamad/blockchain/Handler"
	"github.com/koushamad/blockchain/Wallet"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct{}

func (cli *CommandLine) PrintUsage() {
	fmt.Println("Usage:")
	fmt.Println("get-balance -address ADDRESS -get the balance for address")
	fmt.Println("create-blockchain -address Address creates a blockchain")
	fmt.Println("print-chain - prints the block in the chain")
	fmt.Println("send -from FROM -to TO -amount AMOUNT - Send amount")
	fmt.Println("create-wallet Creates a new Wallet")
	fmt.Println("list-address List the address in our wallet file")
	fmt.Println("reindex-utxo Rebuilds the UTXO set")

}

func (cli *CommandLine) ValidateArgs() {
	if len(os.Args) < 2 {
		cli.PrintUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) PrintChain() {

	chain := BlockChain.ContinueBlockChain("")
	defer chain.Database.Close()
	iter := chain.Iterator()

	fmt.Println()

	for {
		block := iter.Next()
		pow := BlockChain.NewProof(block)

		fmt.Printf("PrevHash:	%x\n", block.PrevHash)
		fmt.Printf("Hash:	%x\n", block.Hash)
		fmt.Printf("PoW:	%s\n", strconv.FormatBool(pow.Validate()))

		for _, tx := range block.Transactions {
			fmt.Println(tx)
		}

		fmt.Println()

		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) CreateBlockChain(address string) {
	if !Wallet.ValidateAddress(address) {
		Handler.Handle(errors.New("address is not valid"))
	}

	chain := BlockChain.InitBlockChain(address)
	chain.Database.Close()
	UTOXSet := BlockChain.UTXOSet{chain}
	UTOXSet.Reindex()

	fmt.Println("Finished!")
}

func (cli *CommandLine) GetBalance(address string) {
	if !Wallet.ValidateAddress(address) {
		Handler.Handle(errors.New("address is not valid"))
	}

	chain := BlockChain.ContinueBlockChain(address)
	UTXOSet := BlockChain.UTXOSet{Chain: chain}
	defer chain.Database.Close()

	balance := 0
	pubKeyHash := Wallet.Base58Decode([]byte(address))
	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-Wallet.ChecksumLength]
	UTXOs := UTXOSet.FindUnspentTransactions(pubKeyHash)

	for _, out := range UTXOs {
		balance += out.Value
	}

	fmt.Printf(" Balance of %s: %d \n", address, balance)
}

func (cli *CommandLine) ReindexUTXO() {
	chain := BlockChain.ContinueBlockChain("")
	defer chain.Database.Close()

	UTXOSet := BlockChain.UTXOSet{Chain: chain}
	UTXOSet.Reindex()

	count := UTXOSet.CountTransactions()
	fmt.Printf("Done! there are %d transactions in the UTXO set.\n", count)
}

func (cli *CommandLine) ListAddress() {
	wallets, err := Wallet.CreateWallets()
	Handler.Handle(err)

	chain := BlockChain.ContinueBlockChain("")
	defer chain.Database.Close()
	UTXOSet := BlockChain.UTXOSet{Chain: chain}
	addresses := wallets.GetAllAddresses()

	fmt.Println()
	for address, wallet := range addresses {

		pubKeyHash := Wallet.PublicKeyHash(wallet.PublicKey)
		total, _ := UTXOSet.FindAllSpendableOutputs(pubKeyHash)

		fmt.Printf("Wallet 	address:		%s 			Value: 	%d\n 	Pulic Key Hash: 	%x\n 	Public Key: 		%x\n 	Private Key:		%x\n\n", address, total, Wallet.PublicKeyHash(wallet.PublicKey), wallet.PublicKey, wallet.PrivateKey)
	}
	fmt.Println()
}

func (cli *CommandLine) Send(from, to string, amount int) {
	if !Wallet.ValidateAddress(from) {
		Handler.Handle(errors.New("address is not valid"))
	}

	if !Wallet.ValidateAddress(to) {
		Handler.Handle(errors.New("address is not valid"))
	}

	chain := BlockChain.ContinueBlockChain(from)
	UTXOSet := BlockChain.UTXOSet{Chain: chain}
	defer chain.Database.Close()

	tx := BlockChain.NewTransaction(from, to, amount, &UTXOSet)

	//cbTx := BlockChain.CoinbaseTX("17C2p8mb5g12P17f5cCuxe4BU5GcrQ6djH", "")
	var transactions []*BlockChain.Transaction
	//transactions = append(transactions, cbTx)
	transactions = append(transactions, tx)
	block := chain.AddBlock(transactions)
	UTXOSet.Update(block)
	fmt.Println("Success!")
}

func (cli *CommandLine) CreateWallet() {
	wallets, err := Wallet.CreateWallets()
	Handler.Handle(err)
	address := wallets.AddWallet()
	wallets.SaveFile()

	fmt.Printf("New address is: %s\n", address)
}

func (cli *CommandLine) Run() {
	cli.ValidateArgs()

	getBalanceCmd := flag.NewFlagSet("get-balance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("create-blockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print-chain", flag.ExitOnError)
	createWalletCmd := flag.NewFlagSet("create-wallet", flag.ExitOnError)
	listAddressCmd := flag.NewFlagSet("list-address", flag.ExitOnError)
	reindexUTXOCmd := flag.NewFlagSet("reindex-utxo", flag.ExitOnError)

	getBalanceAddress := getBalanceCmd.String("address", "", "The address wont to get balance")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address for create blockchain")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")

	switch os.Args[1] {
	case "get-balance":
		err := getBalanceCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "create-blockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "print-chain":
		err := printChainCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "create-wallet":
		err := createWalletCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "list-address":
		err := listAddressCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "reindex-utxo":
		err := reindexUTXOCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	}

	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.GetBalance(*getBalanceAddress)
	} else if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.CreateBlockChain(*createBlockchainAddress)
	} else if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount == 0 {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.Send(*sendFrom, *sendTo, *sendAmount)
	} else if printChainCmd.Parsed() {
		cli.PrintChain()
	} else if createWalletCmd.Parsed() {
		cli.CreateWallet()
	} else if listAddressCmd.Parsed() {
		cli.ListAddress()
	} else if reindexUTXOCmd.Parsed() {
		cli.ReindexUTXO()
	} else {
		cli.PrintUsage()
	}

}
