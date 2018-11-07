package commands

import (
	"flag"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/urfave/cli.v1"

	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	tmflags "github.com/tendermint/tendermint/libs/cli/flags"
	"github.com/tendermint/tendermint/libs/log"
	"strings"
)

//nolint
const (
	FlagLogLevel = "log_level"

	defaultLogLevel = "error"
)

var (
	config  = DefaultConfig()
	context *cli.Context
	logger  = log.NewTMLogger(log.NewSyncWriter(os.Stdout)).With("module", "main")

	CliContext *cli.Context
)

// preRunSetup should be set as PersistentPreRunE on the root command to
// properly handle the logging and the tracer
func preRunSetup(cmd *cobra.Command, args []string) (err error) {
	config, err = ParseConfig()
	if err != nil {
		return err
	}
	level := viper.GetString(FlagLogLevel)
	logger, err = tmflags.ParseLogLevel(level, logger, defaultLogLevel)
	if err != nil {
		return err
	}
	if viper.GetBool(tmcli.TraceFlag) {
		logger = log.NewTracingLogger(logger)
	}
	flag := cmd.Flags().Lookup("ethparam")
	if flag == nil {
		setupUUUContext("")
	} else {
		value := flag.Value
		setupUUUContext(value.String())
	}

	return nil
}

// SetUpRoot - initialize the root command
func SetUpRoot(cmd *cobra.Command) {
	cmd.PersistentPreRunE = preRunSetup
	cmd.PersistentFlags().String(FlagLogLevel, defaultLogLevel, "Log level")
}

var (
	// flags that configure the go-ethereum node

	nodeFlags = []cli.Flag{
		ethUtils.DataDirFlag,
		ethUtils.KeyStoreDirFlag,
		ethUtils.NoUSBFlag,
		// Performance tuning
		ethUtils.CacheFlag,
		ethUtils.TrieCacheGenFlag,
		// Account settings
		ethUtils.UnlockedAccountFlag,
		ethUtils.PasswordFileFlag,
		ethUtils.VMEnableDebugFlag,
		// Logging and debug settings
		ethUtils.NoCompactionFlag,
		// Gas price oracle settings
		ethUtils.GpoBlocksFlag,
		ethUtils.GpoPercentileFlag,
		TargetGasLimitFlag,
		// Gas Price
		//ethUtils.GasPriceFlag,
		// Network Id
		ethUtils.NetworkIdFlag,
	}

	rpcFlags = []cli.Flag{
		ethUtils.RPCEnabledFlag,
		ethUtils.RPCListenAddrFlag,
		ethUtils.RPCPortFlag,
		ethUtils.RPCCORSDomainFlag,
		ethUtils.RPCApiFlag,
		ethUtils.RPCVirtualHostsFlag,
		ethUtils.IPCDisabledFlag,
		ethUtils.WSEnabledFlag,
		ethUtils.WSListenAddrFlag,
		ethUtils.WSPortFlag,
		ethUtils.WSApiFlag,
		ethUtils.WSAllowedOriginsFlag,
	}

	// flags that configure the ABCI app
	tendermintFlags = []cli.Flag{
		TendermintAddrFlag,
		ABCIAddrFlag,
		ABCIProtocolFlag,
		VerbosityFlag,
		ConfigFileFlag,
		WithTendermintFlag,
		ethUtils.GCModeFlag,
	}
)

func setDefualtFlagValue(flagset *flag.FlagSet) error{
	var split []string
	split = append(split, "--rpcport")
	split = append(split, "9147")
	split = append(split, "--datadir")
	split = append(split, config.BaseConfig.RootDir)
	return flagset.Parse(split)

}

func parseethparam(flagset *flag.FlagSet, value string) error {
	split := strings.Split(value, " ")
	return flagset.Parse(split)
}

func setupUUUContext(ethparam string) error {
	// create a new context to invoke unetwork
	a := cli.NewApp()
	a.Name = "unetwork"
	a.Flags = []cli.Flag{}
	a.Flags = append(a.Flags, nodeFlags...)
	a.Flags = append(a.Flags, rpcFlags...)
	a.Flags = append(a.Flags, tendermintFlags...)

	set, err := flagSet(a.Name, a.Flags)
	if err != nil {
		return err
	}
	err = setDefualtFlagValue(set)
	if err != nil {
		return err
	}
	err = parseethparam(set, ethparam)
	if err != nil {
		return err
	}
	context = cli.NewContext(a, set, nil)
	CliContext = context

	context.GlobalSet(ethUtils.NetworkIdFlag.Name, strconv.Itoa(int(config.EMConfig.ChainId)))
	context.GlobalSet(VerbosityFlag.Name, strconv.Itoa(int(config.EMConfig.VerbosityFlag)))

	context.GlobalSet(TendermintAddrFlag.Name, config.TMConfig.RPC.ListenAddress)

	context.GlobalSet(ABCIAddrFlag.Name, config.EMConfig.ABCIAddr)
	context.GlobalSet(ABCIProtocolFlag.Name, config.EMConfig.ABCIProtocol)

	context.GlobalSet(ethUtils.RPCEnabledFlag.Name, strconv.FormatBool(config.EMConfig.RPCEnabledFlag))
	context.GlobalSet(ethUtils.RPCApiFlag.Name, config.EMConfig.RPCApiFlag)
	context.GlobalSet(ethUtils.RPCVirtualHostsFlag.Name, config.EMConfig.RPCVirtualHostsFlag)

	context.GlobalSet(ethUtils.RPCListenAddrFlag.Name, config.EMConfig.RPCListenAddrFlag)
	//context.GlobalSet(ethUtils.RPCPortFlag.Name, strconv.Itoa(int(config.EMConfig.RPCPortFlag)))
	context.GlobalSet(ethUtils.RPCCORSDomainFlag.Name, config.EMConfig.RPCCORSDomainFlag)

	context.GlobalSet(ethUtils.WSEnabledFlag.Name, strconv.FormatBool(config.EMConfig.WSEnabledFlag))
	context.GlobalSet(ethUtils.WSApiFlag.Name, config.EMConfig.WSApiFlag)

	if err := Setup(context); err != nil {
		return err
	}
	return nil
}

func flagSet(name string, flags []cli.Flag) (*flag.FlagSet, error) {
	set := flag.NewFlagSet(name, flag.ContinueOnError)

	for _, f := range flags {
		f.Apply(set)
	}
	return set, nil
}
