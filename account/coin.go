package account

import (
	"io"

	"UGCNetwork/common/serialization"
	"UGCNetwork/core/transaction"
)

type AddressType byte

const (
	SingleSign AddressType = 0
	MultiSign  AddressType = 1
)

type Coin struct {
	Output      *transaction.TxOutput
	AddressType AddressType
}

func (coin *Coin) Serialize(w io.Writer, version string) error {
	coin.Output.Serialize(w)
	switch {
	case version > "1.0.0":
		w.Write([]byte{byte(coin.AddressType)})
	default:
		break
	}
	return nil
}

func (coin *Coin) Deserialize(r io.Reader, version string) error {
	coin.Output = new(transaction.TxOutput)
	coin.Output.Deserialize(r)
	switch {
	case version > "1.0.0":
		addrType, err := serialization.ReadUint8(r)
		if err != nil {
			return err
		}
		coin.AddressType = AddressType(addrType)
	default:
		break
	}

	return nil
}
