package store

// DataEntryPrefix
type DataEntryPrefix byte

const (
	// DATA
	DATA_BlockHash   DataEntryPrefix = 0x00
	DATA_Header      DataEntryPrefix = 0x01
	DATA_Transaction DataEntryPrefix = 0x02
	DATA_Contract    DataEntryPrefix = 0x03

	// INDEX
	IX_HeaderHashList DataEntryPrefix = 0x80
	IX_Enrollment     DataEntryPrefix = 0x84
	IX_Unspent        DataEntryPrefix = 0x90
	IX_Unspent_UTXO   DataEntryPrefix = 0x91
	IX_Vote           DataEntryPrefix = 0x94

	// ASSET
	ST_Info           DataEntryPrefix = 0xc0
	ST_QuantityIssued DataEntryPrefix = 0xc1
	ST_Contract       DataEntryPrefix = 0xc2
	ST_Storage        DataEntryPrefix = 0xc3
	ST_ACCOUNT        DataEntryPrefix = 0xc4
	ST_AssetState     DataEntryPrefix = 0xc5
	ST_Validator      DataEntryPrefix = 0xc6
	ST_Locked         DataEntryPrefix = 0xc7
	ST_User           DataEntryPrefix = 0xc8 // key: username 			value: program hash + reputation
	ST_Post           DataEntryPrefix = 0xc9 // key: username 			value: all post and reply transaction hash
	ST_Like           DataEntryPrefix = 0xca // key: post/reply txid	value: liker/disliker
	ST_Withdraw       DataEntryPrefix = 0xcb // key: username   		value: withdrawn token
	ST_Total          DataEntryPrefix = 0xcc // key: username   		value: total token

	//SYSTEM
	SYS_CurrentBlock DataEntryPrefix = 0x40
	// SYS_CurrentHeader     DataEntryPrefix = 0x41
	SYS_CurrentBookKeeper DataEntryPrefix = 0x42

	//CONFIG
	CFG_Version DataEntryPrefix = 0xf0
)
