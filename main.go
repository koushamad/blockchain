package main

import (
	"github.com/koushamad/blockchain/Chain"
	"github.com/koushamad/blockchain/CommandLine"
	"os"
)

func main() {
	defer os.Exit(0)
	chain := Chain.InitBlockChain()
	defer chain.Database.Close()

	cli := CommandLine.Init(chain)
	cli.Run()

}
