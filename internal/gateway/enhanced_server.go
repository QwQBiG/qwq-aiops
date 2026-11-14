package gateway

// EnhancedGatewayServer keeps the existing constructor compatible while using
// the same console proxy and lifecycle as GatewayServer.
type EnhancedGatewayServer = GatewayServer

func NewEnhancedGatewayServer(address string) *EnhancedGatewayServer {
	return NewGatewayServer(address)
}
