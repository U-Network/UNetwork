package core

import (
	"encoding/gob"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"bytes"
)

const (
	SecondsHour        = 60 * 60
	Second             = 1000 * 1000 * 1000
	ResetTime   uint64 = 12 * SecondsHour
	proportion         = 100
)

//Account is a free gas data structure for each account.
type Account struct {
	User      common.Address `json:"user"`
	UseAmount *big.Int       `json:"useAmount"`
	Timestamp *big.Int       `json:"timestamp"`
}

func (g *Account) DeepCopy(dst interface{}) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(g); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}

func (g *Account) CalculateUsedGas(bTime *big.Int) {
	if g.Timestamp.Uint64() / ResetTime != bTime.Uint64() / ResetTime {
		g.Timestamp = bTime
		g.UseAmount = new(big.Int).SetInt64(0)
		//g.Amount = new(big.Int).SetUint64((balance.Uint64() / 1e18) * proportion)
	}
}

func (g *Account) Marshal() (b []byte, err error) {
	return json.Marshal(g)
}

func (g *Account) UnMarshal(data []byte) (err error) {
	return json.Unmarshal(data, g)
}
