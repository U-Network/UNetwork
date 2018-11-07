package miner

import (
	"github.com/U-Network/UNetwork/app/ethereum"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

type MinerExtend struct {
	ethState *ethereum.EthereumWorkState
}

func NewMinerExtend(state *ethereum.EthereumWorkState) *MinerExtend {
	return &MinerExtend{
		ethState: state,
	}
}

func (self *MinerExtend) Pending() (*types.Block, *state.StateDB) {
	self.ethState.Mtx.Lock()
	defer self.ethState.Mtx.Unlock()
	return types.NewBlock(self.ethState.Header, self.ethState.Transactions, nil, self.ethState.Receipts), self.ethState.State.Copy()
}
