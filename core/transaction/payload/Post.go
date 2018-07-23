package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

type ContentType byte
const (
	PostArticlePrefix = "PostArticlePrefix"
)

const (
	Post    ContentType = 0x00
	Reply   ContentType = 0x01
	Reviset ContentType = 0x02
)

type ArticleInfo struct {
	ContentType   ContentType
	Articlehash Uint256
	Author string
	Body []byte
	Parent_author string
	Parent_articlehash Uint256
	Title string
	Json_metadata string
	Extension string
}

func (p *ArticleInfo) Data(version byte) []byte {
	return []byte{0}
}

func (p *ArticleInfo) Serialize(w io.Writer, version byte) error {
	if err := serialization.WriteByte(w, byte(p.ContentType)); err != nil {
		return err
	}
	if _, err := p.Articlehash.Serialize(w); err != nil {
		return err
	}

	if err := serialization.WriteVarString(w, p.Author); err != nil {
		return err
	}

	if err := serialization.WriteVarBytes(w, p.Body); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Parent_author); err != nil {
		return err
	}
	if _, err := p.Parent_articlehash.Serialize(w); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Title); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Json_metadata); err != nil {
		return err
	}
	if err := serialization.WriteVarString(w, p.Extension); err != nil {
		return err
	}
	return nil
}

func (p *ArticleInfo) Deserialize(r io.Reader, version byte) error {
	var err error
	if contentType, err := serialization.ReadByte(r); err != nil {
		return err
	} else {
		p.ContentType = ContentType(contentType)
	}

	if err = p.Articlehash.Deserialize(r); err != nil {
		return nil
	}
	if author, err := serialization.ReadVarString(r); err != nil {
		return err
	} else {
		p.Author = author
	}


	if body, err := serialization.ReadVarBytes(r); err != nil {
		return err
	} else {
		p.Body = body
	}

	if parent_author, err := serialization.ReadVarString(r); err != nil {
		return err
	} else {
		p.Parent_author = parent_author
	}

	if err = p.Parent_articlehash.Deserialize(r); err != nil {
		return err
	}

	if title, err := serialization.ReadVarString(r); err != nil {
		return err
	} else {
		p.Title = title
	}

	if json_metadata, err := serialization.ReadVarString(r); err != nil {
		return err
	} else {
		p.Json_metadata = json_metadata
	}

	if extension, err := serialization.ReadVarString(r); err != nil {
		if err.Error() == "EOF" {
			return nil
		} else {
			return err
		}
	} else {
		p.Extension = extension
	}

	return nil
}

func (p *ArticleInfo) ToString() string {
	return PostArticlePrefix + p.Author
}
