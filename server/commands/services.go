package commands

import (
	"github.com/U-Network/UNetwork/api"
	ethbaseapp "github.com/U-Network/UNetwork/app/ethereum"
	tmbaseapp "github.com/U-Network/UNetwork/app/tendermint"
	ethServer "github.com/U-Network/UNetwork/server/ethereum"
	"github.com/U-Network/UNetwork/server/tendermint"
	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth"
	ethlog "github.com/ethereum/go-ethereum/log"
	tmtypes "github.com/tendermint/tendermint/abci/types"
	tmnode "github.com/tendermint/tendermint/node"
	tmclient "github.com/tendermint/tendermint/rpc/client"
	"os"
)

type Services struct {
	// tm
	tmNode  *tmnode.Node
	tmApp   *tmtypes.Application
	tmLocal *tmclient.Local
	// eth
	ethNode  *eth.Ethereum
	ethState *ethbaseapp.EthereumWorkState
}

func StartServices(rootDir string) (*Services, error) {
	// ethereum node
	emNode := ethServer.StartNewEthereum(CliContext)

	var backend *api.Backend
	// Get a registered backend object pointer from Ethereum
	if err := emNode.Service(&backend); err != nil {
		ethUtils.Fatalf("ethereum backend service not running: %v", err)
	}

	ethEthereum := backend.Ethereum()

	// eth state api
	ethState := ethbaseapp.NewEthereumWorkState(ethEthereum)

	// tendermint application
	tdmtApp := tmbaseapp.NewTendermintApplication()

	// set eth state
	tdmtApp.SetEthState(ethState)

	// Create & start tendermint node
	tmNode, err := tendermint.StartTendermint(rootDir, tdmtApp)
	if err != nil {
		ethlog.Warn(err.Error())
		os.Exit(1)
	}

	// abci local client
	tmLocal := tmclient.NewLocal(tmNode)

	// Create Services
	var tmApp tmtypes.Application = tdmtApp
	services := &Services{
		tmNode,
		&tmApp,
		tmLocal,
		ethEthereum,
		ethState,
	}

	// start tx add pool event listen
	txListen := ethbaseapp.NewTxListenPush(ethEthereum, services)
	txListen.Start()

	return services, nil
}

// broadcast tx by tendermint
func (s *Services) BroadcastTransaction(tx []byte) {
	s.tmLocal.BroadcastTxAsync(tx)
}
