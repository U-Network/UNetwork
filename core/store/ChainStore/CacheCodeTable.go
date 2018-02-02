package ChainStore

import (
	"UNetwork/core/store"
	"UNetwork/smartcontract/states"
)

type CacheCodeTable struct {
	dbCache *DBCache
}

func NewCacheCodeTable(dbCache *DBCache) *CacheCodeTable {
	return &CacheCodeTable{
		dbCache: dbCache,
	}
}

func (table *CacheCodeTable) GetCode(codeHash []byte) ([]byte, error) {
	value, err := table.dbCache.TryGet(store.ST_Contract, string(codeHash))
	if err != nil {
		return nil, err
	}
	return value.(*states.ContractState).Code.Code, nil
}
