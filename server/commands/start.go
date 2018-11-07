package commands

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	tmcmd "github.com/tendermint/tendermint/cmd/tendermint/commands"
	"github.com/tendermint/tendermint/libs/cli"
	cmn "github.com/tendermint/tendermint/libs/common"
)

// GetStartCmd - initialize a command as the start command with tick
func GetStartCmd() *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start this full node",
		RunE:  startCmd(),
	}
	tmcmd.AddNodeFlags(startCmd)
	startCmd.Flags().Int("consensus.timeout_commit", config.TMConfig.Consensus.TimeoutCommit, "Set commit time")
	startCmd.Flags().String("ethparam", "", "set ethereum params.")
	//startCmd.Flags().String("datadir", "", "set the direction.")
	return startCmd
}

// nolint TODO: move to config file
const EyesCacheSize = 10000

//returns the start command which uses the tick
func startCmd() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		rootDir := viper.GetString(cli.HomeFlag)
		return start(rootDir)
	}
}

func start(rootDir string) error {
	sys, err := StartServices(rootDir)
	if err != nil {
		return errors.Errorf("Error in start services: %v\n", err)
	}

	// time.Sleep(time.Second * 30)
	// sys.tmNode.Stop()
	// sys.ethNode.Stop()
	// wait forever
	cmn.TrapSignal(func() {
		if sys.tmNode != nil {
			sys.tmNode.Stop()
		} else {
			logger.Error("sys.tmNode is nil")
		}
		if sys.ethNode != nil {
			sys.ethNode.Stop()
		} else {
			logger.Error("sys.ethNode is nil")
		}
	})
	return nil
}
