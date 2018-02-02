package payload

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

type LockAsset struct {
	ProgramHash  Uint160
	AssetID      Uint256
	Amount       Fixed64
	UnlockHeight uint32
}

func (p *LockAsset) Data(version byte) []byte {
	return []byte{0}
}

func (p *LockAsset) Serialize(w io.Writer, version byte) error {
	_, err := p.ProgramHash.Serialize(w)
	if err != nil {
		return err
	}
	_, err = p.AssetID.Serialize(w)
	if err != nil {
		return err
	}
	if err := p.Amount.Serialize(w); err != nil {
		return err
	}
	if err := serialization.WriteUint32(w, p.UnlockHeight); err != nil {
		return err
	}

	return nil
}

func (p *LockAsset) Deserialize(r io.Reader, version byte) error {
	if err := p.ProgramHash.Deserialize(r); err != nil {
		return err
	}
	if err := p.AssetID.Deserialize(r); err != nil {
		return err
	}
	if err := p.Amount.Deserialize(r); err != nil {
		return err
	}
	height, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	p.UnlockHeight = height

	return nil
}

func (p *LockAsset) ToString() string {
	str := ""
	str += BytesToHexString(p.ProgramHash.ToArray())
	str += BytesToHexString(p.AssetID.ToArray())

	return str
}
