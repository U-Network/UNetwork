package commands

import (
	"encoding/json"
	"fmt"
	global "github.com/U-Network/UNetwork/global"
	ugeth "github.com/U-Network/UNetwork/server/ethereum"
	"github.com/ethereum/go-ethereum/core"
	"github.com/spf13/cobra"
	cfg "github.com/tendermint/tendermint/config"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
	"os"
	"path/filepath"
)

const (
	FlagChainID = "chain-id"
	FlagENV     = "env"

	defaultEnv = "private"
)

const (
	Staging      = 20
	TestNet      = 19
	MainNet      = 18
	PrivateChain = 1234
)

var InitCmd = GetInitCmd()

func GetInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize",
		RunE:  initFiles,
	}
	initCmd.Flags().String(FlagChainID, "local", "Chain ID")
	initCmd.Flags().String(FlagENV, defaultEnv, "Environment (mainnet|staging|testnet|private)")
	initCmd.Flags().String("ethparam", "", "set ethereum params.")
	return initCmd
}

func initFiles(cmd *cobra.Command, args []string) error {
	var err error
	sdir := os.ExpandEnv(filepath.Join("$HOME", ".unetwork"))
	err = InitConfigFiles(sdir)

	if err != nil {
		return err
	}
	if len(args) > 0 {
		err = initEth(args[0])
	}
	return nil
}

func InitConfigFiles(path string) error {
	config := cfg.DefaultConfig()
	config.RootDir = path
	cfg.EnsureRoot(path)

	// private validator
	privValFile := config.PrivValidatorFile()
	var pv *privval.FilePV
	if cmn.FileExists(privValFile) {
		pv = privval.LoadFilePV(privValFile)
		logger.Info("Found private validator", "path", privValFile)
	} else {
		pv = privval.GenFilePV(privValFile)
		pv.Save()
		logger.Info("Generated private validator", "path", privValFile)
	}

	nodeKeyFile := config.NodeKeyFile()
	if cmn.FileExists(nodeKeyFile) {
		logger.Info("Found node key", "path", nodeKeyFile)
	} else {
		if _, err := p2p.LoadOrGenNodeKey(nodeKeyFile); err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)
	}

	// genesis file
	genFile := config.GenesisFile()
	if cmn.FileExists(genFile) {
		logger.Info("Found genesis file", "path", genFile)
	} else {
		genDoc := types.GenesisDoc{
			ChainID:         fmt.Sprintf("test-chain-%v", cmn.RandStr(6)),
			GenesisTime:     tmtime.Now(),
			ConsensusParams: types.DefaultConsensusParams(),
		}
		genDoc.Validators = []types.GenesisValidator{{
			Address: pv.GetPubKey().Address(),
			PubKey:  pv.GetPubKey(),
			Power:   10,
		}}

		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	//ethprivatekey
	content, err := global.GenEthPrivatekey()
	if err != nil {
		return err
	}

	pathdir := filepath.Join(config.RootDir, "config")
	ethprvfilename := filepath.Join(pathdir, "eth_privatekey.json")
	global.SaveContent(content, ethprvfilename)
	return nil
}

func initEth(filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		return err
	}
	stack := ugeth.MakeFullNode(CliContext)

	for _, name := range []string{"chaindata", "lightchaindata"} {
		chaindb, err := stack.OpenDatabase(name, 0, 0)
		if err != nil {
			return err
		}

		_, hash, err := core.SetupGenesisBlock(chaindb, genesis)
		if err != nil {
			return err
		}
		logger.Info("Successfully wrote genesis state", "database", name, "hash", hash)
	}
	return nil
}
