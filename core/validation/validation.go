package validation

import (
	"UNetwork/common"
	. "UNetwork/common"
	sig "UNetwork/core/signature"
	"UNetwork/crypto"
	. "UNetwork/errors"
	"UNetwork/vm/avm"
	"UNetwork/vm/avm/interfaces"
)

func VerifySignableData(signableData sig.SignableData) (bool, error) {

	hashes, err := signableData.GetProgramHashes()
	if err != nil {
		return false, err
	}

	programs := signableData.GetPrograms()
	Length := len(hashes)
	if Length != len(programs) {
		return false, NewErr("The number of data hashes is different with number of programs.")
	}

	programs = signableData.GetPrograms()
	for i := 0; i < len(programs); i++ {
		temp, _ := ToCodeHash(programs[i].Code)
		if hashes[i] != temp {
			return false, NewErr("The data hashes is different with corresponding program code.")
		}
		//execute program on VM
		var cryptos interfaces.ICrypto
		cryptos = new(avm.ECDsaCrypto)
		se := avm.NewExecutionEngine(signableData, cryptos, nil, nil, common.Fixed64(0))
		se.LoadCode(programs[i].Code, false)
		se.LoadCode(programs[i].Parameter, true)
		err := se.Execute()

		if err != nil {
			return false, NewDetailErr(err, ErrNoCode, "")
		}

		if se.GetState() != avm.HALT {
			return false, NewDetailErr(NewErr("[VM] Finish State not equal to HALT."), ErrNoCode, "")
		}

		if se.GetEvaluationStack().Count() != 1 {
			return false, NewDetailErr(NewErr("[VM] Execute Engine Stack Count Error."), ErrNoCode, "")
		}

		flag := se.GetExecuteResult()
		if !flag {
			return false, NewDetailErr(NewErr("[VM] Check Sig FALSE."), ErrNoCode, "")
		}
	}

	return true, nil
}

func VerifySignature(signableData sig.SignableData, pubkey *crypto.PubKey, signature []byte) (bool, error) {
	err := crypto.Verify(*pubkey, sig.GetHashData(signableData), signature)
	if err != nil {
		return false, NewDetailErr(err, ErrNoCode, "[Validation], VerifySignature failed.")
	} else {
		return true, nil
	}
}
