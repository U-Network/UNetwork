package core

import (
	"bytes"
	"encoding/gob"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/db"
	"math/big"
	"sync"
	"time"
)

type StateDB struct {
	CurFreeGas  map[common.Address]*Account
	DiskDb      db.DB
	eth_backend *BlockChain
	Mux         *sync.RWMutex
}

func NewStateDB(diskb db.DB, chain *BlockChain) *StateDB {
	return &StateDB{
		CurFreeGas:  make(map[common.Address]*Account),
		DiskDb:      diskb,
		eth_backend: chain,
		Mux:         new(sync.RWMutex),
	}
}

// SetAccountUsedGas Put the user into the state pool, put in the time before the time and the amount of the check
func (s *StateDB) SetAccountUsedGas(account *Account) (err error) {
	if account == nil {
		return errors.New("Invalid Account memory")
	}
	if account.UseAmount.Int64() <= 0 || account.Timestamp.Int64() <= 0 {
		return errors.New("Invalid time or invalid gas quota")
	}
	s.Mux.Lock()
	defer s.Mux.Unlock()
	cur, _ := s.GetAccount(account.User)
	cur.Timestamp.Set(account.Timestamp)
	cur.UseAmount.Set(account.UseAmount)
	s.CurFreeGas[cur.User] = cur
	return nil
}

// GetAccount Check if the user needs to update the used gas interval, if it is updating the used gas,
// and return a deep copy of the user's free gas quota data structure.
func (s *StateDB) GetAccount(addr common.Address) (account *Account, err error) {
	s.Mux.Lock()
	defer s.Mux.Unlock()
	CurAccount, ok := s.CurFreeGas[addr]
	if ok {
		account = new(Account)
		err = CurAccount.DeepCopy(account)
		return account, err
	}
	return s.getAccount(addr)
}

//
func (s *StateDB) getAccount(addr common.Address) (account *Account, err error) {
	//Check if the account exists on the disk
	b := s.DiskDb.Has(addr[:])
	if b {
		by := s.DiskDb.Get(addr[:])
		var CurFreeGas *Account = new(Account)
		err = CurFreeGas.UnMarshal(by)
		if err != nil {
			return nil, errors.New("An error occurred while using the byte array UnMarshal account errcode:" + err.Error())
		}
		//s.eth_backend.TxPool().State().GetBalance(addr)

		CurFreeGas.CalculateUsedGas(s.eth_backend.CurrentBlock().Header().Time)
		err = CurFreeGas.DeepCopy(account)
		return account, err
	}
	//The account appears for the first time
	var curAccount *Account = new(Account)
	curAccount.User = addr
	curAccount.UseAmount = new(big.Int)
	curAccount.Timestamp = new(big.Int).SetInt64(time.Now().Unix())
	s.CurFreeGas[addr] = curAccount
	account = new(Account)
	err = curAccount.DeepCopy(account)
	return account, err
}

// IsExist Check if the account exists, return true if it exists
func (s *StateDB) IsExist(key common.Address) bool {
	s.Mux.RLock()
	defer s.Mux.RUnlock()
	if _, ok := s.CurFreeGas[key]; ok {
		return ok
	}
	return s.DiskDb.Has(key[:])
}

// ReSetState Reset all accounts in StateDB
func (s *StateDB) ReSetState() {
	s.Mux.Lock()
	defer s.Mux.Unlock()
	s.CurFreeGas = make(map[common.Address]*Account)
}

// DeepCopy Deep copy StateDB and return a copy of StateDB
func (s *StateDB) DeepCopy(dst interface{}) error {
	var buf bytes.Buffer
	s.Mux.RLock()
	defer s.Mux.RUnlock()
	if err := gob.NewEncoder(&buf).Encode(s); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

// IsEmpty Check if the account in StateDB is empty, return true if it is empty
func (s *StateDB) IsEmpty() bool {
	s.Mux.RLock()
	defer s.Mux.RUnlock()
	return len(s.CurFreeGas) == 0
}
