package main

import (
	"os"
	"sort"

	_ "UNetwork/cli"
	"UNetwork/cli/asset"
	"UNetwork/cli/bookkeeper"
	. "UNetwork/cli/common"
	"UNetwork/cli/consensus"
	"UNetwork/cli/data"
	"UNetwork/cli/debug"
	"UNetwork/cli/info"
	"UNetwork/cli/multisig"
	"UNetwork/cli/privpayload"
	"UNetwork/cli/recover"
	"UNetwork/cli/smartcontract"
	"UNetwork/cli/test"
	"UNetwork/cli/wallet"

	"github.com/urfave/cli"
)

var Version string

func main() {
	app := cli.NewApp()
	app.Name = "nodectl"
	app.Version = Version
	app.HelpName = "nodectl"
	app.Usage = "command line tool for UNetwork blockchain"
	app.UsageText = "nodectl [global options] command [command options] [args]"
	app.HideHelp = false
	app.HideVersion = false
	//global options
	app.Flags = []cli.Flag{
		NewIpFlag(),
		NewPortFlag(),
	}
	//commands
	app.Commands = []cli.Command{
		*consensus.NewCommand(),
		*debug.NewCommand(),
		*info.NewCommand(),
		*test.NewCommand(),
		*wallet.NewCommand(),
		*asset.NewCommand(),
		*privpayload.NewCommand(),
		*data.NewCommand(),
		*bookkeeper.NewCommand(),
		*recover.NewCommand(),
		*multisig.NewCommand(),
		*smartcontract.NewCommand(),
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
