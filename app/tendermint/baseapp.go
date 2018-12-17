package tendermint

import (
	"encoding/binary"
	ethbaseapp "github.com/U-Network/UNetwork/app/ethereum"

	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	"math/big"
)

var blockheight int64

type TendermintApplication struct {
	types.BaseApplication
	ethState *ethbaseapp.EthereumWorkState
}

func NewTendermintApplication() *TendermintApplication {
	return &TendermintApplication{}
}

func (app *TendermintApplication) SetEthState(state *ethbaseapp.EthereumWorkState) {
	app.ethState = state
}

func (app *TendermintApplication) Info(req types.RequestInfo) types.ResponseInfo {
	return types.ResponseInfo{Data: "nothing"}
}

func (app *TendermintApplication) SetOption(req types.RequestSetOption) types.ResponseSetOption {
	return types.ResponseSetOption{}
}

func (app *TendermintApplication) CheckTx(tx []byte) types.ResponseCheckTx {
	checkTx, err := app.ethState.DecodeTx(tx)
	if err != nil {
		return types.ResponseCheckTx{Code: code.CodeTypeEncodingError}
	}
	// unetwork check gas
	if checkTx.GasPrice().Cmp(big.NewInt(0)) == 0 {
		// Restore sender
		from, err := ethTypes.Sender(app.ethState.GetEthBackend().TxPool().GetSigner(), checkTx)
		if err != nil {
			return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized}
		}
		// Account contains the used gas
		account, _ := app.ethState.GetFreeGasManager().StateDB().GetAccount(from)
		// Free gas calculated after deducting the current token
		freeGas, _ := app.ethState.GetFreeGasManager().CalculateFreeGas(account, app.ethState.State.GetBalance(from))

		//fmt.Println("CheckTx freeGas: ", freeGas.String())

		if freeGas.Cmp(new(big.Int).SetUint64(checkTx.Gas())) < 0 { //Free free gas is available
			return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized}
		} else {
			account.UseAmount.Add(account.UseAmount, new(big.Int).SetUint64(checkTx.Gas()))
			app.ethState.GetFreeGasManager().StateDB().SetAccountUsedGas(account)

			//fmt.Println("CheckTx account.UseAmount : ", account.UseAmount.String())

			return types.ResponseCheckTx{Code: code.CodeTypeOK}
		}
	} else {
		return types.ResponseCheckTx{Code: code.CodeTypeOK}
	}
	return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized}
}

func (app *TendermintApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	err := app.ethState.DeliverTx(tx)
	if err != nil {
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError, Log: err.Error()}
	}
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

func (app *TendermintApplication) BeginBlock(params types.RequestBeginBlock) types.ResponseBeginBlock {
	// update latest block info
	blockheight = params.Header.Height
	tmHeader := params.GetHeader()
	err := app.ethState.BeginBlock(tmHeader.LastCommitHash, uint64(tmHeader.Time.Unix()), uint64(tmHeader.GetNumTxs()))
	if err != nil {

	}
	return types.ResponseBeginBlock{}
}

func (app *TendermintApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	return types.ResponseEndBlock{}
}

func (app *TendermintApplication) Commit() (resp types.ResponseCommit) {

	apphash := make([]byte, 32)
	binary.PutVarint(apphash, blockheight)
	//apphash[0] = blockHash[0]
	_, err := app.ethState.Commit(uint64(blockheight))
	if err != nil {
		return types.ResponseCommit{
			Data: apphash[:],
		}
	}
	return types.ResponseCommit{Data: apphash[:]}
}
