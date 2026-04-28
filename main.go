package main

import (
	"context"
	"log"

	"github.com/ikem-legend/blockchain-indexer/api"
	"github.com/ikem-legend/blockchain-indexer/config"
	"github.com/ikem-legend/blockchain-indexer/decoder"
	"github.com/ikem-legend/blockchain-indexer/listener"
	"github.com/ikem-legend/blockchain-indexer/storage"
)

func main() {
	cfg := config.Load()

	// Initialize storage
	stor, err := storage.NewSQLiteStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v\n", err)
	}
	defer stor.Close()

	// Initialize decoder
	decoder, err := decoder.NewEventDecoder("./decoder/abi/usdt.abi")
	if err != nil {
		log.Fatalf("Failed to initialize decoder: %v\n", err)
	}

	// Initialize listener
	lis, err := listener.New(cfg, decoder, stor)
	if err != nil {
		log.Fatalf("Failed to initialize listener: %v\n", err)
	}

	// Start API server in Goroutine
	srv := api.New(stor, cfg.HTTPPort)
	go srv.Start()

	// Start listening for events
	ctx := context.Background()
	lis.Start(ctx)
}
