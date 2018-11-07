package main

import (
	"github.com/U-Network/UNetwork/testing"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	testing.AddCmd(testing.GenerateKey)
	testing.AddCmd(testing.TestCmd)
	testing.AddCmd(testing.BalanceCmd)
	testing.Execute()
}
