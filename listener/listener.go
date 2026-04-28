package listener

import (
	"context"
	"log"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ikem-legend/blockchain-indexer/config"
	"github.com/ikem-legend/blockchain-indexer/decoder"
	"github.com/ikem-legend/blockchain-indexer/models"
	"github.com/ikem-legend/blockchain-indexer/storage"
)

type Listener struct {
    client    *ethclient.Client
    config    *config.Config
    decoder   decoder.Decoder
    storage   storage.Storage
}

func New(cfg *config.Config, dec decoder.Decoder, stor storage.Storage) (*Listener, error) {
	client, err := ethclient.Dial(cfg.RPCUrl)
	if err != nil {
		log.Fatal("Error dialing network:", err)
		return nil, err
	}
	return &Listener{
		client: client,
		config: cfg,
		decoder: dec,
		storage: stor,
	}, nil
}

func (l *Listener) Start (ctx context.Context) {
	ticker := time.NewTicker(time.Duration(l.config.PollInterval) * time.Second)
	defer ticker.Stop()

	var lastBlockNumber uint64 = 24975100
	var recentBlockNumber uint64 = 24975100

	for {
		select {
		case <-ctx.Done():
			log.Println("Listener stopped")
			return
		case <-ticker.C:
			logs, err := l.fetchLogs(ctx, lastBlockNumber, recentBlockNumber)
			if err != nil {
				log.Printf("Error reading logs: %v\n", err)
				continue
			}

			for _, rawLog := range logs {
				decodedEvent, err := l.decoder.Decode(&rawLog)
				if err != nil {
					log.Printf("Error decoding event: %v\n", err)
					continue
				}

				err = l.storage.SaveEvent(ctx, decodedEvent)
				if err != nil {
					log.Printf("Error saving event: %v\n", err)
				}
			}

			if len(logs) > 0 {
				lastBlockNumber = logs[len(logs) - 1].BlockNumber
				log.Printf("Processed %d events. Latest block %d", len(logs), lastBlockNumber)
			}
		}
	}
}

func (l *Listener) fetchLogs(ctx context.Context, fromBlock uint64, toBlock uint64) ([]models.RawLog, error) {
	query := ethereum.FilterQuery{
		Addresses: []common.Address{
			common.HexToAddress(l.config.ContractAddr),
		},
		FromBlock: new(big.Int).SetUint64(fromBlock),
		ToBlock: new(big.Int).SetUint64(toBlock),
	}
	logs, clientErr := l.client.FilterLogs(ctx, query)
	if clientErr != nil {
		log.Fatal("Error retrieving logs:", clientErr)
	}

	rawLogs := make([]models.RawLog, len(logs))
	for i, lg := range logs {
		topics := make([]string, len(lg.Topics))
		for j, t := range lg.Topics {
			topics[j] = t.Hex()
		}
		rawLogs[i] = models.RawLog{
			Address:     lg.Address.Hex(),
			Topics:      topics,
			Data:        common.Bytes2Hex(lg.Data),
			BlockNumber: lg.BlockNumber,
			TxHash:      lg.TxHash.Hex(),
			Index:       lg.Index,
		}
	}

	return rawLogs, nil
}
