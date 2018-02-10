package httpjsonrpc

import (
	. "UNetwork/common/config"
	"UNetwork/common/log"
	"net/http"
	"strconv"
)

const (
	LocalHost = "127.0.0.1"
)

func StartRPCServer() {
	log.Debug()
	http.HandleFunc("/", Handle)

	HandleFunc("getbestblockhash", getBestBlockHash)
	HandleFunc("getblock", getBlock)
	HandleFunc("getblockcount", getBlockCount)
	HandleFunc("getblockhash", getBlockHash)
	HandleFunc("getconnectioncount", getConnectionCount)
	HandleFunc("getrawmempool", getRawMemPool)
	HandleFunc("getrawtransaction", getRawTransaction)
	HandleFunc("sendrawtransaction", sendRawTransaction)
	HandleFunc("getversion", getVersion)
	HandleFunc("getneighbor", getNeighbor)
	HandleFunc("getnodestate", getNodeState)

	HandleFunc("setdebuginfo", setDebugInfo)
	HandleFunc("sendtoaddress", sendToAddress)
	HandleFunc("lockasset", lockAsset)

	HandleFunc("createmultisigtransaction", createMultisigTransaction)
	HandleFunc("signmultisigtransaction", signMultisigTransaction)

	// Following transactions should be passed through
	// sendrawtrasaction interface of restfull interface.
	// Put them here only for forum feature testing.
	HandleFunc("registeruser", registerUser)
	HandleFunc("postarticle", postArticle)
	HandleFunc("replyarticle", replyArticle)
	HandleFunc("likearticle", likeArticle)
	HandleFunc("withdrawal", withdrawal)

	err := http.ListenAndServe(LocalHost+":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
