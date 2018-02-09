package httpjsonrpc

var (
	UnetworkRpcInvalidHash        = responsePacking("invalid hash")
	UnetworkRpcInvalidBlock       = responsePacking("invalid block")
	UnetworkRpcInvalidTransaction = responsePacking("invalid transaction")
	UnetworkRpcInvalidParameter   = responsePacking("invalid parameter")

	UnetworkRpcUnknownBlock       = responsePacking("unknown block")
	UnetworkRpcUnknownTransaction = responsePacking("unknown transaction")

	UnetworkRpcNil           = responsePacking(nil)
	UnetworkRpcUnsupported   = responsePacking("Unsupported")
	UnetworkRpcInternalError = responsePacking("internal error")
	UnetworkRpcIOError       = responsePacking("internal IO error")
	UnetworkRpcAPIError      = responsePacking("internal API error")
	UnetworkRpcSuccess       = responsePacking(true)
	UnetworkRpcFailed        = responsePacking(false)

	// error code for wallet
	UnetworkRpcWalletAlreadyExists = responsePacking("wallet already exist")
	UnetworkRpcWalletNotExists     = responsePacking("wallet doesn't exist")

	UnetworkRpc = responsePacking
)
