package recover

import (
	"fmt"
	"os"
	"time"

	"UGCNetwork/account"
	. "UGCNetwork/cli/common"
	"UGCNetwork/common/password"
	"github.com/urfave/cli"
)

func recoverAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	privateKey := c.String("key")
	if privateKey == "" {
		fmt.Println("missing -k,--key option")
		os.Exit(1)
	}
	newPassword, err := password.GetConfirmedPassword()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	newWalletName := fmt.Sprintf("wallet-%s-recovered.dat", time.Now().Format("2006-01-02-15-04-05"))
	_, err = account.Recover(newWalletName, []byte(newPassword), privateKey)
	if err != nil {
		fmt.Println("failed to recover wallet from private key")
		os.Exit(1)
	}
	fmt.Println("wallet is recovered successfully")
	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "recover",
		Usage:       "recover wallet from private key",
		Description: "With nodectl recover, you could recover your asset.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "key, k",
				Usage: "private key",
			},
		},
		Action: recoverAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "recover")
			return cli.NewExitError("", 1)
		},
	}
}
