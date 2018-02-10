package payload

import (
	"io"

	"UNetwork/common/serialization"
)

const (
	WithdrawPrefix = "WithdrawPrefix"
)

type Withdrawal struct {
	Payee string
}

func (p *Withdrawal) Data(version byte) []byte {
	return []byte{0}
}

func (p *Withdrawal) Serialize(w io.Writer, version byte) error {
	if err := serialization.WriteVarString(w, p.Payee); err != nil {
		return err
	}

	return nil
}

func (p *Withdrawal) Deserialize(r io.Reader, version byte) error {
	var err error
	p.Payee, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}

	return nil
}

func (p *Withdrawal) ToString() string {
	return WithdrawPrefix + p.Payee
}
