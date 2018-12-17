package ethereum

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/urfave/cli.v1"

	"github.com/U-Network/UNetwork/api"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	ethUtils "github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

//StartNewEthereum Create and launch the Ethereum node framework service
func StartNewEthereum(context *cli.Context) *node.Node {
	emNode := MakeFullNode(context)
	startNode(context, emNode)
	return emNode
}

// Start starts base node and stop p2p server
func Start(node *node.Node) error {
	// start p2p server
	err := node.Start()
	if err != nil {
		return err
	}
	// stop it
	node.Server().Stop()
	return nil
}

// startNode The logic copied from Ethereum is mainly
// used to launch the Ethereum node service and unlock specific accounts.
func startNode(ctx *cli.Context, stack *node.Node) {
	Start(stack)

	// Unlock any account specifically requested
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	passwords := ethUtils.MakePasswordList(ctx)
	unlocks := strings.Split(ctx.GlobalString(ethUtils.UnlockedAccountFlag.Name), ",")
	for i, account := range unlocks {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, ks, trimmed, i, passwords)
		}
	}
	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	go func() {
		// Create an chain state reader for self-derivation
		rpcClient, err := stack.Attach()
		if err != nil {
			ethUtils.Fatalf("Failed to attach to self: %v", err)
		}
		stateReader := ethclient.NewClient(rpcClient)

		// Open and self derive any wallets already attached
		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			} else {
				wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			if event.Kind == accounts.WalletArrived {
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url",
						event.Wallet.URL(), "err", err)
				} else {
					status, _ := event.Wallet.Status()
					log.Info("New wallet appeared", "url", event.Wallet.URL(),
						"status", status)
					event.Wallet.SelfDerive(accounts.DefaultBaseDerivationPath,
						stateReader)
				}
			} else {
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()
}

// tries unlocking the specified account a few times.
// nolint: unparam
func unlockAccount(ctx *cli.Context, ks *keystore.KeyStore, address string, i int,
	passwords []string) (accounts.Account, string) {

	account, err := ethUtils.MakeAddress(ks, address)
	if err != nil {
		ethUtils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	ethUtils.Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

// getPassPhrase retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
// nolint: unparam
func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		log.Info(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		ethUtils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			ethUtils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			ethUtils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError,
	auth string) accounts.Account {

	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your passphrase against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := ks.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		ethUtils.Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your passphrase unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}

///////////////////////////////////////////////////////

// MakeFullNode creates a full go-ethereum node
// #unstable
func MakeFullNode(ctx *cli.Context) *node.Node {

	ethcnf := eth.DefaultConfig
	config := DefaultNodeConfig()

	ethUtils.SetNodeConfig(ctx, &config)

	stack, err := node.New(&config)
	if err != nil {
	}
	ethUtils.SetEthConfig(ctx, stack, &ethcnf)

	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		return api.NewBackend(ctx, &ethcnf)
	}); err != nil {
		ethUtils.Fatalf("Failed to register the ABCI application service: %v", err)
	}

	return stack
}

// DefaultNodeConfig returns the default configuration for a go-ethereum node
// #unstable
func DefaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = "unetwork_vm"
	cfg.Version = params.Version
	cfg.HTTPModules = append(cfg.HTTPModules, "eth", "net", "web3", "personal", "freegas")
	cfg.WSModules = append(cfg.WSModules, "eth", "net", "web3", "personal", "freegas")
	cfg.IPCPath = "unetwork.ipc"

	emHome := os.Getenv("EMHOME")
	if emHome != "" {
		cfg.DataDir = emHome
	}

	return cfg
}
