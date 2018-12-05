package api

import (
	"github.com/U-Network/UNetwork/app/ethereum"
	"github.com/U-Network/UNetwork/app/ethereum/consensus"
	uMiner "github.com/U-Network/UNetwork/app/ethereum/miner"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/les"
	"github.com/ethereum/go-ethereum/log"
	ethMiner "github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	rpcClient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	"time"
)

//----------------------------------------------------------------------
// Backend manages the underlying ethereum state for storage and processing,
// and maintains the connection to Tendermint for forwarding txs

// Backend handles the chain database and VM
type Backend struct {
	// backing ethereum structures
	ethereum  *eth.Ethereum
	ethConfig *eth.Config

	// txBroadcastLoop subscription
	//txSub *event.TypeMuxSubscription
	txsCh  chan core.NewTxsEvent
	txsSub event.Subscription

	// Local client is mainly used for Ethereum to interact with tendermint.
	// There is no overhead in in-process use.
	localClient *rpcClient.Local

	// unetwork chain id
	chainID string

	// moved from txpool.pendingState
	managedState *state.ManagedState

	EthState *ethereum.EthereumWorkState
}

// NewBackend creates a new Backend
func NewBackend(ctx *node.ServiceContext, ethConfig *eth.Config) (*Backend, error) {
	ethBackend := &Backend{
		ethConfig: ethConfig,
	}
	ethConfig.LightServ = 50
	ethConfig.LightPeers = 20

	// eth.New takes a ServiceContext for the EventMux, the AccountManager,
	// and some basic functions around the DataDir.
	miner := ethMiner.NewMiner()
	miner.MinerExtend = uMiner.NewMinerExtend(ethBackend.EthState)
	ethereum, err := eth.UnetNewEthereum(ctx, ethConfig, consensus.CreateConsensusEngine(), miner)
	if err != nil {
		return nil, err
	}
	//uCommon.G_GasManager = uCommon.NewFreeGasManager(ethereum)
	ls, _ := les.NewLesServer(ethereum, ethConfig)
	ethereum.AddLesServer(ls)

	// send special event to go-ethereum to switch homestead=true.
	currentBlock := ethereum.BlockChain().CurrentBlock()
	ethereum.EventMux().Post(core.ChainHeadEvent{currentBlock}) // nolint: vet, errcheck

	// We don't need PoW/Uncle validation.
	ethereum.BlockChain().SetValidator(NullBlockProcessor{})
	ethBackend.ethereum = ethereum

	ethBackend.ResetState()
	return ethBackend, nil
}

func (b *Backend) ResetState() (*state.ManagedState, error) {
	currentState, err := b.Ethereum().BlockChain().State()
	if err != nil {
		return nil, err
	}
	b.managedState = state.ManageState(currentState)
	return b.managedState, nil
}

func (b *Backend) ManagedState() *state.ManagedState {
	return b.managedState
}

// Ethereum returns the underlying the ethereum object.
func (b *Backend) Ethereum() *eth.Ethereum {
	return b.ethereum
}

// Config returns the eth.Config.
func (b *Backend) Config() *eth.Config {
	return b.ethConfig
}

func (b *Backend) PeerCount() int {
	var net *ctypes.ResultNetInfo
	net, _ = b.GetLocalClient().NetInfo()
	if net != nil {
		return len(net.Peers)
	}
	return 0
}

func (b *Backend) GetLocalClient() *rpcClient.Local {
	for b.localClient == nil {
		log.Info("Waiting for local client to set up...")
		time.Sleep(time.Second * 1)
	}
	return b.localClient
}

//----------------------------------------------------------------------
// Implements: node.Service

func (b *Backend) APIs() []rpc.API {
	//retApis := []rpc.API{}
	retApis := b.ethereum.APIs()
	return retApis
}

// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (b *Backend) Start(srvr *p2p.Server) error {
	return b.ethereum.Start(srvr)
	//return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// Ethereum protocol.
func (b *Backend) Stop() error {
	b.ethereum.Stop() // nolint: errcheck
	return nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (b *Backend) Protocols() []p2p.Protocol {
	return nil
}

//----------------------------------------------------------------------
// We need a block processor that just ignores PoW and uncles and so on

// NullBlockProcessor does not validate anything
type NullBlockProcessor struct{}

// ValidateBody does not validate anything
func (NullBlockProcessor) ValidateBody(*ethTypes.Block) error { return nil }

// ValidateState does not validate anything
func (NullBlockProcessor) ValidateState(block, parent *ethTypes.Block, state *state.StateDB,
	receipts ethTypes.Receipts, usedGas uint64) error {
	return nil
}
