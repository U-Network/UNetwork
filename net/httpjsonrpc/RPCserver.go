package httpjsonrpc

import (
	."UNetwork/common/config"
	"UNetwork/common/log"
	"net/http"
	"strconv"
)

const (
	LocalHost = "127.0.0.1"
)

const (
	RPCGetBestBlockHash          = "getbestblockhash"
	RPCGetBlock                  = "getblock"
	RPCGetBlockCount             = "getblockcount"
	RPCGetBlockHash              = "getblockhash"
	RPCGetConnectionCount        = "getconnectioncount"
	RPCGetRawMemPool             = "getrawmempool"
	RPCGetRawTransaction         = "getrawtransaction"
	RPCSendRawTransaction        = "sendrawtransaction"
	RPCGetVersion                = "getversion"
	RPCGetNeighbor               = "getneighbor"
	RPCGetNodeState              = "getnodestate"
	RPCSetDebugInfo              = "setdebuginfo"
	RPCSendToAddress             = "sendtoaddress"
	RPCLockAsset                 = "lockasset"
	RPCCreateMultisigTransaction = "createmultisigtransaction"
	RPCSignMultisigTrasaction    = "signmultisigtransaction"

	RPCRegisterUser = "registeruser"
	RPCPostarticle  = "postarticle"
	RPCReplyArticle = "replyarticle"
	RPCLikeArticle  = "likearticle"
	RPCWithdrawl    = "withdrawal"
	RPCGetUTXOByAddr    = "getutxobyaddr"
	RPCGetUtxoCoins  = "getutxocoins"
	RPCGetLikeArticleAdresslist ="getlikearticleadresslist"
	RPCRegAsset  = "regasset"
	RPCIssueAsset  = "issueasset"
	RPCSendToAddresses            = "sendtoaddresses"
)

func StartRPCServer() {
	log.Debug()
	http.HandleFunc("/", Handle)

	HandleFunc(RPCGetBestBlockHash, getBestBlockHash)
	HandleFunc(RPCGetBlock, getBlock)
	HandleFunc(RPCGetBlockCount, getBlockCount)
	HandleFunc(RPCGetBlockHash, getBlockHash)
	HandleFunc(RPCGetConnectionCount, getConnectionCount)
	HandleFunc(RPCGetRawMemPool, getRawMemPool)
	HandleFunc(RPCGetRawTransaction, getRawTransaction)
	HandleFunc(RPCSendRawTransaction, sendRawTransaction)
	HandleFunc(RPCGetVersion, getVersion)
	HandleFunc(RPCGetNeighbor, getNeighbor)
	HandleFunc(RPCGetNodeState, getNodeState)

	HandleFunc(RPCSetDebugInfo, setDebugInfo)
	HandleFunc(RPCSendToAddress, sendToAddress)
	HandleFunc(RPCLockAsset, lockAsset)

	HandleFunc(RPCCreateMultisigTransaction, createMultisigTransaction)
	HandleFunc(RPCSignMultisigTrasaction, signMultisigTransaction)

	// Following transactions should be passed through
	// sendrawtrasaction interface of restfull interface.
	// Put them here only for forum feature testing.
	HandleFunc(RPCRegisterUser, registerUser)
	//HandleFunc(RPCPostarticle, postArticle)
	HandleFunc(RPCReplyArticle, replyArticle)
	//HandleFunc(RPCLikeArticle, likeArticle)
	HandleFunc(RPCWithdrawl, withdrawal)
	HandleFunc(RPCGetUTXOByAddr, getUtxoByAddr)
	HandleFunc(RPCGetUtxoCoins, getUtxoCoins)
	HandleFunc(RPCGetLikeArticleAdresslist, getLikeArticleAdresslist)
	HandleFunc(RPCRegAsset, regAsset)
	HandleFunc(RPCIssueAsset, issueAsset)
	HandleFunc(RPCSendToAddresses, sendToAddresses)

	err := http.ListenAndServe(LocalHost+":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
