package evm

import (
	"UNetwork/vm/evm/common"
	"math/big"
)

func memoryMStore(stack *Stack) *big.Int {
	return common.CalcMemSize(stack.Back(0), big.NewInt(32))
}
