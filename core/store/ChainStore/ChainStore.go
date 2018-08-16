package ChainStore

import (
	"bytes"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
    "UNetwork/errors"
	. "UNetwork/common"
	"UNetwork/common/log"
	"UNetwork/common/serialization"
	"UNetwork/core/account"
	. "UNetwork/core/asset"
	"UNetwork/core/contract/program"
	"UNetwork/core/forum"
	. "UNetwork/core/ledger"
	. "UNetwork/core/store"
	. "UNetwork/core/store/LevelDBStore"
	tx "UNetwork/core/transaction"
	"UNetwork/core/transaction/payload"
	"UNetwork/core/validation"
	"UNetwork/crypto"
	"UNetwork/events"
	"UNetwork/net/httpwebsocket"
	"UNetwork/smartcontract"
	"UNetwork/smartcontract/service"
	"UNetwork/smartcontract/states"
)

const (
	HeaderHashListCount = 2000
	CleanCacheThreshold = 2
	TaskChanCap         = 4
	DEPLOY_TRANSACTION  = "DeployTransaction"
	INVOKE_TRANSACTION  = "InvokeTransaction"
)

var (
	ErrDBNotFound = errors.NewErr("leveldb: not found")
)

type persistTask interface{}
type persistHeaderTask struct {
	header *Header
}
type persistBlockTask struct {
	block  *Block
	ledger *Ledger
}

type ChainStore struct {
	st IStore

	taskCh chan persistTask
	quit   chan chan bool

	mu          sync.RWMutex // guard the following var
	headerIndex map[uint32]Uint256
	blockCache  map[Uint256]*Block
	headerCache map[Uint256]*Header

	currentBlockHeight uint32
	storedHeaderCount  uint32
}

func NewStore(file string) (IStore, error) {
	ldbs, err := NewLevelDBStore(file)

	return ldbs, err
}

func NewLedgerStore() (ILedgerStore, error) {
	// TODO: read config file decide which db to use.
	cs, err := NewChainStore("Chain")
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func NewChainStore(file string) (*ChainStore, error) {

	st, err := NewStore(file)
	if err != nil {
		return nil, err
	}

	chain := &ChainStore{
		st:                 st,
		headerIndex:        map[uint32]Uint256{},
		blockCache:         map[Uint256]*Block{},
		headerCache:        map[Uint256]*Header{},
		currentBlockHeight: 0,
		storedHeaderCount:  0,
		taskCh:             make(chan persistTask, TaskChanCap),
		quit:               make(chan chan bool, 1),
	}

	go chain.loop()

	return chain, nil
}

func (self *ChainStore) Close() {
	closed := make(chan bool)
	self.quit <- closed
	<-closed

	self.st.Close()
}

func (self *ChainStore) loop() {
	for {
		select {
		case t := <-self.taskCh:
			now := time.Now()
			switch task := t.(type) {
			case *persistHeaderTask:
				self.handlePersistHeaderTask(task.header)
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle header exetime: %g \n", tcall)

			case *persistBlockTask:
				self.handlePersistBlockTask(task.block, task.ledger)
				tcall := float64(time.Now().Sub(now)) / float64(time.Second)
				log.Debugf("handle block exetime: %g num transactions:%d \n", tcall, len(task.block.Transactions))
			}

		case closed := <-self.quit:
			closed <- true
			return
		}
	}
}

// can only be invoked by backend write goroutine
func (self *ChainStore) clearCache() {
	self.mu.Lock()
	defer self.mu.Unlock()

	currBlockHeight := self.currentBlockHeight
	for hash, header := range self.headerCache {
		if header.Blockdata.Height+CleanCacheThreshold < currBlockHeight {
			delete(self.headerCache, hash)
		}
	}

	for hash, block := range self.blockCache {
		if block.Blockdata.Height+CleanCacheThreshold < currBlockHeight {
			delete(self.blockCache, hash)
		}
	}

}

func (bd *ChainStore) InitLedgerStoreWithGenesisBlock(genesisBlock *Block, defaultBookKeeper []*crypto.PubKey) (uint32, error) {

	hash := genesisBlock.Hash()
	bd.headerIndex[0] = hash

	prefix := []byte{byte(CFG_Version)}
	version, err := bd.st.Get(prefix)
	if err != nil {
		version = []byte{0x00}
	}

	if version[0] == 0x01 {
		// GenesisBlock should exist in chain
		// Or the bookkeepers are not consistent with the chain
		if !bd.IsBlockInStore(hash) {
			return 0, errors.NewErr("bookkeepers are not consistent with the chain")
		}
		// Get Current Block
		currentBlockPrefix := []byte{byte(SYS_CurrentBlock)}
		data, err := bd.st.Get(currentBlockPrefix)
		if err != nil {
			return 0, err
		}

		r := bytes.NewReader(data)
		var blockHash Uint256
		blockHash.Deserialize(r)
		bd.currentBlockHeight, err = serialization.ReadUint32(r)
		current_Header_Height := bd.currentBlockHeight

		log.Debugf("blockHash: %x\n", blockHash.ToArray())

		var listHash Uint256
		iter := bd.st.NewIterator([]byte{byte(IX_HeaderHashList)})
		for iter.Next() {
			rk := bytes.NewReader(iter.Key())
			// read prefix
			_, _ = serialization.ReadBytes(rk, 1)
			startNum, err := serialization.ReadUint32(rk)
			if err != nil {
				return 0, err
			}
			log.Debugf("start index: %d\n", startNum)

			r = bytes.NewReader(iter.Value())
			listNum, err := serialization.ReadVarUint(r, 0)
			if err != nil {
				return 0, err
			}

			for i := 0; i < int(listNum); i++ {
				listHash.Deserialize(r)
				bd.headerIndex[startNum+uint32(i)] = listHash
				bd.storedHeaderCount++
				//log.Debug( fmt.Sprintf( "listHash %d: %x\n", startNum+uint32(i), listHash ) )
			}
		}

		if bd.storedHeaderCount == 0 {
			iter = bd.st.NewIterator([]byte{byte(DATA_BlockHash)})
			for iter.Next() {
				rk := bytes.NewReader(iter.Key())
				// read prefix
				_, _ = serialization.ReadBytes(rk, 1)
				listheight, err := serialization.ReadUint32(rk)
				if err != nil {
					return 0, err
				}
				//log.Debug(fmt.Sprintf( "DATA_BlockHash block height: %d\n", listheight ))

				r := bytes.NewReader(iter.Value())
				listHash.Deserialize(r)
				//log.Debug(fmt.Sprintf( "DATA_BlockHash block hash: %x\n", listHash ))

				bd.headerIndex[listheight] = listHash
			}
		} else if current_Header_Height >= bd.storedHeaderCount {
			hash = blockHash
			for {
				if hash == bd.headerIndex[bd.storedHeaderCount-1] {
					break
				}

				header, err := bd.GetHeader(hash)
				if err != nil {
					return 0, err
				}

				//log.Debug(fmt.Sprintf( "header height: %d\n", header.Blockdata.Height ))
				//log.Debug(fmt.Sprintf( "header hash: %x\n", hash ))

				bd.headerIndex[header.Blockdata.Height] = hash
				hash = header.Blockdata.PrevBlockHash
			}
		}

		return bd.currentBlockHeight, nil

	} else {
		// batch delete old data
		bd.st.NewBatch()
		iter := bd.st.NewIterator(nil)
		for iter.Next() {
			bd.st.BatchDelete(iter.Key())
		}
		iter.Release()

		err := bd.st.BatchCommit()
		if err != nil {
			return 0, err
		}

		///////////////////////////////////////////////////
		// process defaultBookKeeper
		///////////////////////////////////////////////////
		// sort defaultBookKeeper
		sort.Sort(crypto.PubKeySlice(defaultBookKeeper))

		// currBookKeeper key
		bkListKey := bytes.NewBuffer(nil)
		bkListKey.WriteByte(byte(SYS_CurrentBookKeeper))

		// currBookKeeper value
		bkListValue := bytes.NewBuffer(nil)
		serialization.WriteUint8(bkListValue, uint8(len(defaultBookKeeper)))
		for k := 0; k < len(defaultBookKeeper); k++ {
			defaultBookKeeper[k].Serialize(bkListValue)
		}

		// nextBookKeeper value
		serialization.WriteUint8(bkListValue, uint8(len(defaultBookKeeper)))
		for k := 0; k < len(defaultBookKeeper); k++ {
			defaultBookKeeper[k].Serialize(bkListValue)
		}

		// defaultBookKeeper put value
		bd.st.Put(bkListKey.Bytes(), bkListValue.Bytes())
		///////////////////////////////////////////////////

		// persist genesis block
		bd.persist(genesisBlock)

		// put version to db
		err = bd.st.Put(prefix, []byte{0x01})
		if err != nil {
			return 0, err
		}

		return 0, nil
	}
}

func (bd *ChainStore) InitLedgerStore(l *Ledger) error {
	// TODO: InitLedgerStore
	return nil
}

func (bd *ChainStore) IsTxHashDuplicate(txhash Uint256) bool {
	prefix := []byte{byte(DATA_Transaction)}
	_, err_get := bd.st.Get(append(prefix, txhash.ToArray()...))
	if err_get != nil {
		return false
	} else {
		return true
	}
}

func (bd *ChainStore) IsDoubleSpend(tx *tx.Transaction) bool {
	if len(tx.UTXOInputs) == 0 {
		return false
	}

	unspentPrefix := []byte{byte(IX_Unspent)}
	for i := 0; i < len(tx.UTXOInputs); i++ {
		txhash := tx.UTXOInputs[i].ReferTxID
		unspentValue, err_get := bd.st.Get(append(unspentPrefix, txhash.ToArray()...))
		if err_get != nil {
			return true
		}

		unspents, _ := GetUint16Array(unspentValue)
		findFlag := false
		for k := 0; k < len(unspents); k++ {
			if unspents[k] == tx.UTXOInputs[i].ReferTxOutputIndex {
				findFlag = true
				break
			}
		}

		if !findFlag {
			return true
		}
	}

	return false
}

func (bd *ChainStore) GetBlockHash(height uint32) (Uint256, error) {
	queryKey := bytes.NewBuffer(nil)
	queryKey.WriteByte(byte(DATA_BlockHash))
	err := serialization.WriteUint32(queryKey, height)

	if err != nil {
		return Uint256{}, err
	}
	blockHash, err_get := bd.st.Get(queryKey.Bytes())
	if err_get != nil {
		//TODO: implement error process
		return Uint256{}, err_get
	}
	blockHash256, err_parse := Uint256ParseFromBytes(blockHash)
	if err_parse != nil {
		return Uint256{}, err_parse
	}

	return blockHash256, nil
}

func (bd *ChainStore) GetCurrentBlockHash() Uint256 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.headerIndex[bd.currentBlockHeight]
}

func (bd *ChainStore) GetContract(codeHash Uint160) ([]byte, error) {
	prefix := []byte{byte(ST_Contract)}
	bData, err_get := bd.st.Get(append(prefix, codeHash.ToArray()...))
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	log.Debug("GetContract Data: ", bData)

	return bData, nil
}

func (bd *ChainStore) getHeaderWithCache(hash Uint256) *Header {
	if _, ok := bd.headerCache[hash]; ok {
		return bd.headerCache[hash]
	}

	header, _ := bd.GetHeader(hash)

	return header
}

func (bd *ChainStore) verifyHeader(header *Header) bool {
	prevHeader := bd.getHeaderWithCache(header.Blockdata.PrevBlockHash)

	if prevHeader == nil {
		log.Error("[verifyHeader] failed, not found prevHeader.")
		return false
	}

	if prevHeader.Blockdata.Height+1 != header.Blockdata.Height {
		log.Error("[verifyHeader] failed, prevHeader.Height + 1 != header.Height")
		return false
	}

	if prevHeader.Blockdata.Timestamp >= header.Blockdata.Timestamp {
		log.Error("[verifyHeader] failed, prevHeader.Timestamp >= header.Timestamp")
		return false
	}

	flag, err := validation.VerifySignableData(header.Blockdata)
	if flag == false || err != nil {
		log.Error("[verifyHeader] failed, VerifySignableData failed.")
		log.Error(err)
		return false
	}

	return true
}

func (self *ChainStore) AddHeaders(headers []Header, ledger *Ledger) error {

	sort.Slice(headers, func(i, j int) bool {
		return headers[i].Blockdata.Height < headers[j].Blockdata.Height
	})

	for i := 0; i < len(headers); i++ {
		self.taskCh <- &persistHeaderTask{header: &headers[i]}
	}

	return nil

}

func (bd *ChainStore) GetHeader(hash Uint256) (*Header, error) {
	bd.mu.RLock()
	if header, ok := bd.headerCache[hash]; ok {
		bd.mu.RUnlock()
		return header, nil
	}
	bd.mu.RUnlock()

	var h *Header = new(Header)

	h.Blockdata = new(Blockdata)
	h.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	log.Debug("GetHeader Data:", hash.ToArray())
	data, err_get := bd.st.Get(append(prefix, hash.ToArray()...))
	//log.Debug( "Get Header Data: %x\n",  data )
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	r := bytes.NewReader(data)

	// first 8 bytes is sys_fee
	sysfee, err := serialization.ReadUint64(r)
	if err != nil {
		return nil, err
	}
	log.Debug(fmt.Sprintf("sysfee: %d\n", sysfee))

	// Deserialize block data
	err = h.Deserialize(r)
	if err != nil {
		return nil, err
	}

	return h, err
}

func (bd *ChainStore) SaveAsset(assetId Uint256, asset *Asset) error {
	w := bytes.NewBuffer(nil)

	asset.Serialize(w)

	// generate key
	assetKey := bytes.NewBuffer(nil)
	// add asset prefix.
	assetKey.WriteByte(byte(ST_Info))
	// contact asset id
	assetId.Serialize(assetKey)

	log.Debug(fmt.Sprintf("asset key: %x\n", assetKey))

	// PUT VALUE
	err := bd.st.BatchPut(assetKey.Bytes(), w.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) SaveLockedAsset(programHash Uint160, assetID Uint256, lockAsset []*LockAsset) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Locked))
	programHash.Serialize(key)
	assetID.Serialize(key)

	value := bytes.NewBuffer(nil)
	serialization.WriteVarUint(value, uint64(len(lockAsset)))
	for _, v := range lockAsset {
		if err := v.Serialize(value); err != nil {
			return err
		}
	}

	if err := bd.st.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) GetAssetState(assetId Uint256) (*states.AssetState, error) {
	assetState := new(states.AssetState)
	data, err := bd.st.Get(append([]byte{byte(ST_AssetState)}, assetId.ToArray()...))
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(data)
	assetState.Deserialize(r)
	return assetState, nil
}

func (bd *ChainStore) GetAsset(hash Uint256) (*Asset, error) {
	log.Debug(fmt.Sprintf("GetAsset Hash: %x\n", hash))

	asset := new(Asset)

	prefix := []byte{byte(ST_Info)}
	data, err_get := bd.st.Get(append(prefix, hash.ToArray()...))

	log.Debug(fmt.Sprintf("GetAsset Data: %x\n", data))
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	r := bytes.NewReader(data)
	asset.Deserialize(r)

	return asset, nil
}

func (bd *ChainStore) GetTransaction(hash Uint256) (*tx.Transaction, error) {
	log.Debugf("GetTransaction Hash: %x\n", hash)

	t := new(tx.Transaction)
	err := bd.getTx(t, hash)

	if err != nil {
		return nil, err
	}

	return t, nil
}

func (bd *ChainStore) getTx(tx *tx.Transaction, hash Uint256) error {
	prefix := []byte{byte(DATA_Transaction)}
	tHash, err_get := bd.st.Get(append(prefix, hash.ToArray()...))
	if err_get != nil {
		//TODO: implement error process
		return err_get
	}

	r := bytes.NewReader(tHash)

	// get height
	_, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}

	// Deserialize Transaction
	err = tx.Deserialize(r)

	return err
}

func (bd *ChainStore) SaveTransaction(tx *tx.Transaction, height uint32) error {
	//////////////////////////////////////////////////////////////
	// generate key with DATA_Transaction prefix
	txhash := bytes.NewBuffer(nil)
	// add transaction header prefix.
	txhash.WriteByte(byte(DATA_Transaction))
	// get transaction hash
	txHashValue := tx.Hash()
	txHashValue.Serialize(txhash)
	log.Debug(fmt.Sprintf("transaction header + hash: %x\n", txhash))

	// generate value
	w := bytes.NewBuffer(nil)
	serialization.WriteUint32(w, height)
	tx.Serialize(w)
	log.Debug(fmt.Sprintf("transaction tx data: %x\n", w))

	// put value
	err := bd.st.BatchPut(txhash.Bytes(), w.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) GetBlock(hash Uint256) (*Block, error) {
	bd.mu.RLock()
	if block, ok := bd.blockCache[hash]; ok {
		bd.mu.RUnlock()
		return block, nil
	}
	bd.mu.RUnlock()

	var b *Block = new(Block)

	b.Blockdata = new(Blockdata)
	b.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	bHash, err_get := bd.st.Get(append(prefix, hash.ToArray()...))
	if err_get != nil {
		//TODO: implement error process
		return nil, err_get
	}

	r := bytes.NewReader(bHash)

	// first 8 bytes is sys_fee
	_, err := serialization.ReadUint64(r)
	if err != nil {
		return nil, err
	}

	// Deserialize block data
	err = b.FromTrimmedData(r)
	if err != nil {
		return nil, err
	}

	// Deserialize transaction
	for i := 0; i < len(b.Transactions); i++ {
		err = bd.getTx(b.Transactions[i], b.Transactions[i].Hash())
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

func (self *ChainStore) GetBookKeeperList() ([]*crypto.PubKey, []*crypto.PubKey, error) {
	prefix := []byte{byte(SYS_CurrentBookKeeper)}
	bkListValue, err_get := self.st.Get(prefix)
	if err_get != nil {
		return nil, nil, err_get
	}

	r := bytes.NewReader(bkListValue)

	// first 1 bytes is length of list
	currCount, err := serialization.ReadUint8(r)
	if err != nil {
		return nil, nil, err
	}

	var currBookKeeper = make([]*crypto.PubKey, currCount)
	for i := uint8(0); i < currCount; i++ {
		bk := new(crypto.PubKey)
		err := bk.DeSerialize(r)
		if err != nil {
			return nil, nil, err
		}

		currBookKeeper[i] = bk
	}

	nextCount, err := serialization.ReadUint8(r)
	if err != nil {
		return nil, nil, err
	}

	var nextBookKeeper = make([]*crypto.PubKey, nextCount)
	for i := uint8(0); i < nextCount; i++ {
		bk := new(crypto.PubKey)
		err := bk.DeSerialize(r)
		if err != nil {
			return nil, nil, err
		}

		nextBookKeeper[i] = bk
	}

	return currBookKeeper, nextBookKeeper, nil
}

func (bd *ChainStore) updateAccountState(b *Block) error {
	accounts := make(map[Uint160]*account.AccountState, 0)
	nLen := len(b.Transactions)
	for i := 0; i < nLen; i++ {
		for index := 0; index < len(b.Transactions[i].Outputs); index++ {
			output := b.Transactions[i].Outputs[index]
			programHash := output.ProgramHash
			assetId := output.AssetID
			if value, ok := accounts[programHash]; ok {
				value.Balances[assetId] += output.Value
			} else {
				accountState, err := bd.GetAccount(programHash)
				if err != nil && err.Error() != ErrDBNotFound.Error() {
					return err
				}
				if accountState != nil {
					accountState.Balances[assetId] += output.Value
				} else {
					balances := make(map[Uint256]Fixed64, 0)
					balances[assetId] = output.Value
					accountState = account.NewAccountState(programHash, balances)
				}
				accounts[programHash] = accountState
			}
		}

		for index := 0; index < len(b.Transactions[i].UTXOInputs); index++ {
			input := b.Transactions[i].UTXOInputs[index]
			transaction, err := bd.GetTransaction(input.ReferTxID)
			if err != nil {
				return err
			}
			index := input.ReferTxOutputIndex
			output := transaction.Outputs[index]
			programHash := output.ProgramHash
			assetId := output.AssetID
			if value, ok := accounts[programHash]; ok {
				value.Balances[assetId] -= output.Value
			} else {
				accountState, err := bd.GetAccount(programHash)
				if err != nil {
					return err
				}
				accountState.Balances[assetId] -= output.Value
				accounts[programHash] = accountState
			}
			if accounts[programHash].Balances[assetId] < 0 {
				return errors.NewErr(fmt.Sprintf("account programHash:%v, assetId:%v insufficient of balance", programHash, assetId))
			}
		}
	}
	for programHash, value := range accounts {
		accountKey := new(bytes.Buffer)
		accountKey.WriteByte(byte(ST_ACCOUNT))
		programHash.Serialize(accountKey)

		accountValue := new(bytes.Buffer)
		value.Serialize(accountValue)
		bd.st.BatchPut(accountKey.Bytes(), accountValue.Bytes())
	}
	return nil
}

func (bd *ChainStore) updateutxoUnspents(b *Block) error {
	utxoUnspents := make(map[Uint160]map[Uint256][]*tx.UTXOUnspent)
    var err error
	nLen := len(b.Transactions)
	for i := 0; i < nLen; i++ {
		for index := 0; index < len(b.Transactions[i].Outputs); index++ {
			output := b.Transactions[i].Outputs[index]
			programHash := output.ProgramHash
			assetId := output.AssetID

			// add utxoUnspent
			if _, ok := utxoUnspents[programHash]; !ok {
				utxoUnspents[programHash] = make(map[Uint256][]*tx.UTXOUnspent)
			}

			if _, ok := utxoUnspents[programHash][assetId]; !ok {
				utxoUnspents[programHash][assetId], err = bd.GetUnspentFromProgramHash(programHash, assetId)
				if err != nil {
					utxoUnspents[programHash][assetId] = make([]*tx.UTXOUnspent, 0)
				}
			}

			unspent := new(tx.UTXOUnspent)
			unspent.Txid = b.Transactions[i].Hash()
			unspent.Index = uint32(index)
			unspent.Value = output.Value

			utxoUnspents[programHash][assetId] = append(utxoUnspents[programHash][assetId], unspent)
		}

		for index := 0; index < len(b.Transactions[i].UTXOInputs); index++ {
			input := b.Transactions[i].UTXOInputs[index]
			transaction, err := bd.GetTransaction(input.ReferTxID)
			if err != nil {
				return err
			}
			index := input.ReferTxOutputIndex
			output := transaction.Outputs[index]
			programHash := output.ProgramHash
			assetId := output.AssetID

			// delete utxoUnspent
			if _, ok := utxoUnspents[programHash]; !ok {
				utxoUnspents[programHash] = make(map[Uint256][]*tx.UTXOUnspent)
			}

			if _, ok := utxoUnspents[programHash][assetId]; !ok {
				utxoUnspents[programHash][assetId], err = bd.GetUnspentFromProgramHash(programHash, assetId)
				if err != nil {
					return errors.NewErr(fmt.Sprintf("[persist] utxoUnspents programHash:%v, assetId:%v has no unspent UTXO.", programHash, assetId))
				}
			}

			flag := false
			listnum := len(utxoUnspents[programHash][assetId])
			for i := 0; i < listnum; i++ {
				if utxoUnspents[programHash][assetId][i].Txid.CompareTo(transaction.Hash()) == 0 && utxoUnspents[programHash][assetId][i].Index == uint32(index) {
					utxoUnspents[programHash][assetId][i] = utxoUnspents[programHash][assetId][listnum-1]
					utxoUnspents[programHash][assetId] = utxoUnspents[programHash][assetId][:listnum-1]

					flag = true
					break
				}
			}

			if !flag {
				return errors.NewErr(fmt.Sprintf("[persist] utxoUnspents NOT find UTXO by txid: %x, index: %d.", transaction.Hash(), index))
			}

		}
	}
	// batch put the utxoUnspents
	for programHash, programHash_value := range utxoUnspents {
		for assetId, unspents := range programHash_value {
			err := bd.saveUnspentWithProgramHash(programHash, assetId, unspents)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
func (bd *ChainStore) persist(b *Block) error {
	unspents := make(map[Uint256][]uint16)
	quantities := make(map[Uint256]Fixed64)
	dbCache := NewDBCache(bd)
	lockedAssets := make(map[Uint160]map[Uint256][]*LockAsset)
	articleInfo := make(map[string][]*payload.ArticleInfo)
	likeInfo := make(map[Uint256][]*payload.LikeArticle)

	///////////////////////////////////////////////////////////////
	// Get Unspents for every tx
	unspentPrefix := []byte{byte(IX_Unspent)}

	///////////////////////////////////////////////////////////////
	// batch write begin
	bd.st.NewBatch()

	//////////////////////////////////////////////////////////////
	// generate key with DATA_Header prefix
	bhhash := bytes.NewBuffer(nil)
	// add block header prefix.
	bhhash.WriteByte(byte(DATA_Header))
	// calc block hash
	blockHash := b.Hash()
	blockHash.Serialize(bhhash)
	log.Debugf("block header + hash: %x\n", bhhash)

	// generate value
	w := bytes.NewBuffer(nil)
	var sysfee uint64
	for _, v := range b.Transactions[0].Outputs {
		sysfee += uint64(v.Value)
	}
	serialization.WriteUint64(w, sysfee)
	b.Trim(w)

	// BATCH PUT VALUE
	bd.st.BatchPut(bhhash.Bytes(), w.Bytes())

	//////////////////////////////////////////////////////////////
	// generate key with DATA_BlockHash prefix
	bhash := bytes.NewBuffer(nil)
	bhash.WriteByte(byte(DATA_BlockHash))
	err := serialization.WriteUint32(bhash, b.Blockdata.Height)
	if err != nil {
		return err
	}
	log.Debugf("DATA_BlockHash table key: %x\n", bhash)

	// generate value
	hashWriter := bytes.NewBuffer(nil)
	hashValue := b.Blockdata.Hash()
	hashValue.Serialize(hashWriter)
	log.Debugf("DATA_BlockHash table value: %x\n", hashValue)

	needUpdateBookKeeper := false
	currBookKeeper, nextBookKeeper, err := bd.GetBookKeeperList()
	// update current BookKeeperList
	if len(currBookKeeper) != len(nextBookKeeper) {
		needUpdateBookKeeper = true
	} else {
		for i := range currBookKeeper {
			if currBookKeeper[i].X.Cmp(nextBookKeeper[i].X) != 0 ||
				currBookKeeper[i].Y.Cmp(nextBookKeeper[i].Y) != 0 {
				needUpdateBookKeeper = true
				break
			}
		}
	}
	if needUpdateBookKeeper {
		currBookKeeper = make([]*crypto.PubKey, len(nextBookKeeper))
		for i := 0; i < len(nextBookKeeper); i++ {
			currBookKeeper[i] = new(crypto.PubKey)
			currBookKeeper[i].X = new(big.Int).Set(nextBookKeeper[i].X)
			currBookKeeper[i].Y = new(big.Int).Set(nextBookKeeper[i].Y)
		}
	}

	// BATCH PUT VALUE
	bd.st.BatchPut(bhash.Bytes(), hashWriter.Bytes())

	//////////////////////////////////////////////////////////////
	// save transactions to leveldb
	nLen := len(b.Transactions)

	for i := 0; i < nLen; i++ {
		err = bd.SaveTransaction(b.Transactions[i], b.Blockdata.Height)
		if err != nil {
			return err
		}
		txHash := b.Transactions[i].Hash()
		switch b.Transactions[i].TxType {
		case tx.RegisterAsset:
			ar := b.Transactions[i].Payload.(*payload.RegisterAsset)
			err = bd.SaveAsset(b.Transactions[i].Hash(), ar.Asset)
			if err != nil {
				return err
			}
		case tx.RegisterUser:
			userInfo := b.Transactions[i].Payload.(*payload.RegisterUser)

			err = bd.SaveUserInfo(userInfo)
			if err != nil {
				return err
			}
		case tx.PostArticle:
			author := b.Transactions[i].Payload.(*payload.ArticleInfo).Author
			tmp := b.Transactions[i].Payload.(*payload.ArticleInfo)
			articleInfo[author] = append(articleInfo[author], tmp)
		case tx.LikeArticle:
			hash := b.Transactions[i].Payload.(*payload.LikeArticle).Articlehash
			tmp := b.Transactions[i].Payload.(*payload.LikeArticle)
			likeInfo[hash] = append(likeInfo[hash], tmp)
		/*case tx.ReplyArticle:
			replier := b.Transactions[i].Payload.(*payload.ReplyArticle).Replier
			tmp := &forum.ArticleInfo{
				ParentTxnHash: b.Transactions[i].Payload.(*payload.ReplyArticle).PostHash,
				ContentHash:   b.Transactions[i].Payload.(*payload.ReplyArticle).ContentHash,
				ContentType:   forum.Reply,
			}
			articleInfo[replier] = append(articleInfo[replier], tmp)*/
		case tx.Withdrawal:
			payee := b.Transactions[i].Payload.(*payload.Withdrawal).Payee
			info := &forum.TokenInfo{
				Number: b.Transactions[i].Outputs[0].Value,
			}
			if err := bd.UpdateUserWithdrawnToken(payee, forum.WithdrawnToken, info); err != nil {
				log.Error("Failed to update withdraw info")
				continue
			}
		case tx.LockAsset:
			lp := b.Transactions[i].Payload.(*payload.LockAsset)
			if _, ok := lockedAssets[lp.ProgramHash]; !ok {
				lockedAssets[lp.ProgramHash] = make(map[Uint256][]*LockAsset)
			}
			if _, ok := lockedAssets[lp.ProgramHash][lp.AssetID]; !ok {
				lockedAssets[lp.ProgramHash][lp.AssetID], err = bd.GetLockedFromProgramHash(lp.ProgramHash, lp.AssetID)
				if err != nil {
					lockedAssets[lp.ProgramHash][lp.AssetID] = make([]*LockAsset, 0)
				}
			}
			newAsset := &LockAsset{
				Lock:   b.Blockdata.Height,
				Unlock: lp.UnlockHeight,
				Amount: lp.Amount,
			}
			lockedAssets[lp.ProgramHash][lp.AssetID] = append(lockedAssets[lp.ProgramHash][lp.AssetID], newAsset)

		case tx.IssueAsset:
			results := b.Transactions[i].GetMergedAssetIDValueFromOutputs()
			for assetId, value := range results {
				if _, ok := quantities[assetId]; !ok {
					quantities[assetId] += value
				} else {
					quantities[assetId] = value
				}
			}
		case tx.DeployCode:
			deployCode := b.Transactions[i].Payload.(*payload.DeployCode)
			codeHash := deployCode.Code.CodeHash()
			dbCache.GetOrAdd(ST_Contract, string(codeHash.ToArray()), &states.ContractState{
				Code:        deployCode.Code,
				Name:        deployCode.Name,
				Version:     deployCode.CodeVersion,
				Author:      deployCode.Author,
				Email:       deployCode.Email,
				Description: deployCode.Description,
				Language:    deployCode.Language,
				ProgramHash: deployCode.ProgramHash,
			})

			smartContract, err := smartcontract.NewSmartContract(&smartcontract.Context{
				Language:     deployCode.Language,
				Caller:       deployCode.ProgramHash,
				StateMachine: service.NewStateMachine(dbCache, NewDBCache(bd)),
				DBCache:      dbCache,
				Code:         deployCode.Code.Code,
				Time:         big.NewInt(int64(b.Blockdata.Timestamp)),
				BlockNumber:  big.NewInt(int64(b.Blockdata.Height)),
				Gas:          Fixed64(0),
			})

			if err != nil {
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, DEPLOY_TRANSACTION, err)
				return err
			}

			ret, err := smartContract.DeployContract()
			if err != nil {
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, DEPLOY_TRANSACTION, err)
				continue
			}

			hash, err := ToCodeHash(ret)
			if err != nil {
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, DEPLOY_TRANSACTION, err)
				return err
			}

			httpwebsocket.PushResult(txHash, 0, DEPLOY_TRANSACTION, BytesToHexString(hash.ToArrayReverse()))
			err = dbCache.Commit()
			if err != nil {
				return err
			}
		case tx.InvokeCode:
			invokeCode := b.Transactions[i].Payload.(*payload.InvokeCode)
			contract, err := bd.GetContract(invokeCode.CodeHash)
			if err != nil {
				log.Error("db getcontract err:", err)
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, INVOKE_TRANSACTION, err)
				continue
			}
			state, err := states.GetStateValue(ST_Contract, contract)
			if err != nil {
				log.Error("states GetStateValue err:", err)
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, INVOKE_TRANSACTION, err)
				return err
			}
			contractState := state.(*states.ContractState)
			stateMachine := service.NewStateMachine(dbCache, NewDBCache(bd))
			smartContract, err := smartcontract.NewSmartContract(&smartcontract.Context{
				Language:       contractState.Language,
				Caller:         invokeCode.ProgramHash,
				StateMachine:   stateMachine,
				DBCache:        dbCache,
				CodeHash:       invokeCode.CodeHash,
				Input:          invokeCode.Code,
				SignableData:   b.Transactions[i],
				CacheCodeTable: NewCacheCodeTable(dbCache),
				Time:           big.NewInt(int64(b.Blockdata.Timestamp)),
				BlockNumber:    big.NewInt(int64(b.Blockdata.Height)),
				Gas:            Fixed64(0),
				ReturnType:     contractState.Code.ReturnType,
				ParameterTypes: contractState.Code.ParameterTypes,
			})
			if err != nil {
				log.Error("smartcontract NewSmartContract err:", err)
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, INVOKE_TRANSACTION, err)
				continue
			}
			ret, err := smartContract.InvokeContract()
			if err != nil {
				log.Error("smartContract InvokeContract err:", err)
				httpwebsocket.PushResult(txHash, errors.SMARTCODE_ERROR, INVOKE_TRANSACTION, err)
				continue
			}
			stateMachine.CloneCache.Commit()
			httpwebsocket.PushResult(txHash, 0, INVOKE_TRANSACTION, ret)
		}

		// init unspent in tx
		txhash := b.Transactions[i].Hash()
		for index := 0; index < len(b.Transactions[i].Outputs); index++ {
			unspents[txhash] = append(unspents[txhash], uint16(index))
		}

		// delete unspent when spent in input
		for index := 0; index < len(b.Transactions[i].UTXOInputs); index++ {
			txhash := b.Transactions[i].UTXOInputs[index].ReferTxID

			// if get unspent by utxo
			if _, ok := unspents[txhash]; !ok {
				unspentValue, err_get := bd.st.Get(append(unspentPrefix, txhash.ToArray()...))

				if err_get != nil {
					return err_get
				}

				unspents[txhash], err_get = GetUint16Array(unspentValue)
				if err_get != nil {
					return err_get
				}
			}

			// find Transactions[i].UTXOInputs[index].ReferTxOutputIndex and delete it
			unspentLen := len(unspents[txhash])
			for k, outputIndex := range unspents[txhash] {
				if outputIndex == uint16(b.Transactions[i].UTXOInputs[index].ReferTxOutputIndex) {
					unspents[txhash][k] = unspents[txhash][unspentLen-1]
					unspents[txhash] = unspents[txhash][:unspentLen-1]
					break
				}
			}
		}

		// bookkeeper
		if b.Transactions[i].TxType == tx.BookKeeper {
			bk := b.Transactions[i].Payload.(*payload.BookKeeper)

			switch bk.Action {
			case payload.BookKeeperAction_ADD:
				findflag := false
				for k := 0; k < len(nextBookKeeper); k++ {
					if bk.PubKey.X.Cmp(nextBookKeeper[k].X) == 0 && bk.PubKey.Y.Cmp(nextBookKeeper[k].Y) == 0 {
						findflag = true
						break
					}
				}

				if !findflag {
					needUpdateBookKeeper = true
					nextBookKeeper = append(nextBookKeeper, bk.PubKey)
					sort.Sort(crypto.PubKeySlice(nextBookKeeper))
				}
			case payload.BookKeeperAction_SUB:
				ind := -1
				for k := 0; k < len(nextBookKeeper); k++ {
					if bk.PubKey.X.Cmp(nextBookKeeper[k].X) == 0 && bk.PubKey.Y.Cmp(nextBookKeeper[k].Y) == 0 {
						ind = k
						break
					}
				}

				if ind != -1 {
					needUpdateBookKeeper = true
					// already sorted
					nextBookKeeper = append(nextBookKeeper[:ind], nextBookKeeper[ind+1:]...)
				}
			}

		}

	}
	if err = bd.updateutxoUnspents(b); err != nil {
		return err
	}
	if err = bd.updateAccountState(b); err != nil {
		return err
	}
	if needUpdateBookKeeper {
		//bookKeeper key
		bkListKey := bytes.NewBuffer(nil)
		bkListKey.WriteByte(byte(SYS_CurrentBookKeeper))

		//bookKeeper value
		bkListValue := bytes.NewBuffer(nil)

		serialization.WriteUint8(bkListValue, uint8(len(currBookKeeper)))
		for k := 0; k < len(currBookKeeper); k++ {
			currBookKeeper[k].Serialize(bkListValue)
		}

		serialization.WriteUint8(bkListValue, uint8(len(nextBookKeeper)))
		for k := 0; k < len(nextBookKeeper); k++ {
			nextBookKeeper[k].Serialize(bkListValue)
		}

		// BookKeeper put value
		bd.st.BatchPut(bkListKey.Bytes(), bkListValue.Bytes())

		///////////////////////////////////////////////////////
	}
	///////////////////////////////////////////////////////
	//*/


	// batch put the unspents
	for txhash, value := range unspents {
		unspentKey := bytes.NewBuffer(nil)
		unspentKey.WriteByte(byte(IX_Unspent))
		txhash.Serialize(unspentKey)

		if len(value) == 0 {
			bd.st.BatchDelete(unspentKey.Bytes())
		} else {
			unspentArray := ToByteArray(value)
			bd.st.BatchPut(unspentKey.Bytes(), unspentArray)
		}
	}

	// batch put quantities
	for assetId, value := range quantities {
		quantityKey := bytes.NewBuffer(nil)
		quantityKey.WriteByte(byte(ST_QuantityIssued))
		assetId.Serialize(quantityKey)

		qt, err := bd.GetQuantityIssued(assetId)
		if err != nil {
			return err
		}

		qt = qt + value

		quantityArray := bytes.NewBuffer(nil)
		qt.Serialize(quantityArray)

		bd.st.BatchPut(quantityKey.Bytes(), quantityArray.Bytes())
		log.Debug(fmt.Sprintf("quantityKey: %x\n", quantityKey.Bytes()))
		log.Debug(fmt.Sprintf("quantityArray: %x\n", quantityArray.Bytes()))
	}


	for programHash, assets := range lockedAssets {
		for assetID, locked := range assets {
			if err := bd.SaveLockedAsset(programHash, assetID, locked); err != nil {
				return err
			}
		}
	}

	for user, info := range articleInfo {
		if err := bd.UpdateUserArticleInfo(user, info); err != nil {
			return err
		}
	}

	userTokenInfo := make(map[string]*forum.TokenInfo)
	userReputationInfo := make(map[string]*payload.RegisterUser)
	for postTxnHash, liker := range likeInfo {
		// update like info for each post/reply transaction
		if err := bd.UpdateLikeInfo(postTxnHash, liker); err != nil {
			return err
		}

		// get author of each post/reply transaction
		artinfo,err := bd.GetArticleInfo(postTxnHash)
		if err != nil {
			return err
		}
		author := artinfo.Author

		if _, ok := userTokenInfo[author]; !ok {
			existedTokenInfo, err := bd.GetTokenInfo(author, forum.TotalToken)
			if err != nil {
				return err
			}
			userTokenInfo[author] = existedTokenInfo
		}
		if _, ok := userReputationInfo[author]; !ok {
			existedReputationInfo, err := bd.GetUserInfo(author)
			if err != nil {
				return err
			}
			userReputationInfo[author] = existedReputationInfo
		}

		// calculate total token and reputation info for each author
		for _, l := range liker {
			userInfo, err := bd.GetUserInfo(l.Liker)
			if err != nil {
				return err
			}
			switch l.Liketype() {
			case payload.LikePost:
				userTokenInfo[author].Number += userInfo.Reputation / 1000
				userReputationInfo[author].Reputation += userInfo.Reputation / 1000
			case payload.DislikePost:
				userReputationInfo[author].Reputation -= userInfo.Reputation / 1000
			}
		}
	}
	for user, tokenInfo := range userTokenInfo {
		if err := bd.SaveTokenInfo(user, forum.TotalToken, tokenInfo); err != nil {
			return err
		}
	}
	for _, reputationInfo := range userReputationInfo {
		if reputationInfo.Reputation <= Fixed64(100000000) {
			reputationInfo.Reputation = 100000000
		}
		if err := bd.SaveUserInfo(reputationInfo); err != nil {
			return err
		}
	}

	currentBlockKey := bytes.NewBuffer(nil)
	currentBlockKey.WriteByte(byte(SYS_CurrentBlock))

	currentBlock := bytes.NewBuffer(nil)
	blockHash.Serialize(currentBlock)
	serialization.WriteUint32(currentBlock, b.Blockdata.Height)

	// BATCH PUT VALUE
	bd.st.BatchPut(currentBlockKey.Bytes(), currentBlock.Bytes())

	err = dbCache.Commit()
	if err != nil {
		return err
	}

	err = bd.st.BatchCommit()

	if err != nil {
		return err
	}

	return nil
}

// can only be invoked by backend write goroutine
func (bd *ChainStore) addHeader(header *Header) {

	log.Debugf("addHeader(), Height=%d\n", header.Blockdata.Height)

	hash := header.Blockdata.Hash()

	bd.mu.Lock()
	bd.headerCache[header.Blockdata.Hash()] = header
	bd.headerIndex[header.Blockdata.Height] = hash
	bd.mu.Unlock()

	log.Debug("[addHeader]: finish, header height:", header.Blockdata.Height)
}

func (self *ChainStore) handlePersistHeaderTask(header *Header) {

	if header.Blockdata.Height != uint32(len(self.headerIndex)) {
		return
	}

	if !self.verifyHeader(header) {
		return
	}

	self.addHeader(header)
}

func (self *ChainStore) SaveBlock(b *Block, ledger *Ledger) error {
	log.Debug("SaveBlock()")

	self.mu.RLock()
	headerHeight := uint32(len(self.headerIndex))
	currBlockHeight := self.currentBlockHeight
	self.mu.RUnlock()

	if b.Blockdata.Height <= currBlockHeight {
		return nil
	}

	if b.Blockdata.Height > headerHeight {
		log.Infof("Info: [SaveBlock] block height - headerIndex.count >= 1, block height:%d, headerIndex.count:%d",
			b.Blockdata.Height, headerHeight)
		return nil
	}

	if b.Blockdata.Height == headerHeight {
		err := validation.VerifyBlock(b, ledger, false)
		if err != nil {
			log.Error("VerifyBlock error!")
			return err
		}

		self.taskCh <- &persistHeaderTask{header: &Header{Blockdata: b.Blockdata}}
	} else {
		flag, err := validation.VerifySignableData(b)
		if flag == false || err != nil {
			log.Error("VerifyBlock error!")
			return err
		}
	}

	self.taskCh <- &persistBlockTask{block: b, ledger: ledger}
	return nil
}

func (self *ChainStore) handlePersistBlockTask(b *Block, ledger *Ledger) {
	if b.Blockdata.Height <= self.currentBlockHeight {
		return
	}

	self.mu.Lock()
	self.blockCache[b.Hash()] = b
	self.mu.Unlock()

	if b.Blockdata.Height < uint32(len(self.headerIndex)) {
		self.persistBlocks(ledger)

		self.st.NewBatch()
		storedHeaderCount := self.storedHeaderCount
		for self.currentBlockHeight-storedHeaderCount >= HeaderHashListCount {
			hashBuffer := new(bytes.Buffer)
			serialization.WriteVarUint(hashBuffer, uint64(HeaderHashListCount))
			var hashArray []byte
			for i := 0; i < HeaderHashListCount; i++ {
				index := storedHeaderCount + uint32(i)
				thash := self.headerIndex[index]
				thehash := thash.ToArray()
				hashArray = append(hashArray, thehash...)
			}
			hashBuffer.Write(hashArray)

			hhlPrefix := bytes.NewBuffer(nil)
			hhlPrefix.WriteByte(byte(IX_HeaderHashList))
			serialization.WriteUint32(hhlPrefix, storedHeaderCount)

			self.st.BatchPut(hhlPrefix.Bytes(), hashBuffer.Bytes())
			storedHeaderCount += HeaderHashListCount
		}

		err := self.st.BatchCommit()
		if err != nil {
			log.Error("failed to persist header hash list:", err)
			return
		}
		self.mu.Lock()
		self.storedHeaderCount = storedHeaderCount
		self.mu.Unlock()

		self.clearCache()
	}
}

func (bd *ChainStore) persistBlocks(ledger *Ledger) {
	stopHeight := uint32(len(bd.headerIndex))
	for h := bd.currentBlockHeight + 1; h <= stopHeight; h++ {
		hash := bd.headerIndex[h]
		block, ok := bd.blockCache[hash]
		if !ok {
			break
		}
		err := bd.persist(block)
		if err != nil {
			log.Fatal("[persistBlocks]: error to persist block:", err.Error())
			return
		}

		// PersistCompleted event
		ledger.Blockchain.BlockHeight = block.Blockdata.Height
		bd.mu.Lock()
		bd.currentBlockHeight = block.Blockdata.Height
		bd.mu.Unlock()

		ledger.Blockchain.BCEvents.Notify(events.EventBlockPersistCompleted, block)
		log.Tracef("The latest block height:%d, block hash: %x", block.Blockdata.Height, hash)
	}

}

func (bd *ChainStore) BlockInCache(hash Uint256) bool {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	_, ok := bd.blockCache[hash]
	return ok
}

func (bd *ChainStore) GetQuantityIssued(assetId Uint256) (Fixed64, error) {
	log.Debug(fmt.Sprintf("GetQuantityIssued Hash: %x\n", assetId))

	prefix := []byte{byte(ST_QuantityIssued)}
	data, err_get := bd.st.Get(append(prefix, assetId.ToArray()...))
	log.Debug(fmt.Sprintf("GetQuantityIssued Data: %x\n", data))

	var quantity Fixed64
	if err_get != nil {
		quantity = Fixed64(0)
	} else {
		r := bytes.NewReader(data)
		quantity.Deserialize(r)
	}

	return quantity, nil
}

func (bd *ChainStore) GetUnspent(txid Uint256, index uint16) (*tx.TxOutput, error) {
	if ok, _ := bd.ContainsUnspent(txid, index); ok {
		Tx, err := bd.GetTransaction(txid)
		if err != nil {
			return nil, err
		}

		return Tx.Outputs[index], nil
	}

	return nil, errors.NewErr("[GetUnspent] NOT ContainsUnspent.")
}

func (bd *ChainStore) ContainsUnspent(txid Uint256, index uint16) (bool, error) {
	unspentPrefix := []byte{byte(IX_Unspent)}
	unspentValue, err_get := bd.st.Get(append(unspentPrefix, txid.ToArray()...))

	if err_get != nil {
		return false, err_get
	}

	unspentArray, err_get := GetUint16Array(unspentValue)
	if err_get != nil {
		return false, err_get
	}

	for i := 0; i < len(unspentArray); i++ {
		if unspentArray[i] == index {
			return true, nil
		}
	}

	return false, nil
}

func (bd *ChainStore) GetCurrentHeaderHash() Uint256 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	return bd.headerIndex[uint32(len(bd.headerIndex)-1)]
}

func (bd *ChainStore) GetHeaderHashByHeight(height uint32) Uint256 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.headerIndex[height]
}

func (bd *ChainStore) GetHeaderHeight() uint32 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return uint32(len(bd.headerIndex) - 1)
}

func (bd *ChainStore) GetHeight() uint32 {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	return bd.currentBlockHeight
}

func (bd *ChainStore) GetAccount(programHash Uint160) (*account.AccountState, error) {
	accountPrefix := []byte{byte(ST_ACCOUNT)}

	state, err := bd.st.Get(append(accountPrefix, programHash.ToArray()...))

	if err != nil {
		return nil, err
	}

	accountState := new(account.AccountState)
	accountState.Deserialize(bytes.NewBuffer(state))

	return accountState, nil
}

func (bd *ChainStore) IsBlockInStore(hash Uint256) bool {

	var b *Block = new(Block)

	b.Blockdata = new(Blockdata)
	b.Blockdata.Program = new(program.Program)

	prefix := []byte{byte(DATA_Header)}
	blockData, err_get := bd.st.Get(append(prefix, hash.ToArray()...))
	if err_get != nil {
		return false
	}

	r := bytes.NewReader(blockData)

	// first 8 bytes is sys_fee
	_, err := serialization.ReadUint64(r)
	if err != nil {
		return false
	}

	// Deserialize block data
	err = b.FromTrimmedData(r)
	if err != nil {
		return false
	}

	if b.Blockdata.Height > bd.currentBlockHeight {
		return false
	}

	return true
}

func (bd *ChainStore) GetAvailableAsset(programHash Uint160, assetID Uint256) (Fixed64, Fixed64, error) {
	// get total asset
	var total Fixed64
	utxos, err := bd.GetUnspentFromProgramHash(programHash, assetID)
	if err != nil {
		return Fixed64(-1), Fixed64(-1), err
	}
	for _, v := range utxos {
		total += v.Value
	}

	// get locked asset
	var locked Fixed64
	l, _ := bd.GetLockedFromProgramHash(programHash, assetID)
	for _, v := range l {
		if v.Unlock > bd.currentBlockHeight {
			locked += v.Amount
		}
	}

	return total, locked, nil
}

func (bd *ChainStore) GetLockedFromProgramHash(programHash Uint160, assetID Uint256) ([]*LockAsset, error) {
	prefix := []byte{byte(ST_Locked)}
	key := append(prefix, programHash.ToArray()...)
	key = append(key, assetID.ToArray()...)

	value, err := bd.st.Get(key)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(value)
	num, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return nil, err
	}

	lockedAssets := make([]*LockAsset, num)
	for i := 0; i < int(num); i++ {
		var locked LockAsset
		if err := locked.Deserialize(r); err != nil {
			return nil, err
		}
		lockedAssets[i] = &locked
	}

	return lockedAssets, nil
}

func (bd *ChainStore) GetUnspentFromProgramHash(programHash Uint160, assetid Uint256) ([]*tx.UTXOUnspent, error) {

	prefix := []byte{byte(IX_Unspent_UTXO)}

	key := append(prefix, programHash.ToArray()...)
	key = append(key, assetid.ToArray()...)
	unspentsData, err := bd.st.Get(key)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(unspentsData)
	listNum, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return nil, err
	}

	//log.Trace(fmt.Printf("[getUnspentFromProgramHash] listNum: %d, unspentsData: %x\n", listNum, unspentsData ))

	// read unspent list in store
	unspents := make([]*tx.UTXOUnspent, listNum)
	for i := 0; i < int(listNum); i++ {
		uu := new(tx.UTXOUnspent)
		err := uu.Deserialize(r)
		if err != nil {
			return nil, err
		}

		unspents[i] = uu
	}

	return unspents, nil
}

func (bd *ChainStore) saveUnspentWithProgramHash(programHash Uint160, assetid Uint256, unspents []*tx.UTXOUnspent) error {
	prefix := []byte{byte(IX_Unspent_UTXO)}

	key := append(prefix, programHash.ToArray()...)
	key = append(key, assetid.ToArray()...)

	listnum := len(unspents)
	w := bytes.NewBuffer(nil)
	serialization.WriteVarUint(w, uint64(listnum))
	for i := 0; i < listnum; i++ {
		unspents[i].Serialize(w)
	}

	// BATCH PUT VALUE
	err := bd.st.BatchPut(key, w.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (bd *ChainStore) GetUnspentsFromProgramHash(programHash Uint160) (map[Uint256][]*tx.UTXOUnspent, error) {
	uxtoUnspents := make(map[Uint256][]*tx.UTXOUnspent)

	prefix := []byte{byte(IX_Unspent_UTXO)}
	key := append(prefix, programHash.ToArray()...)
	iter := bd.st.NewIterator(key)
	for iter.Next() {
		rk := bytes.NewReader(iter.Key())

		// read prefix
		_, _ = serialization.ReadBytes(rk, 1)
		var ph Uint160
		ph.Deserialize(rk)
		var assetid Uint256
		assetid.Deserialize(rk)

		r := bytes.NewReader(iter.Value())
		listNum, err := serialization.ReadVarUint(r, 0)
		if err != nil {
			return nil, err
		}

		// read unspent list in store
		unspents := make([]*tx.UTXOUnspent, listNum)
		for i := 0; i < int(listNum); i++ {
			uu := new(tx.UTXOUnspent)
			err := uu.Deserialize(r)
			if err != nil {
				return nil, err
			}

			unspents[i] = uu
		}
		uxtoUnspents[assetid] = unspents
	}

	return uxtoUnspents, nil
}

func (bd *ChainStore) GetUnspentOutputFromProgramHash(programHash Uint160) (map[*tx.UTXOTxInput]*tx.TxOutput, error) {
	unspends, err := bd.GetUnspentsFromProgramHash(programHash)
	if err != nil {
		return nil,  err
	}
	results := make(map[*tx.UTXOTxInput]*tx.TxOutput)
	for _, u := range unspends {
		for _, v := range u {
			input := new(tx.UTXOTxInput)
			input.ReferTxID = v.Txid
			input.ReferTxOutputIndex = uint16(v.Index)

			txn, err := bd.GetTransaction(v.Txid)
			if err != nil {
				return nil, err
			}
			output := new(tx.TxOutput)
			output.AssetID = txn.Outputs[v.Index].AssetID
			output.ProgramHash = txn.Outputs[v.Index].ProgramHash
			output.Value = txn.Outputs[v.Index].Value
			results[input] = output

		}
	}
	return results, nil
}

func (bd *ChainStore) GetAssets() map[Uint256]*Asset {
	assets := make(map[Uint256]*Asset)

	iter := bd.st.NewIterator([]byte{byte(ST_Info)})
	for iter.Next() {
		rk := bytes.NewReader(iter.Key())

		// read prefix
		_, _ = serialization.ReadBytes(rk, 1)
		var assetid Uint256
		assetid.Deserialize(rk)
		log.Tracef("[GetAssets] assetid: %x\n", assetid.ToArray())

		asset := new(Asset)
		r := bytes.NewReader(iter.Value())
		asset.Deserialize(r)

		assets[assetid] = asset
	}

	return assets
}

func (bd *ChainStore) GetStorage(key []byte) ([]byte, error) {
	prefix := []byte{byte(ST_Storage)}
	bData, err_get := bd.st.Get(append(prefix, key...))

	if err_get != nil {
		return nil, err_get
	}
	return bData, nil
}

func (db *ChainStore) GetUserInfo(name string) (*payload.RegisterUser, error) {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_User))
	key.WriteString(name)

	rawUserInfo, err := db.st.Get(key.Bytes())
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(rawUserInfo)
	var userInfo payload.RegisterUser
	if err := userInfo.Deserialize(r, 0); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (bd *ChainStore) SaveUserInfo(userInfo *payload.RegisterUser) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_User))
	key.WriteString(userInfo.UserName)

	value := bytes.NewBuffer(nil)
	if err := userInfo.Serialize(value, 0); err != nil {
		return err
	}

	if err := bd.st.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

//get user's article hash array
func (db *ChainStore) GetUserArticleInfo(author string) ([]Uint256, error) {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Post))
	key.WriteString(author)

	existed, _ := db.st.Get(key.Bytes())
	if len(existed) == 0 {
		return nil, nil
	}
	buf := bytes.NewBuffer(existed)
	num, err := serialization.ReadUint32(buf)
	if err != nil {
		return nil, err
	}
	result := make([]Uint256, num)
	for i := range result {
		if err := result[i].Deserialize(buf); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (db *ChainStore) GetArticleInfo(articlehash Uint256) (payload.ArticleInfo, error) {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Article))
    articlehash.Serialize(key)

	valuebuffer, _ := db.st.Get(key.Bytes())
	buf := bytes.NewBuffer(valuebuffer)
	var artinfo payload.ArticleInfo
	err := artinfo.Deserialize(buf, 0)
	return artinfo, err
}

func (db *ChainStore) SaveArticleInfo(postInfo *payload.ArticleInfo) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Article))
	postInfo.Articlehash.Serialize(key)

	value := bytes.NewBuffer(nil)
	if err := postInfo.Serialize(value, 0); err != nil {
		return err
	}
	if err := db.st.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *ChainStore) SaveUserArticleInfo(author string, postInfo []Uint256) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Post))
	key.WriteString(author)

	value := bytes.NewBuffer(nil)
	serialization.WriteUint32(value, uint32(len(postInfo)))
	for _, info := range postInfo {
		if _, err := info.Serialize(value); err != nil {
			return err
		}

	}
	if err := db.st.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}
	return nil
}

func (db *ChainStore) UpdateUserArticleInfo(author string, postInfo []*payload.ArticleInfo) error {
	existed, err := db.GetUserArticleInfo(author)
	if err != nil {
		return err
	}
	var updated []Uint256
	for _, info := range postInfo {
		db.SaveArticleInfo(info)
		updated = append(updated, info.Articlehash)
	}
	updated = append(updated, existed...)
	if err := db.SaveUserArticleInfo(author, updated); err != nil {
		return err
	}
	return nil
}

func (db *ChainStore) GetLikeInfo(articlehash Uint256) ([]*payload.LikeArticle, error) {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Like))
	articlehash.Serialize(key)

	existed, _ := db.st.Get(key.Bytes())
	if len(existed) == 0 {
		return nil, nil
	}
	buf := bytes.NewBuffer(existed)
	num, err := serialization.ReadVarUint(buf, 0)
	if err != nil {
		return nil, err
	}
	info := make([]*payload.LikeArticle, num)
	for i := range info {
		info[i] = new(payload.LikeArticle)
		if err := info[i].Deserialize(buf, 0); err != nil {
			return nil, err
		}
	}

	return info, nil
}

func (db *ChainStore) SaveLikeInfo(articlehash Uint256, likeInfo []*payload.LikeArticle) error {
	key := bytes.NewBuffer(nil)
	key.WriteByte(byte(ST_Like))
	articlehash.Serialize(key)

	value := bytes.NewBuffer(nil)
	if err := serialization.WriteVarUint(value, uint64(len(likeInfo))); err != nil {
		return err
	}
	for _, info := range likeInfo {
		if err := info.Serialize(value, 0); err != nil {
			return err
		}
	}

	if err := db.st.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) UpdateLikeInfo(articlehash Uint256, likeInfo []*payload.LikeArticle) error {
	existed, err := db.GetLikeInfo(articlehash)
	if err != nil {
		return err
	}
	var updated []*payload.LikeArticle
	updated = append(updated, existed...)
	updated = append(updated, likeInfo...)
	if err := db.SaveLikeInfo(articlehash, updated); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) GetTokenInfo(name string, tokenType forum.TokenType) (*forum.TokenInfo, error) {
	key := bytes.NewBuffer(nil)
	switch tokenType {
	case forum.TotalToken:
		key.WriteByte(byte(ST_Total))
	case forum.WithdrawnToken:
		key.WriteByte(byte(ST_Withdraw))
	}
	key.WriteString(name)

	var tokenInfo forum.TokenInfo
	rawTokenInfo, _ := db.st.Get(key.Bytes())
	if len(rawTokenInfo) == 0 {
		return &tokenInfo, nil
	}
	r := bytes.NewReader(rawTokenInfo)
	if err := tokenInfo.Deserialization(r); err != nil {
		return nil, err
	}

	return &tokenInfo, nil
}

func (db *ChainStore) GetAvailableTokenInfo(name string) (*forum.TokenInfo, error) {
	total, err := db.GetTokenInfo(name, forum.TotalToken)
	if err != nil {
		return nil, err
	}
	withdrawn, err := db.GetTokenInfo(name, forum.WithdrawnToken)
	if err != nil {
		return nil, err
	}
	available := &forum.TokenInfo{
		Number: total.Number - withdrawn.Number,
	}

	return available, nil
}

func (db *ChainStore) SaveTokenInfo(name string, tokenType forum.TokenType, tokenInfo *forum.TokenInfo) error {
	key := bytes.NewBuffer(nil)
	switch tokenType {
	case forum.TotalToken:
		key.WriteByte(byte(ST_Total))
	case forum.WithdrawnToken:
		key.WriteByte(byte(ST_Withdraw))
	}
	key.WriteString(name)

	value := bytes.NewBuffer(nil)
	if err := tokenInfo.Serialization(value); err != nil {
		return err
	}

	if err := db.st.BatchPut(key.Bytes(), value.Bytes()); err != nil {
		return err
	}

	return nil
}

func (db *ChainStore) UpdateUserWithdrawnToken(name string, tokenType forum.TokenType, withdrawInfo *forum.TokenInfo) error {
	existed, err := db.GetTokenInfo(name, tokenType)
	if err != nil {
		return err
	}
	newWithdrawInfo := &forum.TokenInfo{
		Number: existed.Number + withdrawInfo.Number,
	}
	if err := db.SaveTokenInfo(name, tokenType, newWithdrawInfo); err != nil {
		return err
	}

	return nil
}
