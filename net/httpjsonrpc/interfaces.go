package httpjsonrpc

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

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
		trans.Outputs[n].ProgramHash = address
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
			trans.AssetOutputs[n].Txout[m].ProgramHash = BytesToHexString(v[m].ProgramHash.ToArray())
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
	return uNetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}
func getBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var err error
	var hash Uint256
	switch (params[0]).(type) {
	// block height
	case float64:
		index := uint32(params[0].(float64))
		hash, err = ledger.DefaultLedger.Store.GetBlockHash(index)
		if err != nil {
			return uNetworkRpcUnknownBlock
		}
	// block hash
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return uNetworkRpcInvalidParameter
		}
		if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
			return uNetworkRpcInvalidTransaction
		}
	default:
		return uNetworkRpcInvalidParameter
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return uNetworkRpcUnknownBlock
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
	return uNetworkRpc(b)
}

func getBlockCount(params []interface{}) map[string]interface{} {
	return uNetworkRpc(ledger.DefaultLedger.Blockchain.BlockHeight + 1)
}

// A JSON example for getblockhash method as following:
//   {"jsonrpc": "2.0", "method": "getblockhash", "params": [1], "id": 0}
func getBlockHash(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	switch params[0].(type) {
	case float64:
		height := uint32(params[0].(float64))
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return uNetworkRpcUnknownBlock
		}
		return uNetworkRpc(BytesToHexString(hash.ToArrayReverse()))
	default:
		return uNetworkRpcInvalidParameter
	}
}

func getConnectionCount(params []interface{}) map[string]interface{} {
	return uNetworkRpc(node.GetConnectionCnt())
}

func getRawMemPool(params []interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	if len(txs) == 0 {
		return uNetworkRpcNil
	}
	return uNetworkRpc(txs)
}

// A JSON example for getrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "getrawtransaction", "params": ["transactioin hash in hex"], "id": 0}
func getRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return uNetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return uNetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return uNetworkRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		return uNetworkRpc(tran)
	default:
		return uNetworkRpcInvalidParameter
	}
}

// A JSON example for sendrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "sendrawtransaction", "params": ["raw transactioin in hex"], "id": 0}
func sendRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return uNetworkRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return uNetworkRpcInvalidTransaction
		}
		if txn.TxType != tx.InvokeCode && txn.TxType != tx.DeployCode &&
			txn.TxType != tx.TransferAsset && txn.TxType != tx.LockAsset &&
			txn.TxType != tx.BookKeeper {
			return uNetworkRpc("invalid transaction type")
		}
		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return uNetworkRpc(errCode.Error())
		}
	default:
		return uNetworkRpcInvalidParameter
	}
	return uNetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func getTxout(params []interface{}) map[string]interface{} {
	//TODO
	return uNetworkRpcUnsupported
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func submitBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, _ := HexStringToBytes(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return uNetworkRpcInvalidBlock
		}
		if err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return uNetworkRpcInvalidBlock
		}
		if err := node.Xmit(&block); err != nil {
			return uNetworkRpcInternalError
		}
	default:
		return uNetworkRpcInvalidParameter
	}
	return uNetworkRpcSuccess
}

func getNeighbor(params []interface{}) map[string]interface{} {
	addr, _ := node.GetNeighborAddrs()
	return uNetworkRpc(addr)
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
	return uNetworkRpc(n)
}

func startConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Start(); err != nil {
		return uNetworkRpcFailed
	}
	return uNetworkRpcSuccess
}

func stopConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Halt(); err != nil {
		return uNetworkRpcFailed
	}
	return uNetworkRpcSuccess
}

func sendSampleTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var txType string
	switch params[0].(type) {
	case string:
		txType = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}

	issuer, err := account.NewAccount()
	if err != nil {
		return uNetworkRpc("Failed to create account")
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
		return uNetworkRpc(fmt.Sprintf("%d transaction(s) was sent", num))
	default:
		return uNetworkRpc("Invalid transacion type")
	}
}

func setDebugInfo(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcInvalidParameter
	}
	switch params[0].(type) {
	case float64:
		level := params[0].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return uNetworkRpcInvalidParameter
		}
	default:
		return uNetworkRpcInvalidParameter
	}
	return uNetworkRpcSuccess
}

func getVersion(params []interface{}) map[string]interface{} {
	return uNetworkRpc(config.Version)
}

func uploadDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}

	rbuf := make([]byte, 4)
	rand.Read(rbuf)
	tmpname := hex.EncodeToString(rbuf)

	str := params[0].(string)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return uNetworkRpcInvalidParameter
	}
	f, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return uNetworkRpcIOError
	}
	defer f.Close()
	f.Write(data)

	refpath, err := AddFileIPFS(tmpname, true)
	if err != nil {
		return uNetworkRpcAPIError
	}

	return uNetworkRpc(refpath)

}

func regDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytes(str)
		if err != nil {
			return uNetworkRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return uNetworkRpcInvalidTransaction
		}

		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return uNetworkRpcInternalError
		}
	default:
		return uNetworkRpcInvalidParameter
	}
	return uNetworkRpc(BytesToHexString(hash.ToArrayReverse()))
}

func catDataRecord(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		b, err := HexStringToBytesReverse(str)
		if err != nil {
			return uNetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(b))
		if err != nil {
			return uNetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return uNetworkRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)
		//ref := string(record.RecordData[:])
		return uNetworkRpc(info)
	default:
		return uNetworkRpcInvalidParameter
	}
}

func getDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := HexStringToBytesReverse(str)
		if err != nil {
			return uNetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return uNetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return uNetworkRpcUnknownTransaction
		}

		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)

		err = GetFileIPFS(info.IPFSPath, info.Filename)
		if err != nil {
			return uNetworkRpcAPIError
		}
		//TODO: shoud return download address
		return uNetworkRpcSuccess
	default:
		return uNetworkRpcInvalidParameter
	}
}

var Wallet account.Client

func getWalletDir() string {
	home, _ := homedir.Dir()
	return home + "/.wallet/"
}

func createWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return uNetworkRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return uNetworkRpcInternalError
		}
	}
	walletPath := walletDir + "wallet.dat"
	if FileExisted(walletPath) {
		return uNetworkRpcWalletAlreadyExists
	}
	_, err := account.Create(walletPath, password)
	if err != nil {
		return uNetworkRpcFailed
	}
	return uNetworkRpcSuccess
}

func openWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return uNetworkRpcInvalidParameter
	}
	resp := make(map[string]string)
	walletPath := getWalletDir() + "wallet.dat"
	if !FileExisted(walletPath) {
		resp["success"] = "false"
		resp["message"] = "wallet doesn't exist"
		return uNetworkRpc(resp)
	}
	wallet, err := account.Open(walletPath, password)
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "password wrong"
		return uNetworkRpc(resp)
	}
	Wallet = wallet
	programHash, err := wallet.LoadStoredData("ProgramHash")
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "wallet file broken"
		return uNetworkRpc(resp)
	}
	resp["success"] = "true"
	resp["message"] = BytesToHexString(programHash)
	return uNetworkRpc(resp)
}

func closeWallet(params []interface{}) map[string]interface{} {
	Wallet = nil
	return uNetworkRpcSuccess
}

func recoverWallet(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return uNetworkRpcNil
	}
	var privateKey string
	var walletPassword string
	switch params[0].(type) {
	case string:
		privateKey = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		walletPassword = params[1].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return uNetworkRpcInternalError
		}
	}
	walletName := fmt.Sprintf("wallet-%s-recovered.dat", time.Now().Format("2006-01-02-15-04-05"))
	walletPath := walletDir + walletName
	if FileExisted(walletPath) {
		return uNetworkRpcWalletAlreadyExists
	}
	_, err := account.Recover(walletPath, []byte(walletPassword), privateKey)
	if err != nil {
		return uNetworkRpc("wallet recovery failed")
	}

	return uNetworkRpcSuccess
}

func getWalletKey(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return uNetworkRpc("open wallet first")
	}
	account, _ := Wallet.GetDefaultAccount()
	encodedPublickKey, _ := account.PublicKey.EncodePoint(true)
	resp := make(map[string]string)
	resp["PublicKey"] = BytesToHexString(encodedPublickKey)
	resp["PrivateKey"] = BytesToHexString(account.PrivateKey)
	resp["ProgramHash"] = BytesToHexString(account.ProgramHash.ToArrayReverse())

	return uNetworkRpc(resp)
}

func addAccount(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return uNetworkRpc("open wallet first")
	}
	account, err := Wallet.CreateAccount()
	if err != nil {
		return uNetworkRpc("create account error:" + err.Error())
	}

	if err := Wallet.CreateContract(account); err != nil {
		return uNetworkRpc("create contract error:" + err.Error())
	}

	address, err := account.ProgramHash.ToAddress()
	if err != nil {
		return uNetworkRpc("generate address error:" + err.Error())
	}

	return uNetworkRpc(address)
}

func deleteAccount(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var address string
	switch params[0].(type) {
	case string:
		address = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return uNetworkRpc("open wallet first")
	}
	programHash, err := ToScriptHash(address)
	if err != nil {
		return uNetworkRpc("invalid address:" + err.Error())
	}
	if err := Wallet.DeleteAccount(programHash); err != nil {
		return uNetworkRpc("Delete account error:" + err.Error())
	}
	if err := Wallet.DeleteContract(programHash); err != nil {
		return uNetworkRpc("Delete contract error:" + err.Error())
	}
	if err := Wallet.DeleteCoinsData(programHash); err != nil {
		return uNetworkRpc("Delete coins error:" + err.Error())
	}

	return uNetworkRpc(true)
}

func makeRegTxn(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return uNetworkRpcNil
	}
	var assetName, assetValue string
	switch params[0].(type) {
	case string:
		assetName = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		assetValue = params[1].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return uNetworkRpc("open wallet first")
	}

	regTxn, err := sdk.MakeRegTransaction(Wallet, assetName, assetValue)
	if err != nil {
		return uNetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(regTxn); errCode != ErrNoError {
		return uNetworkRpcInvalidTransaction
	}
	return uNetworkRpc(true)
}

func makeIssueTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return uNetworkRpcNil
	}
	var asset, value, address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return uNetworkRpc("open wallet first")
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return uNetworkRpc("invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return uNetworkRpc("invalid asset hash")
	}
	issueTxn, err := sdk.MakeIssueTransaction(Wallet, assetID, address, value)
	if err != nil {
		return uNetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(issueTxn); errCode != ErrNoError {
		return uNetworkRpcInvalidTransaction
	}

	return uNetworkRpc(true)
}

func sendToAddress(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return uNetworkRpcNil
	}
	var asset, address, value string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		address = params[1].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		value = params[2].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return uNetworkRpc("error : wallet is not opened")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return uNetworkRpc("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return uNetworkRpc("error: invalid asset hash")
	}
	txn, err := sdk.MakeTransferTransaction(Wallet, assetID, batchOut)
	if err != nil {
		return uNetworkRpc("error: " + err.Error())
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return uNetworkRpc("error: " + errCode.Error())
	}
	txHash := txn.Hash()
	return uNetworkRpc(BytesToHexString(txHash.ToArrayReverse()))
}

func lockAsset(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return uNetworkRpcNil
	}
	var asset, value string
	var height float64
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		value = params[1].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case float64:
		height = params[2].(float64)
	default:
		return uNetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return uNetworkRpc("error: invalid wallet instance")
	}

	accts := Wallet.GetAccounts()
	if len(accts) > 1 {
		return uNetworkRpc("error: does't support multi-addresses wallet locking asset")
	}

	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return uNetworkRpc("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return uNetworkRpc("error: invalid asset hash")
	}

	txn, err := sdk.MakeLockAssetTransaction(Wallet, assetID, value, uint32(height))
	if err != nil {
		return uNetworkRpc("error: " + err.Error())
	}

	txnHash := txn.Hash()
	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return uNetworkRpc(errCode.Error())
	}
	return uNetworkRpc(BytesToHexString(txnHash.ToArrayReverse()))
}

func signMultisigTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return uNetworkRpcNil
	}
	var signedrawtxn string
	switch params[0].(type) {
	case string:
		signedrawtxn = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}

	rawtxn, _ := HexStringToBytes(signedrawtxn)
	var txn tx.Transaction
	txn.Deserialize(bytes.NewReader(rawtxn))
	if len(txn.Programs) <= 0 {
		return uNetworkRpc("missing the first signature")
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
		return uNetworkRpc("error: no available account detected")
	}

	_, needsig, err := txn.ParseTransactionSig()
	if err != nil {
		return uNetworkRpc("error: " + err.Error())
	}
	if needsig == 0 {
		txnHash := txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return uNetworkRpc(errCode.Error())
		}
		return uNetworkRpc(BytesToHexString(txnHash.ToArrayReverse()))
	} else {
		var buffer bytes.Buffer
		txn.Serialize(&buffer)
		return uNetworkRpc(BytesToHexString(buffer.Bytes()))
	}
}

func createMultisigTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 4 {
		return uNetworkRpcNil
	}
	var asset, from, address, value string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		from = params[1].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	switch params[3].(type) {
	case string:
		value = params[3].(string)
	default:
		return uNetworkRpcInvalidParameter
	}
	if Wallet == nil {
		return uNetworkRpc("error : wallet is not opened")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	tmp, err := HexStringToBytesReverse(asset)
	if err != nil {
		return uNetworkRpc("error: invalid asset ID")
	}
	var assetID Uint256
	if err := assetID.Deserialize(bytes.NewReader(tmp)); err != nil {
		return uNetworkRpc("error: invalid asset hash")
	}
	txn, err := sdk.MakeMultisigTransferTransaction(Wallet, assetID, from, batchOut)
	if err != nil {
		return uNetworkRpc("error: " + err.Error())
	}

	_, needsig, err := txn.ParseTransactionSig()
	if err != nil {
		return uNetworkRpc("error: " + err.Error())
	}
	if needsig == 0 {
		txnHash := txn.Hash()
		if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
			return uNetworkRpc(errCode.Error())
		}
		return uNetworkRpc(BytesToHexString(txnHash.ToArrayReverse()))
	} else {
		var buffer bytes.Buffer
		txn.Serialize(&buffer)
		return uNetworkRpc(BytesToHexString(buffer.Bytes()))
	}
}

func getBalance(params []interface{}) map[string]interface{} {
	if Wallet == nil {
		return uNetworkRpc("open wallet first")
	}
	type AssetInfo struct {
		AssetID string
		Value   string
	}
	balances := make(map[string][]*AssetInfo)
	accounts := Wallet.GetAccounts()
	coins := Wallet.GetCoins()
	for _, account := range accounts {
		assetList := []*AssetInfo{}
		programHash := account.ProgramHash
		for _, coin := range coins {
			if programHash == coin.Output.ProgramHash {
				var existed bool
				assetString := BytesToHexString(coin.Output.AssetID.ToArray())
				for _, info := range assetList {
					if info.AssetID == assetString {
						info.Value += coin.Output.Value.String()
						existed = true
						break
					}
				}
				if !existed {
					assetList = append(assetList, &AssetInfo{AssetID: assetString, Value: coin.Output.Value.String()})
				}
			}
		}
		address, _ := programHash.ToAddress()
		balances[address] = assetList
	}

	return uNetworkRpc(balances)
}
