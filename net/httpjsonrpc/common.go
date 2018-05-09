package httpjsonrpc

import (
	. "UNetwork/common"
	"UNetwork/common/log"
	"UNetwork/consensus/dbft"
	. "UNetwork/core/transaction"
	tx "UNetwork/core/transaction"
	. "UNetwork/errors"
	. "UNetwork/net/protocol"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"bytes"
	"UNetwork/core/ledger"
	. "UNetwork/common/config"
	"strconv"
	"UNetwork/smartcontract/states"
	"time"
)

const (
	AttributeMaxLen = 252
	MinChatMsgLen   = 1
	MaxChatMsgLen   = 1024
)
const TlsPort int = 443

var oauthClient = NewOauthClient()

func init() {
	mainMux.m = make(map[string]func([]interface{}) map[string]interface{})
}

//an instance of the multiplexer
var mainMux ServeMux
var node UNode
var dBFT *dbft.DbftService

//multiplexer that keeps track of every function to be called on specific rpc call
type ServeMux struct {
	sync.RWMutex
	m               map[string]func([]interface{}) map[string]interface{}
	defaultFunction func(http.ResponseWriter, *http.Request)
}

type TxAttributeInfo struct {
	Usage TransactionAttributeUsage
	Data  string
}

type UTXOTxInputInfo struct {
	ReferTxID          string
	ReferTxOutputIndex uint16
}

type BalanceTxInputInfo struct {
	AssetID     string
	Value       string
	ProgramHash string
}

type TxoutputInfo struct {
	AssetID string
	Value   string
	Address string
}

type TxoutputMap struct {
	Key   Uint256
	Txout []TxoutputInfo
}

type AmountMap struct {
	Key   Uint256
	Value Fixed64
}

type ProgramInfo struct {
	Code      string
	Parameter string
}

type Transactions struct {
	TxType         TransactionType
	PayloadVersion byte
	Payload        PayloadInfo
	Attributes     []TxAttributeInfo
	UTXOInputs     []UTXOTxInputInfo
	BalanceInputs  []BalanceTxInputInfo
	Outputs        []TxoutputInfo
	Programs       []ProgramInfo

	AssetOutputs      []TxoutputMap
	AssetInputAmount  []AmountMap
	AssetOutputAmount []AmountMap

	Hash string
}

type BlockHead struct {
	Version          uint32
	PrevBlockHash    string
	TransactionsRoot string
	Timestamp        uint32
	Height           uint32
	ConsensusData    uint64
	NextBookKeeper   string
	Program          ProgramInfo

	Hash string
}

type BlockInfo struct {
	Hash         string
	BlockData    *BlockHead
	Transactions []*Transactions
}

type TxInfo struct {
	Hash string
	Hex  string
	Tx   *Transactions
}

type TxoutInfo struct {
	High  uint32
	Low   uint32
	Txout tx.TxOutput
}

type NodeInfo struct {
	State    uint   // node status
	Port     uint16 // The nodes's port
	ID       uint64 // The nodes's id
	Time     int64
	Version  uint32 // The network protocol the node used
	Services uint64 // The services the node supplied
	Relay    bool   // The relay capability of the node (merge into capbility flag)
	Height   uint64 // The node latest block height
	TxnCnt   uint64 // The transactions be transmit by this node
	RxTxnCnt uint64 // The transaction received by this node
}

type ConsensusInfo struct {
	// TODO
}

func RegistRpcNode(n UNode) {
	if node == nil {
		node = n
	}
}

func RegistDbftService(d *dbft.DbftService) {
	if dBFT == nil {
		dBFT = d
	}
}

//a function to register functions to be called for specific rpc calls
func HandleFunc(pattern string, handler func([]interface{}) map[string]interface{}) {
	mainMux.Lock()
	defer mainMux.Unlock()
	mainMux.m[pattern] = handler
}

//a function to be called if the request is not a HTTP JSON RPC call
func SetDefaultFunc(def func(http.ResponseWriter, *http.Request)) {
	mainMux.defaultFunction = def
}

//this is the funciton that should be called in order to answer an rpc call
//should be registered like "http.HandleFunc("/", httpjsonrpc.Handle)"
func Handle(w http.ResponseWriter, r *http.Request) {
	mainMux.RLock()
	defer mainMux.RUnlock()

	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("content-type", "application/json;charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	//JSON RPC commands should be POSTs
	if r.Method != "POST" {
		if mainMux.defaultFunction != nil {
			log.Info("HTTP JSON RPC Handle - Method!=\"POST\"")
			mainMux.defaultFunction(w, r)
			return
		} else {
			log.Warn("HTTP JSON RPC Handle - Method!=\"POST\"")
			return
		}
	}

	//check if there is Request Body to read
	if r.Body == nil {
		if mainMux.defaultFunction != nil {
			log.Info("HTTP JSON RPC Handle - Request body is nil")
			mainMux.defaultFunction(w, r)
			return
		} else {
			log.Warn("HTTP JSON RPC Handle - Request body is nil")
			return
		}
	}

	//read the body of the request
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Error("HTTP JSON RPC Handle - ioutil.ReadAll: ", err)
		return
	}
	request := make(map[string]interface{})
	err = json.Unmarshal(body, &request)
	if err != nil {
		log.Error("HTTP JSON RPC Handle - json.Unmarshal: ", err)
		return
	}

	//get the corresponding function
	function, ok := mainMux.m[request["method"].(string)]
	if ok {
		response := function(request["params"].([]interface{}))
		data, err := json.Marshal(map[string]interface{}{
			"jsonpc": "2.0",
			"result": response["result"],
			"id":     request["id"],
		})
		if err != nil {
			log.Error("HTTP JSON RPC Handle - json.Marshal: ", err)
			return
		}
		w.Write(data)
	} else {
		//if the function does not exist
		log.Warn("HTTP JSON RPC Handle - No function to call for ", request["method"])
		data, err := json.Marshal(map[string]interface{}{
			"result": nil,
			"error": map[string]interface{}{
				"code":    -32601,
				"message": "Method not found",
				"data":    "The called method was not found on the server",
			},
			"id": request["id"],
		})
		if err != nil {
			log.Error("HTTP JSON RPC Handle - json.Marshal: ", err)
			return
		}
		w.Write(data)
	}
}

func responsePacking(result interface{}) map[string]interface{} {
	resp := map[string]interface{}{
		"result": result,
	}
	return resp
}

// Call sends RPC request to server
func Call(address string, method string, id interface{}, params []interface{}) ([]byte, error) {
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     id,
		"params": params,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Marshal JSON request: %v\n", err)
		return nil, err
	}
	resp, err := http.Post(address, "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GET response: %v\n", err)
		return nil, err
	}

	return body, nil
}

func VerifyAndSendTx(txn *tx.Transaction) ErrCode {
	// if transaction is verified unsucessfully then will not put it into transaction pool
	if errCode := node.AppendTxnPool(txn, true); errCode != ErrNoError {
		log.Warn("Can NOT add the transaction to TxnPool")
		log.Info("[httpjsonrpc] VerifyTransaction failed when AppendTxnPool.")
		return errCode
	}
	if err := node.Xmit(txn); err != nil {
		log.Error("Xmit Tx Error:Xmit transaction failed.", err)
		return ErrXmitFail
	}
	return ErrNoError
}


func OauthRequest(method string, cmd map[string]interface{}, url string) (map[string]interface{}, error) {

	var repMsg = make(map[string]interface{})
	var response *http.Response
	var err error
	switch method {
	case "GET":

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return repMsg, err
		}
		response, err = oauthClient.Do(req)

	case "POST":
		data, err := json.Marshal(cmd)
		if err != nil {
			return repMsg, err
		}
		reqData := bytes.NewBuffer(data)
		req, err := http.NewRequest("POST", url, reqData)
		if err != nil {
			return repMsg, err
		}
		req.Header.Set("Content-type", "application/json")
		response, err = oauthClient.Do(req)
	default:
		return repMsg, err
	}
	if response != nil {
		defer response.Body.Close()

		body, _ := ioutil.ReadAll(response.Body)
		if err := json.Unmarshal(body, &repMsg); err == nil {
			return repMsg, err
		}
	}
	if err != nil {
		return repMsg, err
	}

	return repMsg, err
}

func CheckAccessToken(auth_type, access_token string) (cakey string, errCode int64, result interface{}) {

	if len(Parameters.OauthServerUrl) == 0 {
		return "", SUCCESS, ""
	}
	req := make(map[string]interface{})
	req["token"] = access_token
	req["auth_type"] = auth_type
	rep, err := OauthRequest("GET", req, Parameters.OauthServerUrl+"/"+access_token+"?auth_type="+auth_type)
	if err != nil {
		log.Error("Oauth timeout:", err)
		return "", OAUTH_TIMEOUT, rep
	}
	if errcode, ok := rep["Error"].(float64); ok && errcode == 0 {
		result, ok := rep["Result"].(map[string]interface{})
		if !ok {
			return "", INVALID_TOKEN, rep
		}
		if CAkey, ok := result["CaKey"].(string); ok {
			return CAkey, SUCCESS, rep
		}
	}
	return "", INVALID_TOKEN, rep
}


func ResponsePack(errCode int64) map[string]interface{} {
	resp := map[string]interface{}{
		"Action":  "",
		"Result":  "",
		"Error":   errCode,
		"Desc":    "",
		"Version": "1.0.0",
	}
	return resp
}

func GetBlockInfo(block *ledger.Block) BlockInfo {
	hash := block.Hash()
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
	return b
}

func GetBlockTransactions(block *ledger.Block) interface{} {
	trans := make([]string, len(block.Transactions))
	for i := 0; i < len(block.Transactions); i++ {
		h := block.Transactions[i].Hash()
		trans[i] = BytesToHexString(h.ToArrayReverse())
	}
	hash := block.Hash()
	type BlockTransactions struct {
		Hash         string
		Height       uint32
		Transactions []string
	}
	b := BlockTransactions{
		Hash:         BytesToHexString(hash.ToArrayReverse()),
		Height:       block.Blockdata.Height,
		Transactions: trans,
	}
	return b
}

func getBlock_Err(hash Uint256, getTxBytes bool) (interface{}, int64) {
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		return "", UNKNOWN_BLOCK
	}
	if getTxBytes {
		w := bytes.NewBuffer(nil)
		block.Serialize(w)
		return BytesToHexString(w.Bytes()), SUCCESS
	}
	return GetBlockInfo(block), SUCCESS
}

func SendRawTransaction(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)

	str, ok := cmd["Data"].(string)
	if !ok {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	bys, err := HexStringToBytes(str)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var txn tx.Transaction
	if err := txn.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = INVALID_TRANSACTION
		return resp
	}
	if txn.TxType != tx.TransferAsset && txn.TxType != tx.RegisterUser &&
		txn.TxType != tx.PostArticle && txn.TxType != tx.ReplyArticle &&
		txn.TxType != tx.LikeArticle && txn.TxType != tx.Withdrawal {
		resp["Error"] = INVALID_TRANSACTION
		return resp
	}
	var hash Uint256
	hash = txn.Hash()
	if errCode := VerifyAndSendTx(&txn); errCode != ErrNoError {
		resp["Error"] = int64(errCode)
		return resp
	}
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	//TODO 0xd1 -> tx.InvokeCode
	if txn.TxType == 0xd1 {
		if userid, ok := cmd["Userid"].(string); ok && len(userid) > 0 {
			resp["Userid"] = userid
		}
	}
	return resp
}

func GetConnectionCount(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	if node != nil {
		resp["Result"] = node.GetConnectionCnt()
	}

	return resp
}

//Block
func GetBlockHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	resp["Result"] = ledger.DefaultLedger.Blockchain.BlockHeight
	return resp
}
func GetBlockHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(uint32(height))
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	return resp
}
func GetTotalIssued(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	assetid, ok := cmd["Assetid"].(string)
	if !ok {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var assetHash Uint256

	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	amount, err := ledger.DefaultLedger.Store.GetQuantityIssued(assetHash)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	resp["Result"] = amount.String()
	return resp
}


func GetBlockByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	param := cmd["Hash"].(string)
	if len(param) == 0 {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	var hash Uint256
	hex, err := HexStringToBytesReverse(param)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	if err := hash.Deserialize(bytes.NewReader(hex)); err != nil {
		resp["Error"] = INVALID_TRANSACTION
		return resp
	}

	resp["Result"], resp["Error"] = getBlock_Err(hash, getTxBytes)

	return resp
}
func GetBlockTxsByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = UNKNOWN_BLOCK
		return resp
	}
	block, err := ledger.DefaultLedger.Store.GetBlock(hash)
	if err != nil {
		resp["Error"] = UNKNOWN_BLOCK
		return resp
	}
	resp["Result"] = GetBlockTransactions(block)
	return resp
}
func GetBlockByHeight(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)

	param := cmd["Height"].(string)
	if len(param) == 0 {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var getTxBytes bool = false
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		getTxBytes = true
	}
	height, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	index := uint32(height)
	hash, err := ledger.DefaultLedger.Store.GetBlockHash(index)
	if err != nil {
		resp["Error"] = UNKNOWN_BLOCK
		return resp
	}
	resp["Result"], resp["Error"] = getBlock_Err(hash, getTxBytes)
	return resp
}

//Asset
func GetAssetByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)

	str := cmd["Hash"].(string)
	hex, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(hex))
	if err != nil {
		resp["Error"] = INVALID_ASSET
		return resp
	}
	asset, err := ledger.DefaultLedger.Store.GetAsset(hash)
	if err != nil {
		resp["Error"] = UNKNOWN_ASSET
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		asset.Serialize(w)
		resp["Result"] = BytesToHexString(w.Bytes())
		return resp
	}
	resp["Result"] = asset
	return resp
}
func GetBalanceByAddr(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	addr, ok := cmd["Addr"].(string)
	if !ok {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	unspends, err := ledger.DefaultLedger.Store.GetUnspentsFromProgramHash(programHash)
	var balance Fixed64 = 0
	for _, u := range unspends {
		for _, v := range u {
			balance = balance + v.Value
		}
	}
	resp["Result"] = balance.String()
	return resp
}

func GetLockedAsset(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	addr, a := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !a || !k {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var programHash Uint160
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	tmpID, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	asset, err := Uint256ParseFromBytes(tmpID)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	type locked struct {
		Lock   uint32
		Unlock uint32
		Amount string
	}
	ret := []*locked{}
	lockedAsset, _ := ledger.DefaultLedger.Store.GetLockedFromProgramHash(programHash, asset)
	for _, v := range lockedAsset {
		a := &locked{
			Lock:   v.Lock,
			Unlock: v.Unlock,
			Amount: v.Amount.String(),
		}
		ret = append(ret, a)
	}
	resp["Result"] = ret

	return resp
}

//Transaction
func GetTransactionByHash(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)

	str := cmd["Hash"].(string)
	bys, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var hash Uint256
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = INVALID_TRANSACTION
		return resp
	}
	tx, err := ledger.DefaultLedger.Store.GetTransaction(hash)
	if err != nil {
		resp["Error"] = UNKNOWN_TRANSACTION
		return resp
	}
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		w := bytes.NewBuffer(nil)
		tx.Serialize(w)
		resp["Result"] = BytesToHexString(w.Bytes())
		return resp
	}
	tran := TransArryByteToHexString(tx)
	resp["Result"] = tran
	return resp
}

func GetContract(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	str := cmd["Hash"].(string)
	bys, err := HexStringToBytesReverse(str)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	var hash Uint160
	err = hash.Deserialize(bytes.NewReader(bys))
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	//TODO GetContract from store
	contract, err := ledger.DefaultLedger.Store.GetContract(hash)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	c := new(states.ContractState)
	b := bytes.NewBuffer(contract)
	c.Deserialize(b)
	var params []int
	for _, v := range c.Code.ParameterTypes {
		params = append(params, int(v))
	}
	codehash := c.Code.CodeHash()
	funcCode := &FunctionCodeInfo{
		Code:           BytesToHexString(c.Code.Code),
		ParameterTypes: params,
		ReturnType:     int(c.Code.ReturnType),
		CodeHash:       BytesToHexString(codehash.ToArrayReverse()),
	}
	programHash := c.ProgramHash
	result := DeployCodeInfo{
		Name:        c.Name,
		Author:      c.Author,
		Email:       c.Email,
		Version:     c.Version,
		Description: c.Description,
		Language:    int(c.Language),
		Code:        new(FunctionCodeInfo),
		ProgramHash: BytesToHexString(programHash.ToArrayReverse()),
	}

	result.Code = funcCode
	resp["Result"] = result
	return resp
}

func GetUnspendOutput(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	addr, ok := cmd["Addr"].(string)
	assetid, k := cmd["Assetid"].(string)
	if !ok || !k {
		resp["Error"] = INVALID_PARAMS
		return resp
	}

	var programHash Uint160
	var assetHash Uint256
	programHash, err := ToScriptHash(addr)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	bys, err := HexStringToBytesReverse(assetid)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	if err := assetHash.Deserialize(bytes.NewReader(bys)); err != nil {
		resp["Error"] = INVALID_PARAMS
		return resp
	}
	type UTXOUnspentInfo struct {
		Txid  string
		Index uint32
		Value string
	}
	infos, err := ledger.DefaultLedger.Store.GetUnspentFromProgramHash(programHash, assetHash)
	if err != nil {
		resp["Error"] = INVALID_PARAMS
		resp["Result"] = err
		return resp
	}
	var UTXOoutputs []UTXOUnspentInfo
	for _, v := range infos {
		UTXOoutputs = append(UTXOoutputs, UTXOUnspentInfo{Txid: BytesToHexString(v.Txid.ToArrayReverse()), Index: v.Index, Value: v.Value.String()})
	}
	resp["Result"] = UTXOoutputs
	return resp
}

//record
func getRecordData(cmd map[string]interface{}) ([]byte, int64) {
	if raw, ok := cmd["Raw"].(string); ok && raw == "1" {
		str, ok := cmd["RecordData"].(string)
		if !ok {
			return nil, INVALID_PARAMS
		}
		bys, err := HexStringToBytes(str)
		if err != nil {
			return nil, INVALID_PARAMS
		}
		return bys, SUCCESS
	}
	type Data struct {
		Algrithem string `json:Algrithem`
		Hash      string `json:Hash`
		Signature string `json:Signature`
		Text      string `json:Text`
	}
	type RecordData struct {
		CAkey     string  `json:CAkey`
		Data      Data    `json:Data`
		SeqNo     string  `json:SeqNo`
		Timestamp float64 `json:Timestamp`
	}

	tmp := &RecordData{}
	reqRecordData, ok := cmd["RecordData"].(map[string]interface{})
	if !ok {
		return nil, INVALID_PARAMS
	}
	reqBtys, err := json.Marshal(reqRecordData)
	if err != nil {
		return nil, INVALID_PARAMS
	}

	if err := json.Unmarshal(reqBtys, tmp); err != nil {
		return nil, INVALID_PARAMS
	}
	tmp.CAkey, ok = cmd["CAkey"].(string)
	if !ok {
		return nil, INVALID_PARAMS
	}
	repBtys, err := json.Marshal(tmp)
	if err != nil {
		return nil, INVALID_PARAMS
	}
	return repBtys, SUCCESS
}

func getInnerTimestamp() ([]byte, int64) {
	type InnerTimestamp struct {
		InnerTimestamp float64 `json:InnerTimestamp`
	}
	tmp := &InnerTimestamp{InnerTimestamp: float64(time.Now().Unix())}
	repBtys, err := json.Marshal(tmp)
	if err != nil {
		return nil, INVALID_PARAMS
	}
	return repBtys, SUCCESS
}

func SendRecord(cmd map[string]interface{}) map[string]interface{} {
	resp := ResponsePack(SUCCESS)
	var recordData []byte
	var innerTime []byte
	innerTime, resp["Error"] = getInnerTimestamp()
	if innerTime == nil {
		return resp
	}
	recordData, resp["Error"] = getRecordData(cmd)
	if recordData == nil {
		return resp
	}

	var inputs []*tx.UTXOTxInput
	var outputs []*tx.TxOutput

	transferTx, _ := tx.NewTransferAssetTransaction(inputs, outputs)

	rcdInner := tx.NewTxAttribute(tx.Description, innerTime)
	transferTx.Attributes = append(transferTx.Attributes, &rcdInner)

	bytesBuf := bytes.NewBuffer(recordData)

	buf := make([]byte, AttributeMaxLen)
	for {
		n, err := bytesBuf.Read(buf)
		if err != nil {
			break
		}
		var data = make([]byte, n)
		copy(data, buf[0:n])
		record := tx.NewTxAttribute(tx.Description, data)
		transferTx.Attributes = append(transferTx.Attributes, &record)
	}
	if errCode := VerifyAndSendTx(transferTx); errCode != ErrNoError {
		resp["Error"] = int64(errCode)
		return resp
	}
	hash := transferTx.Hash()
	resp["Result"] = BytesToHexString(hash.ToArrayReverse())
	return resp
}