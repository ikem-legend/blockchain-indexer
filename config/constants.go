package config

type RPCProvider string

type RPCUrl map[RPCProvider]string

const (
	InfuraRPCProvider RPCProvider = "INFURA"
)

var Providers = RPCUrl{
	InfuraRPCProvider: "https://mainnet.infura.io/v3/",
}
