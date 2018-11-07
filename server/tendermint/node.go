package tendermint

import (
	abcitypes "github.com/tendermint/tendermint/abci/types"
	tcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/node"
	"github.com/tendermint/tendermint/p2p"
	pv "github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/proxy"
	"os"
)

var (
	logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")
)

// StartTendermint creates and starts the Tendermint node
func StartTendermint(rootDir string, basecoinApp abcitypes.Application) (*node.Node, error) {
	cfg, err := tcmd.ParseConfig()
	if err != nil {
		return nil, err
	}

	var papp proxy.ClientCreator
	if basecoinApp != nil {
		papp = proxy.NewLocalClientCreator(basecoinApp)
	} else {
		papp = proxy.DefaultClientCreator(cfg.ProxyApp, cfg.ABCI, cfg.DBDir())
	}

	// node key
	nodekey, _ := p2p.LoadNodeKey(rootDir + "/config/node_key.json")

	// Create & start tendermint node
	n, err := node.NewNode(cfg,
		pv.LoadOrGenFilePV(cfg.PrivValidatorFile()),
		nodekey,
		papp,
		node.DefaultGenesisDocProviderFunc(cfg),
		node.DefaultDBProvider,
		node.DefaultMetricsProvider(cfg.Instrumentation),
		logger.With("module", "node"))
	if err != nil {
		return nil, err
	}

	err = n.Start()
	if err != nil {
		return nil, err
	}

	return n, nil
}
