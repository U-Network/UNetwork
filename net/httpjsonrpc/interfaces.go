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

	"UGCNetwork/account"
	. "UGCNetwork/common"
	"UGCNetwork/common/config"
	"UGCNetwork/common/log"
	"UGCNetwork/core/ledger"
	tx "UGCNetwork/core/transaction"
	"UGCNetwork/core/transaction/payload"
	. "UGCNetwork/errors"
	"UGCNetwork/sdk"

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
		trans.Attributes[n].Data = ToHexString(v.Data)
		n++
	}

	n = 0
	trans.UTXOInputs = make([]UTXOTxInputInfo, len(ptx.UTXOInputs))
	for _, v := range ptx.UTXOInputs {
		trans.UTXOInputs[n].ReferTxID = ToHexString(v.ReferTxID.ToArray())
		trans.UTXOInputs[n].ReferTxOutputIndex = v.ReferTxOutputIndex
		n++
	}

	n = 0
	trans.BalanceInputs = make([]BalanceTxInputInfo, len(ptx.BalanceInputs))
	for _, v := range ptx.BalanceInputs {
		trans.BalanceInputs[n].AssetID = ToHexString(v.AssetID.ToArray())
		trans.BalanceInputs[n].Value = v.Value
		trans.BalanceInputs[n].ProgramHash = ToHexString(v.ProgramHash.ToArray())
		n++
	}

	n = 0
	trans.Outputs = make([]TxoutputInfo, len(ptx.Outputs))
	for _, v := range ptx.Outputs {
		trans.Outputs[n].AssetID = ToHexString(v.AssetID.ToArray())
		trans.Outputs[n].Value = v.Value.String()
		address, _ := v.ProgramHash.ToAddress()
		trans.Outputs[n].Address = address
		n++
	}

	n = 0
	trans.Programs = make([]ProgramInfo, len(ptx.Programs))
	for _, v := range ptx.Programs {
		trans.Programs[n].Code = ToHexString(v.Code)
		trans.Programs[n].Parameter = ToHexString(v.Parameter)
		n++
	}

	n = 0
	trans.AssetOutputs = make([]TxoutputMap, len(ptx.AssetOutputs))
	for k, v := range ptx.AssetOutputs {
		trans.AssetOutputs[n].Key = k
		trans.AssetOutputs[n].Txout = make([]TxoutputInfo, len(v))
		for m := 0; m < len(v); m++ {
			trans.AssetOutputs[n].Txout[m].AssetID = ToHexString(v[m].AssetID.ToArray())
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

	mhash := ptx.Hash()
	trans.Hash = ToHexString(mhash.ToArray())

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
	return UgcNetworkRpc(ToHexString(hash.ToArray()))
}

// Input JSON string examples for getblock method as following:
//   {"jsonrpc": "2.0", "method": "getblock", "params": [1], "id": 0}
//   {"jsonrpc": "2.0", "method": "getblock", "params": ["aabbcc.."], "id": 0}
func getBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var err error
	var hash Uint256
	switch (params[0]).(type) {
	// block height
	case float64:
		index := uint32(params[0].(float64))
		hash, err = ledger.DefaultLedger.Store.GetBlockHash(index)
		if err != nil {
			return UgcNetworkRpcUnknownBlock
		}
	// block hash
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return UgcNetworkRpcInvalidParameter
		}
		if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
			return UgcNetworkRpcInvalidTransaction
		}
	default:
		return UgcNetworkRpcInvalidParameter
	}

	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return UgcNetworkRpcUnknownBlock
	}

	blockHead := &BlockHead{
		Version:          block.Blockdata.Version,
		PrevBlockHash:    ToHexString(block.Blockdata.PrevBlockHash.ToArray()),
		TransactionsRoot: ToHexString(block.Blockdata.TransactionsRoot.ToArray()),
		Timestamp:        block.Blockdata.Timestamp,
		Height:           block.Blockdata.Height,
		ConsensusData:    block.Blockdata.ConsensusData,
		NextBookKeeper:   ToHexString(block.Blockdata.NextBookKeeper.ToArray()),
		Program: ProgramInfo{
			Code:      ToHexString(block.Blockdata.Program.Code),
			Parameter: ToHexString(block.Blockdata.Program.Parameter),
		},
		Hash: ToHexString(hash.ToArray()),
	}

	trans := make([]*Transactions, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		trans[i] = TransArryByteToHexString(block.Transactions[i])
	}

	b := BlockInfo{
		Hash:         ToHexString(hash.ToArray()),
		BlockData:    blockHead,
		Transactions: trans,
	}
	return UgcNetworkRpc(b)
}

func getBlockCount(params []interface{}) map[string]interface{} {
	return UgcNetworkRpc(ledger.DefaultLedger.Blockchain.BlockHeight + 1)
}

// A JSON example for getblockhash method as following:
//   {"jsonrpc": "2.0", "method": "getblockhash", "params": [1], "id": 0}
func getBlockHash(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	switch params[0].(type) {
	case float64:
		height := uint32(params[0].(float64))
		hash, err := ledger.DefaultLedger.Store.GetBlockHash(height)
		if err != nil {
			return UgcNetworkRpcUnknownBlock
		}
		return UgcNetworkRpc(fmt.Sprintf("%016x", hash))
	default:
		return UgcNetworkRpcInvalidParameter
	}
}

func getConnectionCount(params []interface{}) map[string]interface{} {
	return UgcNetworkRpc(node.GetConnectionCnt())
}

func getRawMemPool(params []interface{}) map[string]interface{} {
	txs := []*Transactions{}
	txpool := node.GetTxnPool(false)
	for _, t := range txpool {
		txs = append(txs, TransArryByteToHexString(t))
	}
	if len(txs) == 0 {
		return UgcNetworkRpcNil
	}
	return UgcNetworkRpc(txs)
}

// A JSON example for getrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "getrawtransaction", "params": ["transactioin hash in hex"], "id": 0}
func getRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return UgcNetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return UgcNetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UgcNetworkRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		return UgcNetworkRpc(tran)
	default:
		return UgcNetworkRpcInvalidParameter
	}
}

// A JSON example for sendrawtransaction method as following:
//   {"jsonrpc": "2.0", "method": "sendrawtransaction", "params": ["raw transactioin in hex"], "id": 0}
func sendRawTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return UgcNetworkRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return UgcNetworkRpcInvalidTransaction
		}
		if txn.TxType != tx.TransferAsset && txn.TxType != tx.BookKeeper {
			return UgcNetworkRpc("invalid transaction type")
		}
		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UgcNetworkRpc(errCode.Error())
		}
	default:
		return UgcNetworkRpcInvalidParameter
	}
	return UgcNetworkRpc(ToHexString(hash.ToArray()))
}

func getTxout(params []interface{}) map[string]interface{} {
	//TODO
	return UgcNetworkRpcUnsupported
}

// A JSON example for submitblock method as following:
//   {"jsonrpc": "2.0", "method": "submitblock", "params": ["raw block in hex"], "id": 0}
func submitBlock(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, _ := hex.DecodeString(str)
		var block ledger.Block
		if err := block.Deserialize(bytes.NewReader(hex)); err != nil {
			return UgcNetworkRpcInvalidBlock
		}
		if err := ledger.DefaultLedger.Blockchain.AddBlock(&block); err != nil {
			return UgcNetworkRpcInvalidBlock
		}
		if err := node.Xmit(&block); err != nil {
			return UgcNetworkRpcInternalError
		}
	default:
		return UgcNetworkRpcInvalidParameter
	}
	return UgcNetworkRpcSuccess
}

func getNeighbor(params []interface{}) map[string]interface{} {
	addr, _ := node.GetNeighborAddrs()
	return UgcNetworkRpc(addr)
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
	return UgcNetworkRpc(n)
}

func startConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Start(); err != nil {
		return UgcNetworkRpcFailed
	}
	return UgcNetworkRpcSuccess
}

func stopConsensus(params []interface{}) map[string]interface{} {
	if err := dBFT.Halt(); err != nil {
		return UgcNetworkRpcFailed
	}
	return UgcNetworkRpcSuccess
}

func sendSampleTransaction(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var txType string
	switch params[0].(type) {
	case string:
		txType = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}

	issuer, err := account.NewAccount()
	if err != nil {
		return UgcNetworkRpc("Failed to create account")
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
			regTx := NewRegTx(ToHexString(rbuf), i, admin, issuer)
			SignTx(admin, regTx)
			VerifyAndSendTx(regTx)
		}
		return UgcNetworkRpc(fmt.Sprintf("%d transaction(s) was sent", num))
	default:
		return UgcNetworkRpc("Invalid transacion type")
	}
}

func setDebugInfo(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcInvalidParameter
	}
	switch params[0].(type) {
	case float64:
		level := params[0].(float64)
		if err := log.Log.SetDebugLevel(int(level)); err != nil {
			return UgcNetworkRpcInvalidParameter
		}
	default:
		return UgcNetworkRpcInvalidParameter
	}
	return UgcNetworkRpcSuccess
}

func getVersion(params []interface{}) map[string]interface{} {
	return UgcNetworkRpc(config.Version)
}

func uploadDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}

	rbuf := make([]byte, 4)
	rand.Read(rbuf)
	tmpname := hex.EncodeToString(rbuf)

	str := params[0].(string)

	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return UgcNetworkRpcInvalidParameter
	}
	f, err := os.OpenFile(tmpname, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		return UgcNetworkRpcIOError
	}
	defer f.Close()
	f.Write(data)

	refpath, err := AddFileIPFS(tmpname, true)
	if err != nil {
		return UgcNetworkRpcAPIError
	}

	return UgcNetworkRpc(refpath)

}

func regDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var hash Uint256
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return UgcNetworkRpcInvalidParameter
		}
		var txn tx.Transaction
		if err := txn.Deserialize(bytes.NewReader(hex)); err != nil {
			return UgcNetworkRpcInvalidTransaction
		}

		hash = txn.Hash()
		if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
			return UgcNetworkRpcInternalError
		}
	default:
		return UgcNetworkRpcInvalidParameter
	}
	return UgcNetworkRpc(ToHexString(hash.ToArray()))
}

func catDataRecord(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		b, err := hex.DecodeString(str)
		if err != nil {
			return UgcNetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(b))
		if err != nil {
			return UgcNetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UgcNetworkRpcUnknownTransaction
		}
		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)
		//ref := string(record.RecordData[:])
		return UgcNetworkRpc(info)
	default:
		return UgcNetworkRpcInvalidParameter
	}
}

func getDataFile(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	switch params[0].(type) {
	case string:
		str := params[0].(string)
		hex, err := hex.DecodeString(str)
		if err != nil {
			return UgcNetworkRpcInvalidParameter
		}
		var hash Uint256
		err = hash.Deserialize(bytes.NewReader(hex))
		if err != nil {
			return UgcNetworkRpcInvalidTransaction
		}
		tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
		if err != nil {
			return UgcNetworkRpcUnknownTransaction
		}

		tran := TransArryByteToHexString(tx)
		info := tran.Payload.(*DataFileInfo)

		err = GetFileIPFS(info.IPFSPath, info.Filename)
		if err != nil {
			return UgcNetworkRpcAPIError
		}
		//TODO: shoud return download address
		return UgcNetworkRpcSuccess
	default:
		return UgcNetworkRpcInvalidParameter
	}
}

func searchTransactions(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var programHash string
	switch params[0].(type) {
	case string:
		programHash = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}

	resp := make(map[string]string)
	height := ledger.DefaultLedger.GetLocalBlockChainHeight()
	var i uint32
	for i = 1; i <= height; i++ {
		block, err := ledger.DefaultLedger.GetBlockWithHeight(i)
		if err != nil {
			return UgcNetworkRpcInternalError
		}
		// skip the bookkeeping transaction
		for _, t := range block.Transactions[1:] {
			switch t.TxType {
			case tx.RegisterAsset:
				regPayload := t.Payload.(*payload.RegisterAsset)
				controller := ToHexString(regPayload.Controller.ToArray())
				if controller == programHash {
					txHash := t.Hash()
					txid := ToHexString(txHash.ToArray())
					resp[txid] = "registration"
				}
			case tx.IssueAsset:
				for _, v := range t.Outputs {
					regTxn, err := ledger.DefaultLedger.Store.GetTransaction(v.AssetID)
					if err != nil {
						log.Warn("Can not find asset")
						continue

					}
					regPayload := regTxn.Payload.(*payload.RegisterAsset)
					controller := ToHexString(regPayload.Controller.ToArray())
					if controller == programHash {
						txHash := t.Hash()
						txid := ToHexString(txHash.ToArray())
						resp[txid] = "issuance"
					}
				}
			case tx.TransferAsset:
				transferTxnProgram := t.GetPrograms()[0]
				transferTxnProgramHash, _ := ToCodeHash(transferTxnProgram.Code)
				transferTxnProgramHashStr := ToHexString(transferTxnProgramHash.ToArray())
				fmt.Println(transferTxnProgramHashStr)
				if programHash == transferTxnProgramHashStr {
					txHash := t.Hash()
					txid := ToHexString(txHash.ToArray())
					resp[txid] = "transfer"
				}
			default:
				continue
			}
		}
	}

	return UgcNetworkRpc(resp)
}

var walletInstance *account.ClientImpl

func getWalletDir() string {
	home, _ := homedir.Dir()
	return home + "/.wallet/"
}

func createWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return UgcNetworkRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return UgcNetworkRpcInternalError
		}
	}
	walletPath := walletDir + "wallet.dat"
	if FileExisted(walletPath) {
		return UgcNetworkRpcWalletAlreadyExists
	}
	_, err := account.Create(walletPath, password)
	if err != nil {
		return UgcNetworkRpcFailed
	}
	return UgcNetworkRpcSuccess
}

func openWallet(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var password []byte
	switch params[0].(type) {
	case string:
		password = []byte(params[0].(string))
	default:
		return UgcNetworkRpcInvalidParameter
	}
	resp := make(map[string]string)
	walletPath := getWalletDir() + "wallet.dat"
	if !FileExisted(walletPath) {
		resp["success"] = "false"
		resp["message"] = "wallet doesn't exist"
		return UgcNetworkRpc(resp)
	}
	wallet, err := account.Open(walletPath, password)
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "password wrong"
		return UgcNetworkRpc(resp)
	}
	walletInstance = wallet
	programHash, err := wallet.LoadStoredData("ProgramHash")
	if err != nil {
		resp["success"] = "false"
		resp["message"] = "wallet file broken"
		return UgcNetworkRpc(resp)
	}
	resp["success"] = "true"
	resp["message"] = ToHexString(programHash)
	return UgcNetworkRpc(resp)
}

func closeWallet(params []interface{}) map[string]interface{} {
	walletInstance = nil
	return UgcNetworkRpcSuccess
}

func recoverWallet(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UgcNetworkRpcNil
	}
	var privateKey string
	var walletPassword string
	switch params[0].(type) {
	case string:
		privateKey = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case string:
		walletPassword = params[1].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	walletDir := getWalletDir()
	if !FileExisted(walletDir) {
		err := os.MkdirAll(walletDir, 0755)
		if err != nil {
			return UgcNetworkRpcInternalError
		}
	}
	walletName := fmt.Sprintf("wallet-%s-recovered.dat", time.Now().Format("2006-01-02-15-04-05"))
	walletPath := walletDir + walletName
	if FileExisted(walletPath) {
		return UgcNetworkRpcWalletAlreadyExists
	}
	_, err := account.Recover(walletPath, []byte(walletPassword), privateKey)
	if err != nil {
		return UgcNetworkRpc("wallet recovery failed")
	}

	return UgcNetworkRpcSuccess
}

func getWalletKey(params []interface{}) map[string]interface{} {
	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}
	account, _ := walletInstance.GetDefaultAccount()
	encodedPublickKey, _ := account.PublicKey.EncodePoint(true)
	resp := make(map[string]string)
	resp["PublicKey"] = ToHexString(encodedPublickKey)
	resp["PrivateKey"] = ToHexString(account.PrivateKey)
	resp["ProgramHash"] = ToHexString(account.ProgramHash.ToArray())

	return UgcNetworkRpc(resp)
}

func addAccount(params []interface{}) map[string]interface{} {
	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}
	account, err := walletInstance.CreateAccount()
	if err != nil {
		return UgcNetworkRpc("create account error:" + err.Error())
	}

	if err := walletInstance.CreateContract(account); err != nil {
		return UgcNetworkRpc("create contract error:" + err.Error())
	}

	return UgcNetworkRpc(ToHexString(account.ProgramHash.ToArray()))
}

func deleteAccount(params []interface{}) map[string]interface{} {
	if len(params) < 1 {
		return UgcNetworkRpcNil
	}
	var programHash string
	switch params[0].(type) {
	case string:
		programHash = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}
	phBytes, _ := HexToBytes(programHash)
	pbUint160, _ := Uint160ParseFromBytes(phBytes)
	if err := walletInstance.DeleteAccount(pbUint160); err != nil {
		return UgcNetworkRpc("Delete account error:" + err.Error())
	}
	if err := walletInstance.DeleteContract(pbUint160); err != nil {
		return UgcNetworkRpc("Delete contract error:" + err.Error())
	}
	if err := walletInstance.DeleteCoinsData(pbUint160); err != nil {
		return UgcNetworkRpc("Delete coins error:" + err.Error())
	}

	return UgcNetworkRpc(true)
}

func makeRegTxn(params []interface{}) map[string]interface{} {
	if len(params) < 2 {
		return UgcNetworkRpcNil
	}
	var assetName string
	var assetValue float64
	switch params[0].(type) {
	case string:
		assetName = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case float64:
		assetValue = params[1].(float64)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}

	regTxn, err := sdk.MakeRegTransaction(walletInstance, assetName, assetValue)
	if err != nil {
		return UgcNetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(regTxn); errCode != ErrNoError {
		return UgcNetworkRpcInvalidTransaction
	}
	return UgcNetworkRpc(true)
}

func makeIssueTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UgcNetworkRpcNil
	}
	var asset string
	var value float64
	var address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case float64:
		value = params[1].(float64)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}
	assetID, _ := StringToUint256(asset)
	issueTxn, err := sdk.MakeIssueTransaction(walletInstance, assetID, address, value)
	if err != nil {
		return UgcNetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(issueTxn); errCode != ErrNoError {
		return UgcNetworkRpcInvalidTransaction
	}

	return UgcNetworkRpc(true)
}

func makeTransferTxn(params []interface{}) map[string]interface{} {
	if len(params) < 3 {
		return UgcNetworkRpcNil
	}
	var asset string
	var value float64
	var address string
	switch params[0].(type) {
	case string:
		asset = params[0].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	switch params[1].(type) {
	case float64:
		value = params[1].(float64)
	default:
		return UgcNetworkRpcInvalidParameter
	}
	switch params[2].(type) {
	case string:
		address = params[2].(string)
	default:
		return UgcNetworkRpcInvalidParameter
	}

	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}

	batchOut := sdk.BatchOut{
		Address: address,
		Value:   value,
	}
	assetID, _ := StringToUint256(asset)
	txn, err := sdk.MakeTransferTransaction(walletInstance, assetID, batchOut)
	if err != nil {
		return UgcNetworkRpcInternalError
	}

	if errCode := VerifyAndSendTx(txn); errCode != ErrNoError {
		return UgcNetworkRpcInvalidTransaction
	}

	return UgcNetworkRpc(true)
}

func getBalance(params []interface{}) map[string]interface{} {
	if walletInstance == nil {
		return UgcNetworkRpc("open wallet first")
	}
	type AssetInfo struct {
		AssetID string
		Value   string
	}
	balances := make(map[string][]*AssetInfo)
	accounts := walletInstance.GetAccounts()
	coins := walletInstance.GetCoins()
	for _, account := range accounts {
		assetList := []*AssetInfo{}
		programHash := account.ProgramHash
		for _, coin := range coins {
			if programHash == coin.Output.ProgramHash {
				var existed bool
				assetString := ToHexString(coin.Output.AssetID.ToArray())
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

	return UgcNetworkRpc(balances)
}
