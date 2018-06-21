package transaction

import (
	. "UNetwork/common"
	"UNetwork/core/transaction/payload"
)

// ILedgerStore provides func with store package.
type ILedgerStore interface {
	GetTransaction(hash Uint256) (*Transaction, error)
	GetQuantityIssued(AssetId Uint256) (Fixed64, error)
	GetUserInfo(name string) (*payload.RegisterUser, error)
}
