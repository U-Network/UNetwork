package wallet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"UNetwork/account"
	. "UNetwork/cli/common"
	. "UNetwork/common"
	"UNetwork/common/password"
	"UNetwork/crypto"
	"UNetwork/events/signalset"

	"github.com/urfave/cli"
	"UNetwork/net/httpjsonrpc"
)

const (
	MinMultiSignKey int = 3
)

func showAccountsInfo(wallet account.Client) {
	accounts := wallet.GetAccounts()
	fmt.Println(" ID   Address\t\t\t\t Public Key")
	fmt.Println("----  -------\t\t\t\t ----------")
	for i, account := range accounts {
		address, _ := account.ProgramHash.ToAddress()
		publicKey, _ := account.PublicKey.EncodePoint(true)
		fmt.Printf("%4s  %s %s\n", strconv.Itoa(i), address, BytesToHexString(publicKey))
	}
}

func showDefaultAccountInfo(wallet account.Client) {
	mainAccount, err := wallet.GetDefaultAccount()
	if nil == err {
		fmt.Println(" ID   Address\t\t\t\t Public Key")
		fmt.Println("----  -------\t\t\t\t ----------")

		address, _ := mainAccount.ProgramHash.ToAddress()
		publicKey, _ := mainAccount.PublicKey.EncodePoint(true)
		fmt.Printf("%4s  %s %s\n", strconv.Itoa(0), address, BytesToHexString(publicKey))
	} else {
		fmt.Println("GetDefaultAccount err! ", err.Error())
	}
}

func showMultisigInfo(wallet account.Client) {
	contracts := wallet.GetContracts()
	accounts := wallet.GetAccounts()
	resp, _ := httpjsonrpc.Call(Address(), "getutxocoins", 0, []interface{}{})
	coins := wallet.GetCoinsFromBytes(resp)

	multisign := []Uint160{}
	// find multisign address
	for _, contract := range contracts {
		found := false
		for _, account := range accounts {
			if contract.ProgramHash == account.ProgramHash {
				found = true
				break
			}
		}
		if !found {
			multisign = append(multisign, contract.ProgramHash)
		}
	}

	for _, programHash := range multisign {
		assets := make(map[Uint256]Fixed64)
		for _, out := range coins {
			if out.Output.ProgramHash == programHash {
				if _, ok := assets[out.Output.AssetID]; !ok {
					assets[out.Output.AssetID] = out.Output.Value
				} else {
					assets[out.Output.AssetID] += out.Output.Value
				}
			}
		}
		address, _ := programHash.ToAddress()
		fmt.Println("-----------------------------------------------------------------------------------")
		fmt.Printf("Address: %s\n", address)
		if len(assets) != 0 {
			fmt.Println(" ID   Asset ID\t\t\t\t\t\t\t\tAmount")
			fmt.Println("----  --------\t\t\t\t\t\t\t\t------")
			i := 0
			for id, value := range assets {
				fmt.Printf("%4s  %s  %v\n", strconv.Itoa(i), BytesToHexString(id.ToArrayReverse()), value)
				i++
			}
		}
		fmt.Println("-----------------------------------------------------------------------------------\n")
	}
}

func showBalancesInfo(wallet account.Client) {
	resp, _ := httpjsonrpc.Call(Address(), "getutxocoins", 0, []interface{}{})
	coins := wallet.GetCoinsFromBytes(resp)
	assets := make(map[Uint256]Fixed64)
		for _, out := range coins {
		if out.AddressType == account.SingleSign {
			if _, ok := assets[out.Output.AssetID]; !ok {
				assets[out.Output.AssetID] = out.Output.Value
			} else {
				assets[out.Output.AssetID] += out.Output.Value
			}
		}
	}
	if len(assets) == 0 {
		fmt.Println("no assets")
		return
	}
	fmt.Println(" ID   Asset ID\t\t\t\t\t\t\t\tAmount")
	fmt.Println("----  --------\t\t\t\t\t\t\t\t------")

	i := 0
	for id, amount := range assets {
		fmt.Printf("%4s  %s  %v\n", strconv.Itoa(i), BytesToHexString(id.ToArrayReverse()), amount)
		i++
	}
}

func showVerboseInfo(wallet account.Client) {
	accounts := wallet.GetAccounts()
	resp, _ := httpjsonrpc.Call(Address(), "getutxocoins", 0, []interface{}{})
	coins := wallet.GetCoinsFromBytes(resp)
	for _, account := range accounts {
		programHash := account.ProgramHash
		assets := make(map[Uint256]Fixed64)
		address, _ := programHash.ToAddress()
		for _, out := range coins {
			if out.Output.ProgramHash == programHash {
				if _, ok := assets[out.Output.AssetID]; !ok {
					assets[out.Output.AssetID] = out.Output.Value
				} else {
					assets[out.Output.AssetID] += out.Output.Value
				}
			}
		}
		fmt.Println("---------------------------------------------------------------------------------------------------")
		fmt.Printf("Address: %s  ProgramHash: %s\n", address, BytesToHexString(programHash.ToArrayReverse()))
		if len(assets) == 0 {
			continue
		}

		fmt.Println(" ID   Asset ID\t\t\t\t\t\t\t\tAmount")
		fmt.Println("----  --------\t\t\t\t\t\t\t\t------")
		i := 0
		for id, amount := range assets {
			fmt.Printf("%4s  %s  %v\n", strconv.Itoa(i), BytesToHexString(id.ToArrayReverse()), amount)
			i++
		}
	}
}

func showHeightInfo(wallet *account.ClientImpl) {
	h, _ := wallet.LoadStoredData("Height")
	var height uint32
	binary.Read(bytes.NewBuffer(h), binary.LittleEndian, &height)
	fmt.Println("Height: ", height)
}

func getPassword(passwd string) []byte {
	var tmp []byte
	var err error
	if passwd != "" {
		tmp = []byte(passwd)
	} else {
		tmp, err = password.GetPassword()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	return tmp
}

func getConfirmedPassword(passwd string) []byte {
	var tmp []byte
	var err error
	if passwd != "" {
		tmp = []byte(passwd)
	} else {
		tmp, err = password.GetConfirmedPassword()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	return tmp
}

func processSignals(wallet *account.ClientImpl) {
	sigHandler := func(signal os.Signal, v interface{}) {
		switch signal {
		case syscall.SIGINT:
			fmt.Println("Caught SIGINT signal, existing...")
		case syscall.SIGTERM:
			fmt.Println("Caught SIGTERM signal, existing...")
		}
		// hold the mutex lock to prevent any wallet db changes
		wallet.FileStore.Lock()
		os.Exit(0)
	}
	signalSet := signalset.New()
	signalSet.Register(syscall.SIGINT, sigHandler)
	signalSet.Register(syscall.SIGTERM, sigHandler)
	sigChan := make(chan os.Signal, account.MaxSignalQueueLen)
	signal.Notify(sigChan)
	for {
		select {
		case sig := <-sigChan:
			signalSet.Handle(sig, nil)
		}
	}
}

func walletAction(c *cli.Context) error {
	if c.NumFlags() == 0 {
		cli.ShowSubcommandHelp(c)
		return nil
	}
	// wallet name is wallet.dat by default
	name := c.String("name")
	if name == "" {
		fmt.Fprintln(os.Stderr, "invalid wallet name")
		os.Exit(1)
	}
	passwd := c.String("password")
	// create wallet
	if c.Bool("create") {
		if FileExisted(name) {
			fmt.Printf("CAUTION: '%s' already exists!\n", name)
			os.Exit(1)
		} else {
			wallet, err := account.Create(name, getConfirmedPassword(passwd))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			showAccountsInfo(wallet)
		}
		return nil
	}

	// list wallet info
	if item := c.String("list"); item != "" {
		if item != "account" && item != "mainaccount" && item != "balance" && item != "verbose" && item != "multisig" && item != "height" {
			fmt.Fprintln(os.Stderr, "--list [account | balance | verbose | multisig | height]")
			os.Exit(1)
		} else {
			wallet, err := account.Open(name, getPassword(passwd))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			switch item {
			case "account":
				showAccountsInfo(wallet)
			case "mainaccount":
				showDefaultAccountInfo(wallet)
			case "balance":
				showBalancesInfo(wallet)
			case "verbose":
				showVerboseInfo(wallet)
			case "multisig":
				showMultisigInfo(wallet)
			case "height":
				showHeightInfo(wallet)
			}
		}
		return nil
	}

	// change password
	if c.Bool("changepassword") {
		fmt.Printf("Wallet File: '%s'\n", name)
		passwd, _ := password.GetPassword()
		wallet, err := account.Open(name, passwd)
		if err != nil {
			os.Exit(1)
		}
		fmt.Println("# input new password #")
		newPassword, _ := password.GetConfirmedPassword()
		if ok := wallet.ChangePassword([]byte(passwd), newPassword); !ok {
			fmt.Fprintln(os.Stderr, "failed to change password")
			os.Exit(1)
		}
		fmt.Println("password changed")

		return nil
	}

	// rebuild index
	if c.Bool("reset") {
		wallet, err := account.Open(name, getPassword(passwd))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := wallet.Rebuild(); err != nil {
			fmt.Fprintln(os.Stderr, "delete coins info from wallet file error")
			os.Exit(1)
		}
		fmt.Printf("%s was reset successfully\n", name)

		return nil
	}

	// add accounts
	if num := c.Int("addaccount"); num > 0 {
		wallet, err := account.Open(name, getPassword(passwd))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		go processSignals(wallet)
		time.Sleep(500 * time.Millisecond)
		for i := 0; i < num; i++ {
			account, err := wallet.CreateAccount()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := wallet.CreateContract(account); err != nil {
				wallet.DeleteAccount(account.ProgramHash)
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		fmt.Printf("%d accounts created\n", num)
		return nil
	}

	// add multisig account
	multikeys := c.String("addmultisigaccount")
	if multikeys != "" {
		publicKeys := strings.Split(multikeys, ":")
		if len(publicKeys) < MinMultiSignKey {
			fmt.Fprintln(os.Stderr, "public keys is not enough")
			os.Exit(1)
		}
		var keys []*crypto.PubKey
		for _, v := range publicKeys {
			byteKey, err := HexStringToBytes(v)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid public key")
				os.Exit(1)
			}
			rawKey, err := crypto.DecodePoint(byteKey)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid encoded public key")
				os.Exit(1)
			}
			keys = append(keys, rawKey)
		}
		wallet, err := account.Open(name, getPassword(passwd))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		mainAccount, err := wallet.GetDefaultAccount()
		if err != nil {
			fmt.Fprintln(os.Stderr, "wallet is broken, main account missing")
			os.Exit(1)
		}
		// generate M/N multsig contract
		// M = N/2+1
		// M/N could be 2/3, 3/4, 3/5, 4/6, 4/7 ...
		var M = len(keys)/2 + 1
		if err := wallet.CreateMultiSignContract(mainAccount.ProgramHash, M, keys); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("a multisig account created\n")
	}

	// set wallet height
	if height := c.Uint("setheight"); height >= 0 {
		wallet, err := account.Open(name, getPassword(passwd))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		h := make([]byte, 4)
		binary.LittleEndian.PutUint32(h, uint32(height))
		if err := wallet.SaveStoredData("Height", h[:]); err != nil {
			fmt.Println("set wallet height error: ", err)
			os.Exit(1)
		}
		fmt.Println("wallet current height is ", height)

		return nil
	}

	return nil
}

func NewCommand() *cli.Command {
	return &cli.Command{
		Name:        "wallet",
		Usage:       "user wallet operation",
		Description: "With nodectl wallet, you could control your asset.",
		ArgsUsage:   "[args]",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "create, c",
				Usage: "create wallet",
			},
			cli.StringFlag{
				Name:  "list, l",
				Usage: "list wallet information [account, mainaccount, balance, verbose, multisig, height]",
			},
			cli.IntFlag{
				Name:  "addaccount",
				Usage: "add new account address",
			},
			cli.StringFlag{
				Name:  "addmultisigaccount",
				Usage: "add new multi-sign account address",
			},
			cli.UintFlag{
				Name:  "setheight",
				Usage: "set wallet height",
			},
			cli.BoolFlag{
				Name:  "changepassword",
				Usage: "change wallet password",
			},
			cli.BoolFlag{
				Name:  "reset",
				Usage: "reset wallet",
			},
			cli.StringFlag{
				Name:  "name, n",
				Usage: "wallet name",
				Value: account.WalletFileName,
			},
			cli.StringFlag{
				Name:  "password, p",
				Usage: "wallet password",
			},
		},
		Action: walletAction,
		OnUsageError: func(c *cli.Context, err error, isSubcommand bool) error {
			PrintError(c, err, "wallet")
			return cli.NewExitError("", 1)
		},
	}
}
