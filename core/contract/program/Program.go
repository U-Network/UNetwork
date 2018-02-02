package program

import (
	"UNetwork/common/serialization"
	. "UNetwork/errors"
	"io"
)

type Program struct {

	//the contract program code,which will be run on VM or specific envrionment
	Code []byte

	//the program code's parameter
	Parameter []byte
}

//Serialize the Program
func (p *Program) Serialize(w io.Writer) error {
	if err := serialization.WriteVarBytes(w, p.Parameter); err != nil {
		return NewDetailErr(err, ErrNoCode, "Execute Program Serialize Parameter failed.")
	}

	if err := serialization.WriteVarBytes(w, p.Code); err != nil {
		return NewDetailErr(err, ErrNoCode, "Execute Program Serialize Code failed.")
	}

	return nil
}

//Deserialize the Program
func (p *Program) Deserialize(w io.Reader) error {
	parameter, err := serialization.ReadVarBytes(w)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "Execute Program Deserialize Parameter failed.")
	}
	p.Parameter = parameter

	code, err := serialization.ReadVarBytes(w)
	if err != nil {
		return NewDetailErr(err, ErrNoCode, "Execute Program Deserialize Code failed.")
	}
	p.Code = code

	return nil
}
