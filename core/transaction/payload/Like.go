package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
	"UNetwork/core/forum"
)

type LikeArticle struct {
	PostTxnHash Uint256
	Liker       string
	LikeType    forum.LikeType
}

func (p *LikeArticle) Data(version byte) []byte {
	return []byte{0}
}

func (p *LikeArticle) Serialize(w io.Writer, version byte) error {
	if _, err := p.PostTxnHash.Serialize(w); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Liker); err != nil {
		return err
	}
	if err := serialization.WriteByte(w, byte(p.LikeType)); err != nil {
		return err
	}

	return nil
}

func (p *LikeArticle) Deserialize(r io.Reader, version byte) error {
	var err error
	err = p.PostTxnHash.Deserialize(r)
	if err != nil {
		return err
	}
	p.Liker, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}
	t, err := serialization.ReadByte(r)
	if err != nil {
		return err
	}
	p.LikeType = forum.LikeType(t)

	return nil
}

func (p *LikeArticle) ToString() string {
	str := ""
	str += BytesToHexString(p.PostTxnHash.ToArray())
	str += p.Liker
	str += string(p.LikeType)

	return str
}
