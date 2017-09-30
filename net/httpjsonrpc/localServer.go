package httpjsonrpc

import (
	"net/http"
	"strconv"

	. "UGCNetwork/common/config"
	"UGCNetwork/common/log"
)

const (
	localHost string = "127.0.0.1"
	LocalDir  string = "/local"
)

func StartLocalServer() {
	log.Debug()
	http.HandleFunc(LocalDir, Handle)

	HandleFunc("getneighbor", getNeighbor)
	HandleFunc("getnodestate", getNodeState)
	HandleFunc("setdebuginfo", setDebugInfo)

	//HandleFunc("sendsampletransaction", sendSampleTransaction)
	//HandleFunc("startconsensus", startConsensus)
	//HandleFunc("stopconsensus", stopConsensus)
	//HandleFunc("createwallet", createWallet)
	//HandleFunc("openwallet", openWallet)
	//HandleFunc("closewallet", closeWallet)
	//HandleFunc("recoverwallet", recoverWallet)
	//HandleFunc("getwalletkey", getWalletKey)
	//HandleFunc("makeregtxn", makeRegTxn)
	//HandleFunc("makeissuetxn", makeIssueTxn)
	//HandleFunc("maketransfertxn", makeTransferTxn)
	//HandleFunc("addaccount", addAccount)
	//HandleFunc("deleteaccount", deleteAccount)
	//HandleFunc("getbalance", getBalance)
	//HandleFunc("searchtransactions", searchTransactions)

	// TODO: only listen to local host
	err := http.ListenAndServe(":"+strconv.Itoa(Parameters.HttpLocalPort), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err.Error())
	}
}
