package httpjsonrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"UNetwork/account"
	. "UNetwork/common"
	"UNetwork/common/config"
	"UNetwork/common/log"
	"UNetwork/core/ledger"
	"UNetwork/core/signature"
	tx "UNetwork/core/transaction"
	. "UNetwork/errors"
	"UNetwork/sdk"

	"github.com/mitchellh/go-homedir"
)

const (
	RANDBYTELEN = 4
)
func issueAsset(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var asset, value, address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("open wallet first")
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UNetworkRPC("invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UNetworkRPC("invalid asset hash")
	}
	issueTxn, err := sdk.MakeIssueTransaction(Wallet, assetID, address, value)
	if err != nil {
		return UNetworkRPCInternalError
	}

	if errCode := VerifyAndSendTx(issueTxn); errCode != ErrNoError {
		return UNetworkRPCInvalidTransaction
	}
	txHash := issueTxn.Hash()
	return UNetworkRPC(BytesToHexString(txHash.ToArrayReverse()))
}

func regAsset(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UNetworkRPCNil
	}
	var name, value string
	switch params[0].(type) {
	case string:
		name = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("error : wallet is not opened")
	}
	txn, err := sdk.MakeRegTransaction(Wallet, name, value)

	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC("error: " + errCode.Error())
	}
	txHash := txn.Hash()
	return UNetworkRPC(BytesToHexString(txHash.ToArrayReverse()))
}
func getUtxoCoins(params []interface{}) map[string]interface{} {
	type CoinInfo struct {
		ReferTxID string
		ReferTxOutputIndex uint16
		AssetID     string
		Value       Fixed64
		ProgramHash string
	}
	var results []CoinInfo

	if len(params) < 1 {
		coins, err := Wallet.GetCoins()
		if (err != nil) {
			return UNetworkRPC(err.Error())
		}
		for k, coin := range coins {
			ReferTxIDstr := BytesToHexString(k.ReferTxID.ToArrayReverse())
			AssetIDstr := BytesToHexString(coin.Output.AssetID.ToArrayReverse())
			ProgramHashstr,_:= coin.Output.ProgramHash.ToAddress()
			results = append(results, CoinInfo{ReferTxIDstr, k.ReferTxOutputIndex,
				AssetIDstr, coin.Output.Value, ProgramHashstr})
		}
	}else {
		addr := params[0].(string)
		var programHash Uint160
		programHash, err := ToScriptHash(addr)
		if err != nil {
			return UNetworkRPC("Address Wrong!")
		}
		unspends, err := ledger.DefaultLedger.Store.GetUnspentOutputFromProgramHash(programHash)

		for k, coin := range unspends {
			ReferTxIDstr := BytesToHexString(k.ReferTxID.ToArrayReverse())
			AssetIDstr := BytesToHexString(coin.AssetID.ToArrayReverse())
			ProgramHashstr,_ := coin.ProgramHash.ToAddress()
			results = append(results, CoinInfo{ReferTxIDstr, k.ReferTxOutputIndex,
				AssetIDstr, coin.Value, ProgramHashstr})
		}
	}
	return UNetworkRPC(results)
}
func getUtxoByAddr(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	addr := params[0].(string)

	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		return UNetworkRPC("Address Wrong!")
	}

	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	type Result struct {
		AssetId   string
		AssetName string
		Utxo      []UTXOUnspentInfo
	}
	var results []Result
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)

	for k, u := range unspends {
		assetid := BytesToHexString(k.ToArrayReverse())
		asset, err := ledger.DefaultLedger.Store.GetAsset(k)
		if err != nil {
			return UNetworkRPC("INTERNAL_ERROR!")
		}
		var unspendsInfo []UTXOUnspentInfo
		for _, v := range u {
			unspendsInfo = append(unspendsInfo, UTXOUnspentInfo{BytesToHexString(v.Txid.ToArrayReverse()), v.Index, v.Value.String()})
		}
		results = append(results, Result{assetid, asset.Name, unspendsInfo})
	}

	return UNetworkRPC(results)
}

func TransArryByteToHexString(ptx *tx.Transaction) *Transactions {

	trans := new(Transactions)
	trans.TxType = ptx.TxType
	trans.PayloadVersion = ptx.PayloadVersion
	trans.Payload = TransPayloadToHex(ptx.Payload)

	n := 0
	trans.Attributes = make([]TxAttributeInfo, len(ptx.Attributes))
	for _, v := range ptx.Attributes {
		trans.Attributes[n].Usage = v.Usage
		trans.Attributes[n].Data = BytesToHexString(v.Data)
		n++
	}

	n = 0
	trans.UTXOInputs = make([]UTXOTxInputInfo, len(ptx.UTXOInputs))
	for _, v := range ptx.UTXOInputs {
		trans.UTXOInputs[n].ReferTxID = BytesToHexString(v.ReferTxID.ToArrayReverse())
		trans.UTXOInputs[n].ReferTxOutputIndex = v.ReferTxOutputIndex
		n++
	}

	n = 0
	trans.BalanceInputs = make([]BalanceTxInputInfo, len(ptx.BalanceInputs))
	for _, v := range ptx.BalanceInputs {
		trans.BalanceInputs[n].AssetID = BytesToHexString(v.AssetID.ToArrayReverse())
		trans.BalanceInputs[n].Value = v.Value.String()
		trans.BalanceInputs[n].ProgramHash = BytesToHexString(v.ProgramHash.ToArrayReverse())
		n++
	}

	n = 0
	trans.Outputs = make([]TxoutputInfo, len(ptx.Outputs))
	for _, v := range ptx.Outputs {
		trans.Outputs[n].AssetID = BytesToHexString(v.AssetID.ToArrayReverse())
		trans.Outputs[n].Value = v.Value.String()
		address, _ := v.ProgramHash.ToAddress()
		trans.Outputs[n].Address = address
		n++
	}

	n = 0
	trans.Programs = make([]ProgramInfo, len(ptx.Programs))
	for _, v := range ptx.Programs {
		trans.Programs[n].Code = BytesToHexString(v.Code)
		trans.Programs[n].Parameter = BytesToHexString(v.Parameter)
		n++
	}

	n = 0
	trans.AssetOutputs = make([]TxoutputMap, len(ptx.AssetOutputs))
	for k, v := range ptx.AssetOutputs {
		trans.AssetOutputs[n].Key = k
		trans.AssetOutputs[n].Txout = make([]TxoutputInfo, len(v))
		for m := 0; m < len(v); m++ {
			trans.AssetOutputs[n].Txout[m].AssetID = BytesToHexString(v[m].AssetID.ToArrayReverse())
			trans.AssetOutputs[n].Txout[m].Value = v[m].Value.String()
			address, _ := v[m].ProgramHash.ToAddress()
			trans.AssetOutputs[n].Txout[m].Address = address
		}
		n += 1
	}

	n = 0
	trans.AssetInputAmount = make([]AmountMap, len(ptx.AssetInputAmount))
	for k, v := range ptx.AssetInputAmount {
		trans.AssetInputAmount[n].Key = k
		trans.AssetInputAmount[n].Value = v
		n += 1
	}

	n = 0
	trans.AssetOutputAmount = make([]AmountMap, len(ptx.AssetOutputAmount))
	for k, v := range ptx.AssetOutputAmount {
		trans.AssetInputAmount[n].Key = k
		trans.AssetInputAmount[n].Value = v
		n += 1
	}

	mHash := ptx.Hash()
	trans.Hash = BytesToHexString(mHash.ToArrayReverse())

	return trans
}
func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}
func getBestBlockHash(params []interface{}) map[string]interface{} {
	hash := ledger.DefaultLedger.Blockchain.CurrentBlockHash()
	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}
func getBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var err error
	var hash Uint256
	switch (params[0]).(type) {
	// block height
	case float64:
		index := uint32(params[0].(float64))
		hash, err = ledger.DefaultLedger.Store.GetBlockHash(index)
		if err != nil {
			return UNetworkRPCUnknownBlock
		}
	// block hash
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
			return UNetworkRPCInvalidTransaction
		}
	default:
		return UNetworkRPCInvalidParameter
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return UNetworkRPCUnknownBlock
	}

	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    BytesToHexString(block.Blockdata.PrevBlockHash.ToArrayReverse()),
		TransactionsRoot: BytesToHexString(block.Blockdata.TransactionsRoot.ToArrayReverse()),
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		ConsensusData:    block.Blockdata.ConsensusData,
		NextBookKeeper:   BytesToHexString(block.Blockdata.NextBookKeeper.ToArrayReverse()),
		Program: ProgramInfo{
			Code:      BytesToHexString(block.Blockdata.Program.Code),
			Parameter: BytesToHexString(block.Blockdata.Program.Parameter),
		},
		Hash: BytesToHexString(hash.ToArrayReverse()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
	}

	b := BlockInfo{
		Hash:         BytesToHexString(hash.ToArrayReverse()),
		BlockData:    blockHead,
		Transactions: trans,
	}
	return UNetworkRPC(b)
}

func getBlockCount(params []interface{}) map[string]interface{} {
	return UNetworkRPC(ledger.DefaultLedger.Blockchain.BlockHeight + 1)
}

// A JSON example for getblockhash method as following:
//   {"jsonrpc": "2.0", "method": "getblockhash", "params": [1], "id": 0}
func getBlockHash(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	switch params[0].(type) {
	case float64:
		height := uint32(params[0].(float64))
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return UNetworkRPCUnknownBlock
		}
		return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
	default:
		return UNetworkRPCInvalidParameter
	}
}

func getConnectionCount(params []interface{}) map[string]interface{} {
	return UNetworkRPC(node.GetConnectionCnt())
}

func getRawMemPool(params []interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	if len(txs) == 0 {
		return UNetworkRPCNil
	}
	return UNetworkRPC(txs)
}

// A JSON example for getrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "getrawtransaction", "params": ["transactioin hash in hex"], "id": 0}
func getRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return UNetworkRPCInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UNetworkRPCUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		return UNetworkRPC(tran)
	default:
		return UNetworkRPCInvalidParameter
	}
}
// A JSON example for sendrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "sendrawtransaction", "params": ["raw transactioin in hex"], "id": 0}
func sendRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return UNetworkRPCInvalidTransaction
		}

		//if txn.TxType != tx.InvokeCode && txn.TxType != tx.DeployCode &&
		//	txn.TxType != tx.TransferAsset && txn.TxType != tx.LockAsset &&
		//	txn.TxType != tx.BookKeeper {
		//	return UNetworkRPC("invalid transaction type")
		//}
		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UNetworkRPC(errCode.Error())
		}
	default:
		return UNetworkRPCInvalidParameter
	}
	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}

func getTxout(params []interface{}) map[string]interface{} {
	//TODO
	return UNetworkRPCUnsupported
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func submitBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, _ := HexStringToBytes(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return UNetworkRPCInvalidBlock
		}
		if err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return UNetworkRPCInvalidBlock
		}
		if err := node.LocalNode().CleanSubmittedTransactions(&block); err != nil {
			return UNetworkRPCInternalError
		}
		if err := node.Xmit(&block); err != nil {
			return UNetworkRPCInternalError
		}
	default:
		return UNetworkRPCInvalidParameter
	}
	return UNetworkRPCSuccess
}

func getNeighbor(params []interface{}) map[string]interface{} {
	addr, _ := node.GetNeighborAddrs()
	return UNetworkRPC(addr)
}

func getNodeState(params []interface{}) map[string]interface{} {
	n := NodeInfo{
		State:    uint(node.GetState()),
		Time:     node.GetTime(),
		Port:     node.GetPort(),
		ID:       node.GetID(),
		Version:  node.Version(),
		Services: node.Services(),
		Relay:    node.GetRelay(),
		Height:   node.GetHeight(),
		TxnCnt:   node.GetTxnCnt(),
		RxTxnCnt: node.GetRxTxnCnt(),
	}
	return UNetworkRPC(n)
}

func startConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Start(); err != nil {
		return UNetworkRPCFailed
	}
	return UNetworkRPCSuccess
}

func stopConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Halt(); err != nil {
		return UNetworkRPCFailed
	}
	return UNetworkRPCSuccess
}

func sendSampleTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var txType string
	switch params[0].(type) {
	case string:
		txType = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}

	issuer, err := account.NewAccount()
	if err != nil {
		return UNetworkRPC("Failed to create account")
	}
	admin := issuer

	rbuf := make([]byte, RANDBYTELEN)
	rand.Read(rbuf)
	switch string(txType) {
	case "perf":
		num := 1
		if len(params) == 2 {
			switch params[1].(type) {
			case float64:
				num = int(params[1].(float64))
			}
		}
		for i := 0; i < num; i++ {
			regTx := NewRegTx(BytesToHexString(rbuf), i, admin, issuer)
			SignTx(admin, regTx)
			VerifyAndSendTx(regTx)
		}
		return UNetworkRPC(fmt.Sprintf("%d transaction(s) was sent", num))
	default:
		return UNetworkRPC("Invalid transacion type")
	}
}

func setDebugInfo(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCInvalidParameter
	}
	switch params[0].(type) {
	case float64:
		level := params[0].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return UNetworkRPCInvalidParameter
		}
	default:
		return UNetworkRPCInvalidParameter
	}
	return UNetworkRPCSuccess
}

func getVersion(params []interface{}) map[string]interface{} {
	return UNetworkRPC(config.Version)
}

func uploadDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}

	rbuf := make([]byte, 4)
	rand.Read(rbuf)
	tmpname := hex.EncodeToString(rbuf)

	str := params[0].(string)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	f, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return UNetworkRPCIOError
	}
	defer f.Close()
	f.Write(data)

	refpath, err := AddFileIPFS(tmpname, true)
	if err != nil {
		return UNetworkRPCAPIError
	}

	return UNetworkRPC(refpath)

}

func regDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return UNetworkRPCInvalidTransaction
		}

		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UNetworkRPCInternalError
		}
	default:
		return UNetworkRPCInvalidParameter
	}
	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}

func catDataRecord(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		b, err := HexStringToBytesReverse(str)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(b))
		if err != nil {
			return UNetworkRPCInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UNetworkRPCUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)
		//ref := string(record.RecordData[:])
		return UNetworkRPC(info)
	default:
		return UNetworkRPCInvalidParameter
	}
}

func getDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return UNetworkRPCInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UNetworkRPCUnknownTransaction
		}

		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)

		err = GetFileIPFS(info.IPFSPath, info.Filename)
		if err != nil {
			return UNetworkRPCAPIError
		}
		//TODO: shoud return download address
		return UNetworkRPCSuccess
	default:
		return UNetworkRPCInvalidParameter
	}
}

var Wallet account.Client

func getWalletDir() string {
	home, _ := homedir.Dir()
	return home + "/.wallet/"
}

func addAccount(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return UNetworkRPC("open wallet first")
	}
	account, err := Wallet.CreateAccount()
	if err != nil {
		return UNetworkRPC("create account error:" + err.Error())
	}

	if err := Wallet.CreateContract(account); err != nil {
		return UNetworkRPC("create contract error:" + err.Error())
	}

	address, err := account.ProgramHash.ToAddress()
	if err != nil {
		return UNetworkRPC("generate address error:" + err.Error())
	}

	return UNetworkRPC(address)
}

func deleteAccount(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var address string
	switch params[0].(type) {
	case string:
		address = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("open wallet first")
	}
	programHash, err := ToScriptHash(address)
	if err != nil {
		return UNetworkRPC("invalid address:" + err.Error())
	}
	if err := Wallet.DeleteAccount(programHash); err != nil {
		return UNetworkRPC("Delete account error:" + err.Error())
	}
	if err := Wallet.DeleteContract(programHash); err != nil {
		return UNetworkRPC("Delete contract error:" + err.Error())
	}

	return UNetworkRPC(true)
}

func makeRegTxn(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UNetworkRPCNil
	}
	var assetName, assetValue string
	switch params[0].(type) {
	case string:
		assetName = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		assetValue = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("open wallet first")
	}

	regTxn, err := sdk.MakeRegTransaction(Wallet, assetName, assetValue)
	if err != nil {
		return UNetworkRPCInternalError
	}

	if errCode := VerifyAndSendTx(regTxn); errCode != ErrNoError {
		return UNetworkRPCInvalidTransaction
	}
	return UNetworkRPC(true)
}

func makeIssueTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var asset, value, address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("open wallet first")
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UNetworkRPC("invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UNetworkRPC("invalid asset hash")
	}
	issueTxn, err := sdk.MakeIssueTransaction(Wallet, assetID, address, value)
	if err != nil {
		return UNetworkRPCInternalError
	}

	if errCode := VerifyAndSendTx(issueTxn); errCode != ErrNoError {
		return UNetworkRPCInvalidTransaction
	}

	return UNetworkRPC(true)
}

func sendToAddresses(params []interface{}) map[string]interface{} {
	type Outputinfo struct {
		Assetid  string
		Outputs map[string]string
	}
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var outs Outputinfo
	outs.Outputs = make(map[string]string)
	mapobj := params[0].(map[string]interface{})
	outs.Assetid = mapobj["Assetid"].(string)
	mapouts := mapobj["Outputs"].(map[string]interface{})
	for key, vobj := range mapouts {
		outs.Outputs[key] = vobj.(string)
	}

	tmp, err := HexStringToBytesReverse(outs.Assetid)
	if err != nil {
		return UNetworkRPC("invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UNetworkRPC("invalid asset hash")
	}
	var batchouts []sdk.BatchOut
	for ast, v := range outs.Outputs {
		batchouts = append(batchouts, sdk.BatchOut{ast, v})
	}

	txn, err := sdk.MakeTransferTransaction(Wallet, assetID, batchouts...)
	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC("error: " + errCode.Error())
	}
	txHash := txn.Hash()
	return UNetworkRPC(BytesToHexString(txHash.ToArrayReverse()))
}

func sendToAddress(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var asset, address, value string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		address = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		value = params[2].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("error : wallet is not opened")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UNetworkRPC("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UNetworkRPC("error: invalid asset hash")
	}
	txn, err := sdk.MakeTransferTransaction(Wallet, assetID, batchOut)
	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC("error: " + errCode.Error())
	}
	txHash := txn.Hash()
	return UNetworkRPC(BytesToHexString(txHash.ToArrayReverse()))
}

func lockAsset(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var asset, value string
	var height float64
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case float64:
		height = params[2].(float64)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("error: invalid wallet instance")
	}

	accts := Wallet.GetAccounts()
	if len(accts) > 1 {
		return UNetworkRPC("error: does't support multi-addresses wallet locking asset")
	}

	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UNetworkRPC("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UNetworkRPC("error: invalid asset hash")
	}

	txn, err := sdk.MakeLockAssetTransaction(Wallet, assetID, value, uint32(height))
	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}

	txnHash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC(errCode.Error())
	}
	return UNetworkRPC(BytesToHexString(txnHash.ToArrayReverse()))
}

func signMultisigTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var signedrawtxn string
	switch params[0].(type) {
	case string:
		signedrawtxn = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}

	rawtxn, _ := HexStringToBytes(signedrawtxn)
	var txn tx.Transaction
	txn.Deserialize(bytes.NewReader(rawtxn))
	if len(txn.Programs) <= 0 {
		return UNetworkRPC("missing the first signature")
	}

	found := false
	programHashes := txn.ParseTransactionCode()
	for _, hash := range programHashes {
		acct := Wallet.GetAccountByProgramHash(hash)
		if acct != nil {
			found = true
			sig, _ := signature.SignBySigner(&txn, acct)
			txn.AppendNewSignature(sig)
		}
	}
	if !found {
		return UNetworkRPC("error: no available account detected")
	}

	_, needsig, err := txn.ParseTransactionSig()
	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}
	if needsig == 0 {
		txnHash := txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UNetworkRPC(errCode.Error())
		}
		return UNetworkRPC(BytesToHexString(txnHash.ToArrayReverse()))
	} else {
		var buffer bytes.Buffer
		txn.Serialize(&buffer)
		return UNetworkRPC(BytesToHexString(buffer.Bytes()))
	}
}

func createMultisigTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 4 {
		return UNetworkRPCNil
	}
	var asset, from, address, value string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		from = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[3].(type) {
	case string:
		value = params[3].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	if Wallet == nil {
		return UNetworkRPC("error : wallet is not opened")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return UNetworkRPC("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return UNetworkRPC("error: invalid asset hash")
	}
	txn, err := sdk.MakeMultisigTransferTransaction(Wallet, assetID, from, batchOut)
	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}

	_, needsig, err := txn.ParseTransactionSig()
	if err != nil {
		return UNetworkRPC("error: " + err.Error())
	}
	if needsig == 0 {
		txnHash := txn.Hash()
		if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
			return UNetworkRPC(errCode.Error())
		}
		return UNetworkRPC(BytesToHexString(txnHash.ToArrayReverse()))
	} else {
		var buffer bytes.Buffer
		txn.Serialize(&buffer)
		return UNetworkRPC(BytesToHexString(buffer.Bytes()))
	}
}

func registerUser(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UNetworkRPCNil
	}
	var userName, userProgramHash string
	switch params[0].(type) {
	case string:
		userName = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		userProgramHash = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}

	programHash, err := ToScriptHash(userProgramHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	txn, err := sdk.MakeRegisterUserTransaction(userName, programHash)
	if err != nil {
		return UNetworkRPCInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC(errCode.Error())
	}

	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}

/*func postArticle(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UNetworkRPCNil
	}
	var articleHash, author string
	switch params[0].(type) {
	case string:
		articleHash = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		author = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}

	tmpHash, err := HexStringToBytes(articleHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	aHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	txn, err := sdk.MakePostArticleTransaction(Wallet, aHash, author)
	if err != nil {
		return UNetworkRPCInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC(errCode.Error())
	}

	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}*/

func replyArticle(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var postTxnHash, contentHash, replier string
	var err error
	switch params[0].(type) {
	case string:
		postTxnHash = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		contentHash = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		replier = params[2].(string)
	default:
		return UNetworkRPCInvalidParameter
	}

	tmpHash, err := HexStringToBytesReverse(postTxnHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	pHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}

	tmpHash, err = HexStringToBytes(contentHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	cHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}

	txn, err := sdk.MakeReplyArticleTransaction(Wallet, pHash, cHash, replier)
	if err != nil {
		return UNetworkRPCInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC(errCode.Error())
	}

	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}

/*func likeArticle(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var postTxnHash, liker string
	var likeType forum.LikeType
	switch params[0].(type) {
	case string:
		postTxnHash = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		liker = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		v, err := strconv.ParseInt(params[2].(string), 10, 8)
		if err != nil {
			return UNetworkRPCInvalidParameter
		}
		likeType = forum.LikeType(v)
	default:
		return UNetworkRPCInvalidParameter
	}

	tmpHash, err := HexStringToBytesReverse(postTxnHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	aHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	txn, err := sdk.MakeLikeArticleTransaction(Wallet, aHash, liker, likeType)
	if err != nil {
		return UNetworkRPCInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC(errCode.Error())
	}

	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}*/

func withdrawal(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UNetworkRPCNil
	}
	var payee, recipient, asset, amount string
	switch params[0].(type) {
	case string:
		payee = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[1].(type) {
	case string:
		recipient = params[1].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[2].(type) {
	case string:
		asset = params[2].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	switch params[3].(type) {
	case string:
		amount = params[3].(string)
	default:
		return UNetworkRPCInvalidParameter
	}

	tmpHash, err := HexStringToBytesReverse(recipient)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	aHash, err := Uint160ParseFromBytes(tmpHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}

	tmpHash, err = HexStringToBytesReverse(asset)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	bHash, err := Uint256ParseFromBytes(tmpHash)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}

	txn, err := sdk.MakeWithdrawalTransaction(Wallet, payee, aHash, bHash, amount)
	if err != nil {
		return UNetworkRPCInternalError
	}

	hash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UNetworkRPC(errCode.Error())
	}

	return UNetworkRPC(BytesToHexString(hash.ToArrayReverse()))
}

func getLikeArticleAdresslist(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UNetworkRPCNil
	}
	var ahashstr string
	switch params[0].(type) {
	case string:
		ahashstr = params[0].(string)
	default:
		return UNetworkRPCInvalidParameter
	}
	ahashbt, err := HexStringToBytesReverse(ahashstr)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	ahash256, err := Uint256ParseFromBytes(ahashbt)
	if err != nil {
		return UNetworkRPCInvalidParameter
	}
	var results []string
	likeinfoarray,err := ledger.DefaultLedger.Store.GetLikeInfo(ahash256)
	if (err != nil) {
		return UNetworkRPC(err.Error())
	}
	for _, likeinfo := range likeinfoarray {
		author := likeinfo.Liker
		userinfo, err := ledger.DefaultLedger.Store.GetUserInfo(author)
		if (err != nil) {
			return UNetworkRPC(err.Error())
		}
		ProgramHashstr := BytesToHexString(userinfo.UserProgramHash.ToArrayReverse())
		results = append(results, ProgramHashstr)
	}
	return UNetworkRPC(results)
}