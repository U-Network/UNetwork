package tendermint

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	ethbaseapp "github.com/U-Network/UNetwork/app/ethereum"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
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
	return types.ResponseCheckTx{Code: code.CodeTypeOK}
}

func (app *TendermintApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	//Unetwork test start
	//target: to debug the trouble
	fmt.Println("DeliverTx ===================this is shi=========================")
	var des []byte = make([]byte, 4096)
	base64.StdEncoding.Encode(des, tx)
	fmt.Println("this is tendermint callback function DeliverTxAsync txdata:", string(des))
	//end test stop

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
