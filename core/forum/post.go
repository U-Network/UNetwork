package forum

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

type ContentType byte

const (
	Post    ContentType = 0x00
	Reply   ContentType = 0x01
	Reviset ContentType = 0x02
)

type ArticleInfo struct {
	ContentType   ContentType
	ContentHash   Uint256
	ParentTxnHash Uint256
}

func (p ArticleInfo) Serialization(w io.Writer) error {
	if err := serialization.WriteByte(w, byte(p.ContentType)); err != nil {
		return err
	}
	if _, err := p.ContentHash.Serialize(w); err != nil {
		return err
	}
	if p.ContentType != Post {
		if _, err := p.ParentTxnHash.Serialize(w); err != nil {
			return err
		}
	}

	return nil
}

func (p *ArticleInfo) Deserialization(r io.Reader) error {
	var err error
	contentType, err := serialization.ReadByte(r)
	if err != nil {
		return err
	}
	p.ContentType = ContentType(contentType)
	err = p.ContentHash.Deserialize(r)
	if err != nil {
		return err
	}
	if p.ContentType != Post {
		err = p.ParentTxnHash.Deserialize(r)
		if err != nil {
			return err
		}
	}

	return nil
}
