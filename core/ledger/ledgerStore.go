package ledger

import (
	. "UNetwork/common"
	"UNetwork/core/account"
	. "UNetwork/core/asset"
	"UNetwork/core/forum"
	tx "UNetwork/core/transaction"
	"UNetwork/crypto"
	"UNetwork/smartcontract/states"
	"UNetwork/core/transaction/payload"
)

// ILedgerStore provides func with store package.
type ILedgerStore interface {
	//TODO: define the state store func
	SaveBlock(b *Block, ledger *Ledger) error
	GetBlock(hash Uint256) (*Block, error)
	BlockInCache(hash Uint256) bool
	GetBlockHash(height uint32) (Uint256, error)
	InitLedgerStore(ledger *Ledger) error
	IsDoubleSpend(tx *tx.Transaction) bool

	//SaveHeader(header *Header,ledger *Ledger) error
	AddHeaders(headers []Header, ledger *Ledger) error
	GetHeader(hash Uint256) (*Header, error)

	GetTransaction(hash Uint256) (*tx.Transaction, error)

	SaveAsset(assetid Uint256, asset *Asset) error
	GetAsset(hash Uint256) (*Asset, error)

	GetContract(codeHash Uint160) ([]byte, error)
	GetStorage(key []byte) ([]byte, error)
	GetAccount(programHash Uint160) (*account.AccountState, error)
	GetAssetState(assetId Uint256) (*states.AssetState, error)

	GetUserInfo(name string) (*payload.RegisterUser, error)
	GetLikeInfo(articlehash Uint256) ([]*payload.LikeArticle, error)
	GetArticleInfo(articlehash Uint256) (payload.ArticleInfo, error)
	GetUserArticleInfo(name string) ([]Uint256, error)
	GetTokenInfo(name string, tokenType forum.TokenType) (*forum.TokenInfo, error)
	GetAvailableTokenInfo(name string) (*forum.TokenInfo, error)

	GetCurrentBlockHash() Uint256
	GetCurrentHeaderHash() Uint256
	GetHeaderHeight() uint32
	GetHeight() uint32
	GetHeaderHashByHeight(height uint32) Uint256

	GetBookKeeperList() ([]*crypto.PubKey, []*crypto.PubKey, error)
	InitLedgerStoreWithGenesisBlock(genesisblock *Block, defaultBookKeeper []*crypto.PubKey) (uint32, error)

	GetQuantityIssued(assetid Uint256) (Fixed64, error)

	GetUnspent(txid Uint256, index uint16) (*tx.TxOutput, error)
	ContainsUnspent(txid Uint256, index uint16) (bool, error)
	GetUnspentFromProgramHash(programHash Uint160, assetid Uint256) ([]*tx.UTXOUnspent, error)
	GetUnspentsFromProgramHash(programHash Uint160) (map[Uint256][]*tx.UTXOUnspent, error)
	GetUnspentOutputFromProgramHash(programHash Uint160) (map[*tx.UTXOTxInput]*tx.TxOutput, error)
	GetLockedFromProgramHash(programHash Uint160, assetid Uint256) ([]*LockAsset, error)
	GetAvailableAsset(programHash Uint160, assetid Uint256) (Fixed64, Fixed64, error)
	GetAssets() map[Uint256]*Asset

	IsTxHashDuplicate(txhash Uint256) bool
	IsBlockInStore(hash Uint256) bool
	Close()
}
