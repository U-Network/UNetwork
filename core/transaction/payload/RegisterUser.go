package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

const RegUserPrefix = "RegUserPrefix"

type RegisterUser struct {
	UserName        string
	UserProgramHash Uint160
	Reputation      Fixed64
}

func (p *RegisterUser) Data(version byte) []byte {
	return []byte{0}
}

func (p *RegisterUser) Serialize(w io.Writer, version byte) error {
	if err := serialization.WriteVarString(w, p.UserName); err != nil {
		return err
	}
	_, err := p.UserProgramHash.Serialize(w)
	if err != nil {
		return err
	}
	if err := p.Reputation.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (p *RegisterUser) Deserialize(r io.Reader, version byte) error {
	var err error
	p.UserName, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}

	if err := p.UserProgramHash.Deserialize(r); err != nil {
		return err
	}

	if err := p.Reputation.Deserialize(r); err != nil {
		return err
	}

	return nil
}

func (p *RegisterUser) ToString() string {
	return RegUserPrefix + p.UserName
}
