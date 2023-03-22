package main

import (
	"github.com/koushamad/blockchain/CommandLine"
	"os"
)

func main() {
	defer os.Exit(0)
	cli := CommandLine.CommandLine{}
	cli.Run()
	//cli.Send("1GxqJ24MonYcmNuziJdt3Q1rdD7kpD6mCt", "17yQBRti8d9ebtQsP6Vtgr7SVQePqEdB9P", 1)
	//cli.ReindexUTXO()
	//cli.ListAddress()
}
