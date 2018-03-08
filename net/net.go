package net

import (
	. "UNetwork/common"
	"UNetwork/core/ledger"
	"UNetwork/core/transaction"
	"UNetwork/crypto"
	. "UNetwork/errors"
	"UNetwork/events"
	"UNetwork/net/node"
	"UNetwork/net/protocol"
)

type Neter interface {
	GetTxnPool(byCount bool) map[Uint256]*transaction.Transaction
	Xmit(interface{}) error
	GetEvent(eventName string) *events.Event
	GetBookKeepersAddrs() ([]*crypto.PubKey, uint64)
	CleanSubmittedTransactions(block *ledger.Block) error
	GetNeighborUNode() []protocol.UNode
	Tx(buf []byte)
	AppendTxnPool(*transaction.Transaction, bool) ErrCode
}

func StartProtocol(pubKey *crypto.PubKey) protocol.UNode {
	net := node.InitNode(pubKey)
	net.ConnectSeeds()

	return net
}
