package core

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
	"fmt"
	"time"
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
	//fmt.Println("CalculateFreeGas balance: ", balance.String())
	if account == nil || balance == nil {
		return nil, errors.New("CalculateFreeGas Invalid account or balance")
	}

	if balance.Cmp(new(big.Int).SetInt64(int64(0)))  <= 0 {
		return big.NewInt(0),nil
	}

	account.CalculateUsedGas(new(big.Int).SetInt64(time.Now().UTC().Unix()))

	token := new(big.Int).Div(balance, new(big.Int).SetUint64(1e18))
	gas := new(big.Int).Mul(token, new(big.Int).SetUint64(proportion))
	available := new(big.Int).Sub(gas,account.UseAmount)
	if new(big.Int).Set(available).Cmp(big.NewInt(0)) <= 0{
		return big.NewInt(0), nil
	}
	return  available,nil
}

//IsExist Check if the account exists, if it exists, return true
func (f *FreeGasManager) IsExist(key common.Address) bool {
	return f.State.IsExist(key)
}

// Save function Will first copy the data in the StateDB and then write the data to disk.
func (f *FreeGasManager) Save()  {
	batch := f.DiskDb.NewBatch()
	//var duplication map[common.Address]*Account = make(map[common.Address]*Account)
	//f.State.DeepCopy(&duplication)

	f.State.Mux.RLock()
	defer f.State.Mux.RUnlock()
	for k, v := range f.State.CurFreeGas {
		b, _ := v.Marshal()
		batch.Set(k[:], b)
	}
	batch.WriteSync()

}

// GetAccountUseQuota Get the amount the user has used based on the address
func (f *FreeGasManager) GetAccountUseQuota(addr common.Address) *big.Int {
	cur,_:= f.State.GetAccount(addr)
	//fmt.Println("cur.UseAmount", cur.UseAmount.String())
	return cur.UseAmount
}

//GetAccountAvailableCredit Get the quota available for the user's current time interval based on the address
func (f *FreeGasManager)GetAccountAvailableCredit(addr common.Address,  balance *big.Int)  (freeGas *big.Int, err error){
	//fmt.Println("addr: ", common.Bytes2Hex(addr[:]))
	cur, _ := f.StateDB().GetAccount(addr)

	//fmt.Println("cur.UseAmount: ", cur.UseAmount.String())
	return f.CalculateFreeGas(cur,balance)
}

func (f *FreeGasManager)Close() {
	f.Save()
	f.StateDB().ReSetState()
	f.DiskDb.Close()
}