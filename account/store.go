package account

import (
	. "UNetwork/common"
	ct "UNetwork/core/contract"
)

type IClientStore interface {
	BuildDatabase(path string)

	SaveStoredData(name string, value []byte)
	LoadStoredData(name string) []byte

	CreateAccount() (*Account, error)
	CreateAccountByPrivateKey(privateKey []byte) (*Account, error)
	LoadAccounts() map[Uint160]*Account

	CreateContract(account *Account) error
	LoadContracts() map[Uint160]*ct.Contract

	SaveHeight(height uint32) error
	LoadHeight() (uint32, error)
}
