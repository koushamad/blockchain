package CommandLine

import (
	"flag"
	"fmt"
	"github.com/koushamad/blockchain/Block"
	"github.com/koushamad/blockchain/Chain"
	"github.com/koushamad/blockchain/Handler"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct {
	chain *Chain.Chain
}

func Init(chain *Chain.Chain) *CommandLine {
	return &CommandLine{chain: chain}
}

func (cli *CommandLine) PrintUsage() {
	fmt.Println("Usage:")
	fmt.Println("Add -block BLOCK_DATA - add a block to chain")
	fmt.Println("Print - Prints the blocks in the chain")
}

func (cli *CommandLine) ValidateArgs() {
	if len(os.Args) < 2 {
		cli.PrintUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) AddBlock(data string) {
	cli.chain.AddBlock(data)
	fmt.Println("Added block!")
}

func (cli *CommandLine) PrintChain() {
	iter := cli.chain.Iterator()

	for {
		block := iter.Next()
		pow := Block.NewProof(block)

		fmt.Printf("PrevHash %x\n", block.PrevHash)
		fmt.Printf("Hash %x\n", block.Hash)
		fmt.Printf("Data %s\n", block.Data)
		fmt.Printf("PoW: %s\n\n", strconv.FormatBool(pow.Validate()))
	}
}

func (cli *CommandLine) Run() {
	cli.ValidateArgs()

	addBlockCmd := flag.NewFlagSet("add", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print", flag.ExitOnError)
	addBlockData := addBlockCmd.String("block", "", "Block data")

	switch os.Args[1] {
	case "add":
		err := addBlockCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	case "print":
		err := printChainCmd.Parse(os.Args[2:])
		Handler.Handle(err)
	default:
		cli.PrintUsage()
		runtime.Goexit()
	}

	if addBlockCmd.Parsed() {
		if *addBlockData == "" {
			addBlockCmd.Usage()
			runtime.Goexit()
		}

		cli.AddBlock(*addBlockData)
	}

	if printChainCmd.Parsed() {
		cli.PrintChain()
	}
}
