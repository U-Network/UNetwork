package httpjsonrpc

var (
	UgcNetworkRpcInvalidHash        = responsePacking("invalid hash")
	UgcNetworkRpcInvalidBlock       = responsePacking("invalid block")
	UgcNetworkRpcInvalidTransaction = responsePacking("invalid transaction")
	UgcNetworkRpcInvalidParameter   = responsePacking("invalid parameter")

	UgcNetworkRpcUnknownBlock       = responsePacking("unknown block")
	UgcNetworkRpcUnknownTransaction = responsePacking("unknown transaction")

	UgcNetworkRpcNil           = responsePacking(nil)
	UgcNetworkRpcUnsupported   = responsePacking("Unsupported")
	UgcNetworkRpcInternalError = responsePacking("internal error")
	UgcNetworkRpcIOError       = responsePacking("internal IO error")
	UgcNetworkRpcAPIError      = responsePacking("internal API error")
	UgcNetworkRpcSuccess       = responsePacking(true)
	UgcNetworkRpcFailed        = responsePacking(false)

	UgcNetworkRpc = responsePacking
)
