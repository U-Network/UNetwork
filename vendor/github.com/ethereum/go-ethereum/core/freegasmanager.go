package core

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
)

var g_GasManager *FreeGasManager

// FreeGasManager
type FreeGasManager struct {
	DiskDb   db.DB
	State    *StateDB
	eth_backend *BlockChain
}

func NewFreeGasManager(ethBackend *BlockChain) *FreeGasManager {
	disk := db.NewDB("db_backend", db.GoLevelDBBackend, Homedir())
	return &FreeGasManager{
		DiskDb:   disk,
		State:    NewStateDB(disk, ethBackend),
		eth_backend: ethBackend,
	}
}

func GetGlobalGasManager()*FreeGasManager{
	return g_GasManager
}

func SetGlobalGasManager(manager *FreeGasManager){
	g_GasManager = manager
}
func (f *FreeGasManager) StateDB() *StateDB{
	return f.State
}
//CalculateFreeGas Calculate the free gas balance and return the gas balance
func (f *FreeGasManager) CalculateFreeGas(account *Account, balance *big.Int) (freeGas *big.Int, err error) {
	if account == nil || balance == nil {
		return nil, errors.New("CalculateFreeGas Invalid account or balance")
	}
	//return new(big.Int).SetUint64(((balance.Uint64() / 1e18) * proportion) - account.UseAmount.Uint64()), nil
	return new(big.Int).SetUint64(((balance.Uint64() / 1e18) * proportion)), nil
}

//IsExist Check if the account exists, if it exists, return true
func (f *FreeGasManager) IsExist(key common.Address) bool {
	return f.State.IsExist(key)
}

// Save function Will first copy the data in the StateDB and then write the data to disk.
func (f *FreeGasManager) Save() {
	batch := f.DiskDb.NewBatch()
	var duplication map[common.Address]*Account
	f.State.DeepCopy(&duplication)
	f.State.ReSetState()

	for k, v := range f.State.CurFreeGas {
		b, _ := v.Marshal()
		batch.Set(k[:], b)
	}
	batch.WriteSync()
}
