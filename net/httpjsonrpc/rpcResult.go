package httpjsonrpc

var (
	UNetworkRPCInvalidHash        = responsePacking("invalid hash")
	UNetworkRPCInvalidBlock       = responsePacking("invalid block")
	UNetworkRPCInvalidTransaction = responsePacking("invalid transaction")
	UNetworkRPCInvalidParameter   = responsePacking("invalid parameter")

	UNetworkRPCUnknownBlock       = responsePacking("unknown block")
	UNetworkRPCUnknownTransaction = responsePacking("unknown transaction")

	UNetworkRPCNil           = responsePacking(nil)
	UNetworkRPCUnsupported   = responsePacking("Unsupported")
	UNetworkRPCInternalError = responsePacking("internal error")
	UNetworkRPCIOError       = responsePacking("internal IO error")
	UNetworkRPCAPIError      = responsePacking("internal API error")
	UNetworkRPCSuccess       = responsePacking(true)
	UNetworkRPCFailed        = responsePacking(false)

	// error code for wallet
	UNetworkRPCWalletAlreadyExists = responsePacking("wallet already exist")
	UNetworkRPCWalletNotExists     = responsePacking("wallet doesn't exist")

	UNetworkRPC = responsePacking
)
