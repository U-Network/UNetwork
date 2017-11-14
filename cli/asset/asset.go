package asset

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"

	"UGCNetwork/account"
	. "UGCNetwork/cli/common"
	. "UGCNetwork/common"
	"UGCNetwork/core/transaction"
	"UGCNetwork/net/httpjsonrpc"
	"UGCNetwork/sdk"

	"github.com/urfave/cli"
)

const (
	RANDBYTELEN = 4
)

func openWallet(name string, passwd []byte) account.Client {
	if name == account.WalletFileName {
		fmt.Println("Using default wallet: ", account.WalletFileName)
	}
	wallet, err := account.Open(name, passwd)
	if err != nil {
		fmt.Println("Failed to open wallet: ", name)
		os.Exit(1)
	}
	return wallet
}
func parseAssetName(c *cli.Context) string {
	name := c.String("name")
	if name == "" {
		rbuf := make([]byte, RANDBYTELEN)
		rand.Read(rbuf)
		name = "UGCNetwork-" + BytesToHexString(rbuf)
	}

	return name
}

func parseAssetID(c *cli.Context) Uint256 {
	asset := c.String("asset")
	if asset == "" {
		fmt.Println("missing flag [--asset]")
		os.Exit(1)
	}
	var assetBytes []byte
	var assetID Uint256
	assetBytes, err := HexStringToBytesReverse(asset)
	if err != nil {
		fmt.Println("invalid asset ID")
		os.Exit(1)
	}
	if err := assetID.Deserialize(bytes.NewReader(assetBytes)); err != nil {
		fmt.Println("invalid asset hash")
		os.Exit(1)
	}

	return assetID
}

func parseAddress(c *cli.Context) string {
	if address := c.String("to"); address != "" {
		_, err := ToScriptHash(address)
		if err != nil {
			fmt.Println("invalid receiver address")
			os.Exit(1)
		}
		return address
	} else {
		fmt.Println("missing flag [--to]")
		os.Exit(1)
	}

	return ""
}

func parseHeight(c *cli.Context) int64 {
	height := c.Int64("height")
	if height != -1 {
		return height
	} else {
		fmt.Println("invalid parameter [--height]")
		os.Exit(1)
	}

	return 0
}

func assetAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}

	value := c.String("value")
	if value == "" {
		fmt.Println("asset amount is required with [--value]")
		return nil
	}

	var txn *transaction.Transaction
	var buffer bytes.Buffer
	var err error
	switch {
	case c.Bool("reg"):
		name := parseAssetName(c)
		wallet := openWallet(c.String("wallet"), WalletPassword(c.String("password")))
		txn, err = sdk.MakeRegTransaction(wallet, name, value)
	case c.Bool("issue"):
		assetID := parseAssetID(c)
		address := parseAddress(c)
		wallet := openWallet(c.String("wallet"), WalletPassword(c.String("password")))
		txn, err = sdk.MakeIssueTransaction(wallet, assetID, address, value)
	case c.Bool("transfer"):
		assetID := c.String("asset")
		address := parseAddress(c)
		resp, err := httpjsonrpc.Call(Address(), "sendtoaddress", 0, []interface{}{assetID, address, value})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		FormatOutput(resp)
		return nil
	case c.Bool("lock"):
		assetID := c.String("asset")
		height := parseHeight(c)
		resp, err := httpjsonrpc.Call(Address(), "lockasset", 0, []interface{}{assetID, value, height})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
		FormatOutput(resp)
		return nil
	default:
		cli.ShowSubcommandHelp(c)
		return nil
	}
	if err != nil {
		fmt.Println(err)
		return err
	}
	if err := txn.Serialize(&buffer); err != nil {
		fmt.Println("serialize transaction failed")
		return err
	}
	resp, err := httpjsonrpc.Call(Address(), "sendrawtransaction", 0, []interface{}{BytesToHexString(buffer.Bytes())})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	FormatOutput(resp)

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "asset",
		Usage:       "asset registration, issuance and transfer",
		Description: "With nodectl asset, you could control assert through transaction.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "reg, r",
				Usage: "regist a new kind of asset",
			},
			cli.BoolFlag{
				Name:  "issue, i",
				Usage: "issue asset that has been registered",
			},
			cli.BoolFlag{
				Name:  "transfer, t",
				Usage: "transfer asset",
			},
			cli.BoolFlag{
				Name:  "lock",
				Usage: "lock asset",
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
				Name:  "name",
				Usage: "asset name",
			},
			cli.StringFlag{
				Name:  "to",
				Usage: "asset to whom",
			},
			cli.StringFlag{
				Name:  "value, v",
				Usage: "asset amount",
				Value: "",
			},
			cli.Int64Flag{
				Name:  "height",
				Usage: "asset lock height",
				Value: -1,
			},
		},
		Action: assetAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "asset")
			return cli.NewExitError("", 1)
		},
	}
}
