package forum

import (
	"io"

	. "UNetwork/common"
)

type TokenType int

const (
	TotalToken     TokenType = 0
	WithdrawnToken TokenType = 1
)

type TokenInfo struct {
	Number Fixed64
}

func (p TokenInfo) Serialization(w io.Writer) error {
	if err := p.Number.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (p *TokenInfo) Deserialization(r io.Reader) error {
	if err := p.Number.Deserialize(r); err != nil {
		return err
	}

	return nil
}
