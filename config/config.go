package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
    RPCUrl      string
    ContractAddr string
    DBPath      string
    HTTPPort    string
    PollInterval int // seconds
}

func Load() *Config {
    godotenv.Load()

    pollInterval := 30
    if val := os.Getenv("POLL_INTERVAL"); val != "" {
        if parsedVal, err := strconv.Atoi(val); err != nil {
            pollInterval = parsedVal
        }
    }

    return &Config{
        RPCUrl:       Providers[InfuraRPCProvider] + getEnvOrDefault("INFURA_API_KEY", ""),
        ContractAddr: getEnvOrDefault("CONTRACT_ADDRESS", ""), // USDT token example
        HTTPPort:     getEnvOrDefault("HTTP_PORT", "8080"),
        DBPath:       getEnvOrDefault("DB_PATH", "./events.db"),
        PollInterval: pollInterval,
    }
}

func getEnvOrDefault(key, defaultValue string) string {
    if val := os.Getenv(key); val != "" {
        fmt.Printf("%s: %s\n", key, val)
        return val
    }
    fmt.Printf("%s: %s\n", key, defaultValue)
    return defaultValue
}
