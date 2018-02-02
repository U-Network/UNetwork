package asset

import (
	"io"

	. "UNetwork/common"
	"UNetwork/common/serialization"
)

type LockAsset struct {
	Lock   uint32
	Unlock uint32
	Amount Fixed64
}

func (a *LockAsset) Serialize(w io.Writer) error {
	if err := serialization.WriteUint32(w, a.Lock); err != nil {
		return err
	}
	if err := serialization.WriteUint32(w, a.Unlock); err != nil {
		return err
	}
	if err := a.Amount.Serialize(w); err != nil {
		return err
	}

	return nil
}

func (a *LockAsset) Deserialize(r io.Reader) error {
	startHeight, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	a.Lock = startHeight

	endHeight, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	a.Unlock = endHeight

	a.Amount = *new(Fixed64)
	if err := a.Amount.Deserialize(r); err != nil {
		return err
	}

	return nil
}
