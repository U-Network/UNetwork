package forum

import (
	"io"

	. "UNetwork/common"
)

type UserInfo struct {
	UserProgramHash Uint160
	Reputation      Fixed64
}

func (p UserInfo) Serialization(w io.Writer) error {
	if _, err := p.UserProgramHash.Serialize(w); err != nil {
		return err
	}
	if err := p.Reputation.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (p *UserInfo) Deserialization(r io.Reader) error {
	if err := p.UserProgramHash.Deserialize(r); err != nil {
		return err
	}
	if err := p.Reputation.Deserialize(r); err != nil {

	}

	return nil
}
