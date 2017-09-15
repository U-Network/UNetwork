package net

import (
	. "UGCNetwork/common"
	"UGCNetwork/core/ledger"
	"UGCNetwork/core/transaction"
	"UGCNetwork/crypto"
	. "UGCNetwork/errors"
	"UGCNetwork/events"
	"UGCNetwork/net/node"
	"UGCNetwork/net/protocol"
)

type Neter interface {
	GetTxnPool(byCount bool) map[Uint256]*transaction.Transaction
	Xmit(interface{}) error
	GetEvent(eventName string) *events.Event
	GetBookKeepersAddrs() ([]*crypto.PubKey, uint64)
	CleanSubmittedTransactions(block *ledger.Block) error
	GetNeighborNoder() []protocol.Noder
	Tx(buf []byte)
	AppendTxnPool(*transaction.Transaction) ErrCode
}

func StartProtocol(pubKey *crypto.PubKey) protocol.Noder {
	net := node.InitNode(pubKey)
	net.ConnectSeeds()

	return net
}
