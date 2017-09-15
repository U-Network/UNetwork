package main

import (
	"os"
	"sort"

	_ "UGCNetwork/cli"
	"UGCNetwork/cli/asset"
	"UGCNetwork/cli/bookkeeper"
	. "UGCNetwork/cli/common"
	"UGCNetwork/cli/consensus"
	"UGCNetwork/cli/data"
	"UGCNetwork/cli/debug"
	"UGCNetwork/cli/info"
	"UGCNetwork/cli/privpayload"
	"UGCNetwork/cli/test"
	"UGCNetwork/cli/wallet"

	"github.com/urfave/cli"
)

var Version string

func main() {
	app := cli.NewApp()
	app.Name = "nodectl"
	app.Version = Version
	app.HelpName = "nodectl"
	app.Usage = "command line tool for UGCNetwork blockchain"
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
	}
	sort.Sort(cli.CommandsByName(app.Commands))
	sort.Sort(cli.FlagsByName(app.Flags))

	app.Run(os.Args)
}
