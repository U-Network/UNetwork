package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

const (
	PostArticlePrefix = "PostArticlePrefix"
)

type PostArticle struct {
	ContentHash Uint256
	Author      string
}

func (p *PostArticle) Data(version byte) []byte {
	return []byte{0}
}

func (p *PostArticle) Serialize(w io.Writer, version byte) error {
	if _, err := p.ContentHash.Serialize(w); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Author); err != nil {
		return err
	}

	return nil
}

func (p *PostArticle) Deserialize(r io.Reader, version byte) error {
	var err error
	err = p.ContentHash.Deserialize(r)
	if err != nil {
		return err
	}
	p.Author, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}

	return nil
}

func (p *PostArticle) ToString() string {
	return PostArticlePrefix + p.Author
}
