package httpjsonrpc

var (
	uNetworkRpcInvalidHash        = responsePacking("invalid hash")
	uNetworkRpcInvalidBlock       = responsePacking("invalid block")
	uNetworkRpcInvalidTransaction = responsePacking("invalid transaction")
	uNetworkRpcInvalidParameter   = responsePacking("invalid parameter")

	uNetworkRpcUnknownBlock       = responsePacking("unknown block")
	uNetworkRpcUnknownTransaction = responsePacking("unknown transaction")

	uNetworkRpcNil           = responsePacking(nil)
	uNetworkRpcUnsupported   = responsePacking("Unsupported")
	uNetworkRpcInternalError = responsePacking("internal error")
	uNetworkRpcIOError       = responsePacking("internal IO error")
	uNetworkRpcAPIError      = responsePacking("internal API error")
	uNetworkRpcSuccess       = responsePacking(true)
	uNetworkRpcFailed        = responsePacking(false)

	// error code for wallet
	uNetworkRpcWalletAlreadyExists = responsePacking("wallet already exist")
	uNetworkRpcWalletNotExists     = responsePacking("wallet doesn't exist")

	uNetworkRpc = responsePacking
)
