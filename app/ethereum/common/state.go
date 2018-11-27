package common

import (
	"bytes"
	"encoding/gob"
	"github.com/ethereum/go-ethereum/common"
	"sync"
)

type StateDB struct {
	CurFreeGas map[common.Address]*FreeGas
	Mux        *sync.RWMutex
}

func NewStateDB() *StateDB {
	return &StateDB{
		CurFreeGas: make(map[common.Address]*FreeGas),
		Mux:        new(sync.RWMutex),
	}
}

func (s *StateDB) Append(addr common.Address, account *FreeGas) {

	s.Mux.Lock()
	defer s.Mux.Unlock()
	
}

func (s *StateDB) ReSet() {
	s.Mux.Lock()
	defer s.Mux.Unlock()
	s.CurFreeGas = make(map[common.Address]*FreeGas)
}

func (s *StateDB) DeepCopy(dst interface{}) error {
	var buf bytes.Buffer
	s.Mux.RLock()
	defer s.Mux.RUnlock()
	if err := gob.NewEncoder(&buf).Encode(s); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func (s *StateDB) IsEmpty() bool {
	s.Mux.RLock()
	defer s.Mux.RUnlock()
	return len(s.CurFreeGas) == 0
}
