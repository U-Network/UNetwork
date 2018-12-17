package api

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"unsafe"
)

type FreeGasAPI struct {
	b Backend
}

func NewFreeGasAPI(b Backend) *FreeGasAPI {
	return &FreeGasAPI{b}
}

func (s *FreeGasAPI) GetUsed(ctx context.Context, address common.Address) (*hexutil.Big, error) {
	UseQuotas := s.b.Ethereum().GetFreeGasManager().GetAccountUseQuota(address)
	return (*hexutil.Big)(unsafe.Pointer(UseQuotas)), nil
}

func (s *FreeGasAPI) GetSurplus(ctx context.Context, address common.Address) (*hexutil.Big, error) {
	availableQuotas, err := s.b.Ethereum().GetFreeGasManager().GetAccountAvailableCredit(address, s.b.Ethereum().TxPool().State().GetBalance(address))
	return (*hexutil.Big)(unsafe.Pointer(availableQuotas)), err
}
