package forum

import (
	"io"

	"UNetwork/common/serialization"
)

type LikeType byte

const (
	LikePost    LikeType = 0
	DislikePost LikeType = 1
)

type LikeInfo struct {
	Liker    string
	LikeType LikeType
}

func (p LikeInfo) Serialization(w io.Writer) error {
	if err := serialization.WriteVarString(w, p.Liker); err != nil {
		return err
	}
	if err := serialization.WriteByte(w, byte(p.LikeType)); err != nil {
		return err
	}

	return nil
}

func (p *LikeInfo) Deserialization(r io.Reader) error {
	var err error
	p.Liker, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}
	t, err := serialization.ReadByte(r)
	if err != nil {
		return err
	}
	p.LikeType = LikeType(t)

	return nil
}
