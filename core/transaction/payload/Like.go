package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

type LikeType byte

const (
	LikePost    LikeType = 0
	DislikePost LikeType = 1
)
//from(voter),articlehash,weight,votingPrice
type LikeArticle struct {
	Articlehash Uint256
	Liker       string
	Weight uint32
	Gasconsume Fixed64
	Extension string
}

func (p *LikeArticle) Data(version byte) []byte {
	return []byte{0}
}

func (p *LikeArticle) Serialize(w io.Writer, version byte) error {
	if _, err := p.Articlehash.Serialize(w); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Liker); err != nil {
		return err
	}
	if err := serialization.WriteUint32(w, p.Weight); err != nil {
		return err
	}
	if err := serialization.WriteUint64(w, uint64(p.Gasconsume)); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Extension); err != nil {
		return err
	}
	return nil
}

func (p *LikeArticle) Deserialize(r io.Reader, version byte) error {
	var err error
	if err = p.Articlehash.Deserialize(r); err != nil {
		return err
	}
	if p.Liker, err = serialization.ReadVarString(r); err != nil {
		return err
	}
	if p.Weight, err = serialization.ReadUint32(r); err != nil {
		return err
	}
	var gas uint64
	if gas, err = serialization.ReadUint64(r); err != nil {
		return err
	} else {
		p.Gasconsume = Fixed64(gas)
	}
	if p.Extension, err = serialization.ReadVarString(r); err != nil {
		if err.Error() == "EOF" {
			return nil
		} else {
			return err
		}
	}
	return nil
}
func (p *LikeArticle) Liketype() LikeType {
	if p.Gasconsume > 0 {
		return LikePost
	} else {
		return DislikePost
	}
}

func (p *LikeArticle) ToString() string {
	str := ""
	str += BytesToHexString(p.Articlehash.ToArray())
	str += p.Liker
	if p.Gasconsume > 0 {
		str += string(LikePost)
	} else {
		str += string(DislikePost)
	}
	return str
}
