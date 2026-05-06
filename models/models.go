package models

import "time"

type RawLog struct {
    Address     string
    Topics      []string
    Data        string
    BlockNumber uint64
    TxHash      string
    Index       uint
}

type DecodedEvent struct {
    ID          int64             `json:"id"`
    ContractAddr string           `json:"contract_address"`
    EventName   string            `json:"event_name"`
    BlockNumber uint64            `json:"block_number"`
    TxHash      string            `json:"tx_hash"`
    Data        map[string]interface{} `json:"data"`
    Timestamp   time.Time         `json:"timestamp"`
}