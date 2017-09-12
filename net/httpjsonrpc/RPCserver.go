package httpjsonrpc

import (
	. "UGCNetwork/common/config"
	"UGCNetwork/common/log"
	"net/http"
	"strconv"
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
	HandleFunc("submitblock", submitBlock)
	HandleFunc("getversion", getVersion)
	HandleFunc("getdataile", getDataFile)
	HandleFunc("catdatarecord", catDataRecord)
	HandleFunc("regdatafile", regDataFile)
	HandleFunc("uploadDataFile", uploadDataFile)

	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpJsonPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
