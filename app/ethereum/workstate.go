package ethereum

import (
	"bytes"
	"fmt"
	"github.com/U-Network/UNetwork/global"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"strconv"
	"strings"

	"math/big"
	"path/filepath"
	"sync"
)

// EthereumWorkState EthereumWorkState is used for core business logic such as trading and block packaging
type EthereumWorkState struct {
	ethereum *eth.Ethereum

	Header *ethTypes.Header
	parent *ethTypes.Block
	State  *state.StateDB

	txIndex      int
	Transactions []*ethTypes.Transaction
	Receipts     ethTypes.Receipts
	allLogs      []*ethTypes.Log

	totalUsedGas    *uint64
	totalUsedGasFee *big.Int
	gp              *core.GasPool
	Mtx             *sync.Mutex

	receiver   common.Address
	gasManager *core.FreeGasManager

	blockExtraProvision map[uint64]string
}

// NewEthereumWorkState Create and return an EthereumWorkState object pointer
func NewEthereumWorkState(ethereum *eth.Ethereum) *EthereumWorkState {

	// TODO: load eth receiver from config file
	sdir := global.Homedir()
	sdir = filepath.Join(sdir, "config")
	sdir = filepath.Join(sdir, "eth_privatekey.json")
	addr, _ := global.GetEthAddressfromfile(sdir)
	ethReceiverAddress := common.HexToAddress(addr)

	state := &EthereumWorkState{
		ethereum:   ethereum,
		Mtx:        new(sync.Mutex),
		receiver:   ethReceiverAddress,
		gasManager: ethereum.GetFreeGasManager(),
	}
	state.readBlockExtendDataFromConfigDir() // read block extra data
	err := state.reset()
	if err != nil {
		panic("NewEthereumWorkState state.reset() error: " + err.Error())
	}
	return state
}

func (es *EthereumWorkState) GetblockNumber() uint64 {
	return es.ethereum.BlockChain().CurrentBlock().NumberU64()
}

// BeginBlock starts a new Ethereum block
func (es *EthereumWorkState) BeginBlock(blockHash []byte, parentTime uint64, numTx uint64) error {

	log.Info("EthereumWorkState === === === BeginBlock: " + common.Bytes2Hex(blockHash)) // nolint: errcheck

	// update the eth header with the tendermint header
	es.updateHeaderWithTimeInfo(es.ethereum.APIBackend.ChainConfig(), parentTime, numTx)
	return nil
}

// EndBlock starts a new Ethereum block
func (es *EthereumWorkState) EndBlock(blockheight uint64) error {

	// 199 height = 15 min to update
	if blockheight % 199 == 0 {
		es.readBlockExtendDataFromConfigDir()
	}

	return nil
}

// DeliverTx is called back by the tendermint memory pool, as long as it is used to execute the transaction
func (es *EthereumWorkState) DeliverTx(txBytes []byte) error {
	es.Mtx.Lock()
	defer es.Mtx.Unlock()
	tx, err := es.DecodeTx(txBytes)
	if err != nil {
		return err
	}
	blockchain := es.ethereum.BlockChain()
	chainConfig := es.ethereum.APIBackend.ChainConfig()

	if es.gasManager == nil{
		es.gasManager = blockchain.GetFreeGasManager()
	}


	config := eth.DefaultConfig
	blockHash := common.Hash{}

	es.State.Prepare(tx.Hash(), blockHash, es.txIndex)
	receipt, usedGas, err := core.ApplyTransaction(
		chainConfig,
		blockchain,
		nil, // defaults to address of the author of the header
		es.gp,
		es.State,
		es.Header,
		tx,
		es.totalUsedGas,
		vm.Config{EnablePreimageRecording: config.EnablePreimageRecording},
	)
	if err != nil {
		fmt.Println("DeliverTx err: ", err.Error())
		return err
	}
	usedGasFee := big.NewInt(0).Mul(new(big.Int).SetUint64(usedGas), tx.GasPrice())
	es.totalUsedGasFee.Add(es.totalUsedGasFee, usedGasFee)

	// unetwork check gas
	from, err := ethTypes.Sender(es.ethereum.TxPool().GetSigner(), tx)
	if err != nil {
		return core.ErrInvalidSender
	}
	fromAccount, _ := es.gasManager.State.GetAccount(from)
	if tx.GasPrice().Cmp(big.NewInt(0)) == 0 {
		// Account contains the used gas

		//fmt.Println("account.UseAmount : ", account.UseAmount)
		fromAccount.UseAmount.Sub(fromAccount.UseAmount, new(big.Int).SetUint64(tx.Gas()))
		// Free gas calculated after deducting the current token
		freeGas, _ := es.gasManager.CalculateFreeGas(fromAccount, es.State.GetBalance(from))
		//current used freegas
		curUsedGas := new(big.Int).SetUint64(usedGas)

		//fmt.Println("curUsedGas :", curUsedGas.String())

		var freeGasDiff *big.Int
		if freeGas.Cmp(curUsedGas) < 0 {
			freeGasDiff = new(big.Int).Sub(curUsedGas, freeGas)
			if tx.To() != nil{
				toAccount, _ := es.gasManager.State.GetAccount(*tx.To())
				toAccount.UseAmount.Add(toAccount.UseAmount, freeGasDiff)
				fromAccount.UseAmount.Add(fromAccount.UseAmount, new(big.Int).Sub(curUsedGas, freeGasDiff))
				es.gasManager.State.SetAccountUsedGas(fromAccount)
				es.gasManager.State.SetAccountUsedGas(toAccount)
			}
		} else {

			fromAccount.UseAmount.Add(fromAccount.UseAmount, curUsedGas)
			//fmt.Println("fromAccount.UseAmount: ", fromAccount.UseAmount.String())
			es.gasManager.State.SetAccountUsedGas(fromAccount)
		}
	} else {
		fromAccount.UseAmount.Sub(fromAccount.UseAmount, new(big.Int).SetUint64(tx.Gas()))
	}

	logs := es.State.GetLogs(tx.Hash())
	es.txIndex++
	// The slices are allocated in updateHeaderWithTimeInfo
	es.Transactions = append(es.Transactions, tx)
	es.Receipts = append(es.Receipts, receipt)
	es.allLogs = append(es.allLogs, logs...)

	return nil
}

// Commit Callback is done after the completion of the Tendermint consensus,
// mainly used to create blocks and insert blocks into the blockchain
func (es *EthereumWorkState) Commit(blockheight uint64) (common.Hash, error) {

	//log.Info("EthereumWorkState +++ +++ +++ Commit") // nolint: errcheck

	es.Mtx.Lock()
	defer es.Mtx.Unlock()
	// Commit ethereum state and update the header.
	hashArray, err := es.State.Commit(true)
	if err != nil {
		log.Error("Error es.state.Commit() by Commit", "err", err)
		return common.Hash{}, err
	}
	es.Header.Root = hashArray
	es.setBlockExtendData(es.Header)

	for _, log := range es.allLogs {
		log.BlockHash = hashArray
	}
	// Save the block to disk.

	// Create block object and compute final commit hash (hash of the ethereum
	// block).
	block := ethTypes.NewBlock(es.Header, es.Transactions, nil, es.Receipts)
	blockHash := block.Hash()

	blockchain := es.ethereum.BlockChain()
	_, err = blockchain.InsertChain([]*ethTypes.Block{block})

	if err != nil {
		log.Error("Error inserting ethereum block in chain", "err", err)
		// reset all state
		es.reset()
		// error deal by insert empty block
		es.gasManager.State.ReSetState()
		return es.insertEmptyBlockToChain()
	}

	if !es.gasManager.State.IsRefresh() {
		es.gasManager.Save()
		es.gasManager.State.ReSetState()
	}
	//log.Info("Committing block", "stateHash", hashArray, "blockHash", blockHash)
	// reset all state
	es.reset()
	return blockHash, err

}

func (es *EthereumWorkState) insertEmptyBlockToChain() (common.Hash, error) {

	pt := es.parent.Time()
	pt = pt.Add(pt, big.NewInt(1))
	config := es.ethereum.APIBackend.ChainConfig()
	es.updateHeaderWithTimeInfo(config, pt.Uint64(), 0)

	hashArray, er := es.State.Commit(true)
	if er != nil {
		log.Error("Error es.state.Commit() by insertEmptyBlockToChain", "err", er)
		return common.Hash{}, er
	}
	es.Header.Root = hashArray
	es.setBlockExtendData(es.Header)

	for _, log := range es.allLogs {
		log.BlockHash = hashArray
	}

	blockchain := es.ethereum.BlockChain()

	// Create block object and compute final commit hash (hash of the ethereum
	// block).
	block := ethTypes.NewBlock(es.Header, es.Transactions, nil, es.Receipts)
	blockHash := block.Hash()
	_, er = blockchain.InsertChain([]*ethTypes.Block{block})
	if er != nil {
		log.Error("Error inserting ethereum empty block in chain", "err", er)
		return blockHash, er
	}
	return blockHash, nil
}

// reset all work state
func (es *EthereumWorkState) reset() error {

	blockchain := es.ethereum.BlockChain()
	currentBlock := blockchain.CurrentBlock()
	state, err := blockchain.State()
	if err != nil {
		return err
	}
	// test receiver
	ethHeader := newBlockHeader(es.receiver, currentBlock)

	// RESET VALUE
	es.Header = ethHeader
	es.parent = currentBlock
	es.State = state
	es.txIndex = 0
	es.totalUsedGas = new(uint64)
	es.totalUsedGasFee = big.NewInt(0)
	es.gp = new(core.GasPool).AddGas(ethHeader.GasLimit)
	es.Transactions = []*ethTypes.Transaction{} // clean
	es.Receipts = ethTypes.Receipts{}
	es.allLogs = []*ethTypes.Log{} // clean

	return nil

}

func (es *EthereumWorkState) updateHeaderWithTimeInfo(
	config *params.ChainConfig, parentTime uint64, numTx uint64) {

	lastBlock := es.parent
	parentHeader := &ethTypes.Header{
		Difficulty: lastBlock.Difficulty(),
		Number:     lastBlock.Number(),
		Time:       lastBlock.Time(),
	}
	es.Header.Time = new(big.Int).SetUint64(parentTime)
	es.Header.Difficulty = ethash.CalcDifficulty(config, parentTime, parentHeader)
	es.Transactions = make([]*ethTypes.Transaction, 0, numTx)
	es.Receipts = make([]*ethTypes.Receipt, 0, numTx)
	es.allLogs = make([]*ethTypes.Log, 0, numTx)
}

//////////////////////////////////////////////////////

// Create a new block header from the previous block.
func newBlockHeader(receiver common.Address, prevBlock *ethTypes.Block) *ethTypes.Header {
	return &ethTypes.Header{
		Number:     prevBlock.Number().Add(prevBlock.Number(), big.NewInt(1)),
		ParentHash: prevBlock.Hash(),
		GasLimit:   calcGasLimit(prevBlock),
		Coinbase:   receiver,
	}
}

func calcGasLimit(parent *ethTypes.Block) uint64 {
	// Ethereum average block gasLimit * 1000
	var gl uint64 = 8192000000 // 8192m
	return gl
}

/////////////////////////////////////////////////////

// rlp decode an etherum transaction
func (es *EthereumWorkState) DecodeTx(txBytes []byte) (*ethTypes.Transaction, error) {
	tx := new(ethTypes.Transaction)
	rlpStream := rlp.NewStream(bytes.NewBuffer(txBytes), 0)
	if err := tx.DecodeRLP(rlpStream); err != nil {
		return nil, err
	}
	return tx, nil
}

func (es *EthereumWorkState) GetEthBackend() *eth.Ethereum {
	return es.ethereum
}

func (es *EthereumWorkState) GetFreeGasManager() *core.FreeGasManager {
	return es.gasManager
}

func (es *EthereumWorkState) setBlockExtendData(header *ethTypes.Header) {

	h := header.Number.Uint64()
	val, ok := es.blockExtraProvision[h]
	// fmt.Printf(val)
	if ok {
		header.Extra = []byte(val)
	}

}

func (es *EthereumWorkState) readBlockExtendDataFromConfigDir()  {
	refresh := make(map[uint64]string)

	// TODO: load eth receiver from config file
	sdir := global.Homedir()
	sdir = filepath.Join(sdir, "config")
	extraFile := filepath.Join(sdir, "eth_block_extra.line")
	lineData, err := global.ReadFile(extraFile)
	if err != nil {
		return
	}
	// #234325=Yang jie:rlp decode an etherum transaction.
	number := 0
	lines := strings.Split(lineData, "\n")
	for i := 0; i<len(lines); i++ {
		one := lines[i]
		ss1 := strings.SplitAfter(one, "#")
		if len(ss1) != 2 { continue }
		ss2 := strings.Split(ss1[1], "=")
		if len(ss2) != 2 { continue }
		ss3 := strings.Split(ss2[1], ":")
		if len(ss3) != 2 { continue }
		bint, err  := strconv.ParseUint(ss2[0], 10, 64)
		if err != nil { continue }
		refresh[bint] = "["+ss3[0]+"] "+ss3[1]
		number++
	}
	if number > 0 {
		es.blockExtraProvision = refresh
	}
}

