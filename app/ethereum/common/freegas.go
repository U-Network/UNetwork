package common

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
	"time"
)

type FreeGas struct {
	User      common.Address `json:"user"`
	Amount    *big.Int       `json:"amount"`
	UseAmount *big.Int       `json:"useAmount"`
	Timestamp *time.Time     `json:"timestamp"`
}

func (g *FreeGas) Marshal() (b []byte, err error) {
	return json.Marshal(g)
}

func (g *FreeGas) UnMarshal(data []byte) (err error) {
	return json.Unmarshal(data, g)
}

type FreeGasManager struct {
	DiskDb db.DB
	State  *StateDB
}

func NewFreeGasManager() *FreeGasManager {
	return &FreeGasManager{
		DiskDb: db.NewDB("db_backend", db.GoLevelDBBackend, Homedir()),
		State:  NewStateDB(),
	}
}

func (f *FreeGasManager) IsExist(key []byte) bool {
	return f.DiskDb.Has(key)
}

func (f *FreeGasManager) AddAccount(addr []byte) (err error) {
	if f.DiskDb.Has(addr) {
		by := f.DiskDb.Get(addr)
		if by == nil {
			return errors.New("Failed to get gas quota from disk")
		}
		var account *FreeGas = new(FreeGas)
		if err = account.UnMarshal(by); err != nil {
			return err
		}
		f.State.Append(account)

	} else {

	}
}

func (f *FreeGasManager) Save() {
	batch := f.DiskDb.NewBatch()
	var duplication map[common.Address]*FreeGas
	f.State.DeepCopy(&duplication)
	f.State.ReSet()

	for k, v := range f.State.CurFreeGas {
		b, _ := v.Marshal()
		batch.Set(k[:], b)
	}
	batch.WriteSync()
}
