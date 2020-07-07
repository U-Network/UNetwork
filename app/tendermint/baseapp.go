package tendermint

import (
	"encoding/binary"
	ethbaseapp "github.com/U-Network/UNetwork/app/ethereum"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/tendermint/tendermint/abci/example/code"
	"github.com/tendermint/tendermint/abci/types"
	cfg "github.com/tendermint/tendermint/config"
	tendTypes "github.com/tendermint/tendermint/types"
	"log"
	"math/big"
	"os"
)

var (
	blockheight int64
	fileSize    int64
)

const updateConfigInterval = 100

type TendermintApplication struct {
	types.BaseApplication
	ethState         *ethbaseapp.EthereumWorkState
	onReplaying      bool
	validator        map[*tendTypes.GenesisValidator]struct{}
	config           *cfg.Config
	updateConfigFlag int64
}

func NewTendermintApplication(cfg *cfg.Config, state *ethbaseapp.EthereumWorkState) *TendermintApplication {
	return &TendermintApplication{
		config:           cfg,
		ethState:         state,
		updateConfigFlag: -1,
	}
}

func (app *TendermintApplication) InitChain(req types.RequestInitChain) types.ResponseInitChain {
	return types.ResponseInitChain{}
}

func (app *TendermintApplication) Info(req types.RequestInfo) types.ResponseInfo {
	//return types.ResponseInfo{Data: "nothing"}
	//fmt.Println(req)
	blockheight := int64(app.ethState.GetblockNumber())
	apphash := make([]byte, 32)
	binary.PutVarint(apphash, blockheight)
	return types.ResponseInfo{
		LastBlockHeight:  blockheight,
		LastBlockAppHash: apphash,
	}
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
		account, _ := app.ethState.GetFreeGasManager().State.GetAccount(from)
		// Free gas calculated after deducting the current token
		freeGas, _ := app.ethState.GetFreeGasManager().CalculateFreeGas(account, app.ethState.State.GetBalance(from))

		if freeGas.Cmp(new(big.Int).SetUint64(checkTx.Gas())) < 0 { //Free free gas is available
			return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized}
		} else {
			account.UseAmount.Add(account.UseAmount, new(big.Int).SetUint64(checkTx.Gas()))
			app.ethState.GetFreeGasManager().State.SetAccountUsedGas(account)

			return types.ResponseCheckTx{Code: code.CodeTypeOK}
		}
	} else {
		return types.ResponseCheckTx{Code: code.CodeTypeOK}
	}
	return types.ResponseCheckTx{Code: code.CodeTypeUnauthorized}
}

func (app *TendermintApplication) BeginBlock(params types.RequestBeginBlock) types.ResponseBeginBlock {
	// update latest block info
	blockheight = params.Header.Height
	app.onReplaying = app.ethState.GetblockNumber() >= uint64(blockheight)
	if app.onReplaying {
		return types.ResponseBeginBlock{}
	}
	// start new block
	tmHeader := params.GetHeader()
	err := app.ethState.BeginBlock(tmHeader.LastCommitHash, uint64(tmHeader.Time.Unix()), uint64(tmHeader.GetNumTxs()))
	if err != nil {

	}
	return types.ResponseBeginBlock{}
}

func (app *TendermintApplication) DeliverTx(tx []byte) types.ResponseDeliverTx {
	if app.onReplaying {
		return types.ResponseDeliverTx{Code: code.CodeTypeOK}
	}
	err := app.ethState.DeliverTx(tx)
	if err != nil {
		return types.ResponseDeliverTx{Code: code.CodeTypeEncodingError, Log: err.Error()}
	}
	return types.ResponseDeliverTx{Code: code.CodeTypeOK}
}

func (app *TendermintApplication) EndBlock(req types.RequestEndBlock) types.ResponseEndBlock {
	if app.onReplaying {
		return types.ResponseEndBlock{}
	}
	if app.updateConfigFlag < 0 && app.validator == nil {
		//read config
		file := app.config.GenesisFile()
		fileinfo, er := os.Stat(file)
		if er != nil {
			log.Println("EndBlock error : ", er)
			return types.ResponseEndBlock{}
		}
		genesisDoc, err := tendTypes.GenesisDocFromFile(app.config.GenesisFile())
		if err != nil {
			log.Println("EndBlock error : ", err)
			return types.ResponseEndBlock{}
		}
		app.validator = make(map[*tendTypes.GenesisValidator]struct{})
		for _, val := range genesisDoc.Validators {
			app.validator[&val] = struct{}{}
		}
		app.updateConfigFlag = updateConfigInterval
		fileSize = fileinfo.Size()
		return types.ResponseEndBlock{}
	} else if app.updateConfigFlag == 0 {
		//Update and verify config
		fileinfo, err := os.Stat(app.config.GenesisFile())
		if err != nil && os.IsNotExist(err) {
			log.Println("EndBlock error : ", err)
			return types.ResponseEndBlock{}
		}
		app.updateConfigFlag = updateConfigInterval
		if fileinfo.Size() == fileSize {
			return types.ResponseEndBlock{}
		}
		genesisDoc, err := tendTypes.GenesisDocFromFile(app.config.GenesisFile())
		if err != nil {
			log.Println("EndBlock error : ", err)
			return types.ResponseEndBlock{}
		}
		app.validator = make(map[*tendTypes.GenesisValidator]struct{})
		var validatorUpdates []types.ValidatorUpdate = make([]types.ValidatorUpdate, 0)
		for _, val := range genesisDoc.Validators {
			app.validator[&val] = struct{}{}

			var pub types.PubKey
			pub.Data = val.PubKey.Bytes()
			pub.Type = types.PubKeyEd25519
			validator := types.ValidatorUpdate{
				PubKey: pub,
				Power:  val.Power,
			}
			validatorUpdates = append(validatorUpdates, validator)
		}
		fileSize = fileinfo.Size()
		return types.ResponseEndBlock{ValidatorUpdates: validatorUpdates}
	}
	app.updateConfigFlag--

	// call EndBlock
	app.ethState.EndBlock(uint64(req.Height))

	return types.ResponseEndBlock{}
}

func (app *TendermintApplication) Commit() (resp types.ResponseCommit) {
	apphash := make([]byte, 32)
	binary.PutVarint(apphash, blockheight)
	if app.onReplaying {
		return types.ResponseCommit{Data: apphash[:]}
	}
	_, err := app.ethState.Commit(uint64(blockheight))
	if err != nil {
		return types.ResponseCommit{
			Data: apphash[:],
		}
	}
	return types.ResponseCommit{Data: apphash[:]}
}
