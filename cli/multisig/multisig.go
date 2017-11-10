package multisig

import (
	"bytes"
	"fmt"
	"os"

	"UGCNetwork/account"
	. "UGCNetwork/cli/common"
	"UGCNetwork/common"
	"UGCNetwork/core/transaction"
	"UGCNetwork/net/httpjsonrpc"

	"github.com/urfave/cli"
)

func createMultisigTransaction(c *cli.Context) error {
	asset := c.String("asset")
	from := c.String("from")
	to := c.String("to")
	value := c.String("value")

	msg := ""
	switch {
	case asset == "":
		msg = "asset id is required with [--asset]"
	case from == "":
		msg = "sender address is required with [--from]"
	case to == "":
		msg = "receiver address is required with [--to]"
	case value == "":
		msg = "asset amount is required with [--value]"
	}
	if msg != "" {
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
	resp, err := httpjsonrpc.Call(Address(), "createmultisigtransaction", 0, []interface{}{asset, from, to, value})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	FormatOutput(resp)

	return nil
}

func checkMultisigTransaction(c *cli.Context) error {
	if rawtxn := c.String("rawtxn"); rawtxn != "" {
		raw, err := common.HexStringToBytes(rawtxn)
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid transaction format")
			os.Exit(1)
		}
		var txn transaction.Transaction
		err = txn.Deserialize(bytes.NewReader(raw))
		if err != nil {
			fmt.Fprintln(os.Stderr, "invalid transaction")
			os.Exit(1)
		}
		havesig, needsig, err := txn.ParseTransactionSig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "parsing transaction failed")
			os.Exit(1)
		}
		fmt.Println(fmt.Sprintf("[ %v/%v ] signature detected", havesig, havesig+needsig))
	} else {
		fmt.Fprintln(os.Stderr, "raw transaction is required with [--rawtxn]")
		os.Exit(1)
	}

	return nil
}

func signMultisigTransaction(c *cli.Context) error {
	if rawtxn := c.String("rawtxn"); rawtxn != "" {
		resp, err := httpjsonrpc.Call(Address(), "signmultisigtransaction", 0, []interface{}{rawtxn})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		FormatOutput(resp)
	} else {
		fmt.Fprintln(os.Stderr, "raw transaction is required with [--rawtxn]")
		os.Exit(1)
	}

	return nil
}

func multisigAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	var err error
	switch {
	case c.Bool("create"):
		err = createMultisigTransaction(c)
	case c.Bool("check"):
		err = checkMultisigTransaction(c)
	case c.Bool("sign"):
		err = signMultisigTransaction(c)
	default:
		cli.ShowSubcommandHelp(c)
		return nil
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "multisig",
		Usage:       "multisig transaction creation, checking and sign",
		Description: "With nodectl multisig, you use multsig transation.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "create, c",
				Usage: "create a multisig transaction",
			},
			cli.BoolFlag{
				Name:  "check",
				Usage: "check a raw multisig transaction",
			},
			cli.BoolFlag{
				Name:  "sign, s",
				Usage: "sign a multisig transaction",
			},
			cli.StringFlag{
				Name:  "rawtxn",
				Usage: "raw transaction",
			},
			cli.StringFlag{
				Name:  "wallet, w",
				Usage: "wallet name",
				Value: account.WalletFileName,
			},
			cli.StringFlag{
				Name:  "password, p",
				Usage: "wallet password",
			},
			cli.StringFlag{
				Name:  "asset, a",
				Usage: "uniq id for asset",
			},
			cli.StringFlag{
				Name:  "from, f",
				Usage: "asset from which address",
			},
			cli.StringFlag{
				Name:  "to, t",
				Usage: "asset to which address",
			},
			cli.StringFlag{
				Name:  "value, v",
				Usage: "asset amount",
				Value: "",
			},
		},
		Action: multisigAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "multisig")
			return cli.NewExitError("", 1)
		},
	}
}
