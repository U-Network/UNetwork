package ethereum

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"log"
)

type TxListenPushBroadcastTransaction interface {
	BroadcastTransaction([]byte)
}

// TxListenPush Used to subscribe to the transaction in the Ethereum txpool and push the transaction to the tendermint txpool
type TxListenPush struct {
	ethBackend *eth.Ethereum
	ethTxPool  *core.TxPool
	newTxEv    chan core.NewTxsEvent
	hook       TxListenPushBroadcastTransaction
	isStart    bool
}

//NewTxListenPush Create and return a TxListenPush object pointer
func NewTxListenPush(eth *eth.Ethereum, hook TxListenPushBroadcastTransaction) *TxListenPush {
	e := new(TxListenPush)
	e.ethBackend = eth
	e.ethTxPool = eth.TxPool()
	e.hook = hook
	return e
}

// Start Subscribe to the new deal in Ethereum TXpool and launch a go coroutine
func (e *TxListenPush) Start() {
	if e.isStart {
		return
	}
	e.newTxEv = make(chan core.NewTxsEvent)
	sub := e.ethBackend.TxPool().SubscribeNewTxsEvent(e.newTxEv)
	go e.loop(sub)
	e.isStart = true
}

// loop Is a go coroutine function, mainly used to loop to get new transactions and push to tendermint txpool
func (e *TxListenPush) loop(ev event.Subscription) {
	for {
		select {
		case tx := <-e.newTxEv:
			for i := 0; i < len(tx.Txs); i++ {
				by, _ := tx.Txs[i].MarshalJSON()
				fmt.Println("loop json: ", string(by))
				buf := new(bytes.Buffer)
				if err := tx.Txs[i].EncodeRLP(buf); err == nil {
					fmt.Println("loop ===================this is shi=========================")
					e.hook.BroadcastTransaction(buf.Bytes())

					// UNetwork test start
					// target: to debug the trouble of txdata.
					var des []byte = make([]byte, 4096)
					base64.StdEncoding.Encode(des, buf.Bytes())
					log.Println("this is loop tx: ", string(des))
					// UNetwork test stop
				} else {
					log.Printf("Marshal Transaction error: %s", err.Error())
					continue
				}
			}
		case <-ev.Err():
			ev.Unsubscribe()
		}
	}
}
