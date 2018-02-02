package test_case

import (
	"UNetwork/core/ledger"
	"UNetwork/crypto"
	"UNetwork/core/store/ChainStore"
	client "UNetwork/account"
	"UNetwork/vm/evm"
	"strings"
	"UNetwork/vm/evm/abi"
	"UNetwork/common"
	"time"
	"math/big"
)

func NewEngine(ABI, BIN string, params ...interface{}) (*common.Uint160, *common.Uint160, *evm.ExecutionEngine, *abi.ABI, error) {
	ledger.DefaultLedger = new(ledger.Ledger)
	ledger.DefaultLedger.Store = ChainStore.NewLedgerStore()
	ledger.DefaultLedger.Store.InitLedgerStore(ledger.DefaultLedger)
	crypto.SetAlg(crypto.P256R1)
	account, _ := client.NewAccount()
	t := time.Now().Unix()
	e := evm.NewExecutionEngine(nil, big.NewInt(t), big.NewInt(1), common.Fixed64(0))
	parsed, err := abi.JSON(strings.NewReader(ABI))
	if err != nil { return nil, nil, nil, nil, err }
	input, err := parsed.Pack("", params...)
	if err != nil { return nil, nil, nil, nil, err }
	code := common.FromHex(BIN)
	codes := append(code, input...)
	codeHash, _ := common.ToCodeHash(codes)
	_, err = e.Create(account.ProgramHash, codes)
	if err != nil { return nil, nil, nil, nil, err }
	return &codeHash, &account.ProgramHash, e, &parsed,  nil
}



