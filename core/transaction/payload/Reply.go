package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

const (
	ReplyArticlePrefix = "ReplyArticlePrefix"
)

type ReplyArticle struct {
	PostHash    Uint256
	ContentHash Uint256
	Replier     string
}

func (p *ReplyArticle) Data(version byte) []byte {
	return []byte{0}
}

func (p *ReplyArticle) Serialize(w io.Writer, version byte) error {
	if _, err := p.PostHash.Serialize(w); err != nil {
		return err
	}
	if _, err := p.ContentHash.Serialize(w); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Replier); err != nil {
		return err
	}

	return nil
}

func (p *ReplyArticle) Deserialize(r io.Reader, version byte) error {
	var err error
	err = p.PostHash.Deserialize(r)
	if err != nil {
		return err
	}
	err = p.ContentHash.Deserialize(r)
	if err != nil {
		return err
	}
	p.Replier, err = serialization.ReadVarString(r)
	if err != nil {
		return err
	}

	return nil
}

func (p *ReplyArticle) ToString() string {
	return ReplyArticlePrefix + p.Replier
}
