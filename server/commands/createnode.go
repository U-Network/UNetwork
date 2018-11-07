package commands

import (
	"fmt"
	global "github.com/U-Network/UNetwork/global"
	"github.com/spf13/cobra"
	cmn "github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/privval"
	"github.com/tendermint/tendermint/types"
	tmtime "github.com/tendermint/tendermint/types/time"
	"os"
	"path/filepath"
	"strconv"
)

var CreatenodeCmd = GetCreatenodeCmd()

func GetCreatenodeCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "createnode",
		Short: "create all node config files",
		RunE:  initAllFiles,
	}
	return initCmd
}

func initAllFiles(cmd *cobra.Command, args []string) error {
	sdir := os.ExpandEnv(filepath.Join("$HOME", global.ProjectDir))

	if len(args) >= 2 {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		return initAllPeersConfigFiles(num, args[1], sdir)
	}
	return nil
}

func initAllPeersConfigFiles(peernum int, chainid string, dir string) error {
	genDoc := types.GenesisDoc{
		ChainID:         chainid,
		GenesisTime:     tmtime.Now(),
		ConsensusParams: types.DefaultConsensusParams(),
	}
	for i := 1; i <= peernum; i++ {
		pathdir := filepath.Join(dir, fmt.Sprintf("config%d", i))
		os.Mkdir(pathdir, 0777)
		var pv *privval.FilePV
		privValFile := filepath.Join(pathdir, "priv_validator.json")
		if cmn.FileExists(privValFile) {
			os.Remove(privValFile)
		}
		pv = privval.GenFilePV(privValFile)
		pv.Save()
		logger.Info("Generated private validator", "path", privValFile)

		genDoc.Validators = append(genDoc.Validators, types.GenesisValidator{
			Address: pv.GetPubKey().Address(),
			PubKey:  pv.GetPubKey(),
			Power:   10,
		})

		nodeKeyFile := filepath.Join(pathdir, "node_key.json")
		if cmn.FileExists(nodeKeyFile) {
			os.Remove(nodeKeyFile)
		}
		nodeKey, err := p2p.LoadOrGenNodeKey(nodeKeyFile)
		if err != nil {
			return err
		}
		logger.Info("Generated node key", "path", nodeKeyFile)

		id := nodeKey.ID() //p2p.PubKeyToID(pv.GetPubKey())
		err1 := global.SaveContent("{\"ID\":\""+string(id)+"\"}"+"\n", filepath.Join(pathdir, "mainid.json"))
		if err1 != nil {
			return err1
		}
	}

	for i := 1; i <= peernum; i++ {
		pathdir := filepath.Join(dir, fmt.Sprintf("config%d", i))
		genFile := filepath.Join(pathdir, "genesis.json")
		if err := genDoc.SaveAs(genFile); err != nil {
			return err
		}
		logger.Info("Generated genesis file", "path", genFile)
	}

	//generate etheruem private key
	err := global.SaveEthPrivateKey(peernum, dir)
	return err
}
