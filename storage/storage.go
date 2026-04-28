package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/ikem-legend/blockchain-indexer/models"
	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
    SaveEvent(ctx context.Context, event *models.DecodedEvent) error
    GetEvents(ctx context.Context, limit int) ([]*models.DecodedEvent, error)
    Close() error
}

type SQLiteStorage struct {
    db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		contract_address TEXT,
		event_name TEXT,
		block_number INTEGER,
		tx_hash TEXT,
		data TEXT,
		timestamp DATETIME
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) SaveEvent(ctx context.Context, event *models.DecodedEvent) error {
	dataJSON, _ := json.Marshal(event.Data)
	_, err := s.db.ExecContext(ctx, 
		`INSERT INTO events (contract_address, event_name, block_number, tx_hash, data)
		VALUES(?, ?, ?, ?, ?)`,
		event.ContractAddr, event.EventName, event.BlockNumber, event.TxHash, string(dataJSON),
	)
	return err
}

func (s *SQLiteStorage) GetEvents(ctx context.Context, limit int) ([]*models.DecodedEvent, error) {
	rows, err := s.db.QueryContext(ctx, 
		`SELECT * FROM events`,
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var events []*models.DecodedEvent
	for rows.Next() {
		var event models.DecodedEvent
		var dataJSON string
		err := rows.Scan(&event.ContractAddr, &event.EventName, &event.BlockNumber, &event.TxHash, &dataJSON)
		if err != nil {
			log.Fatal(err)
		}
		json.Unmarshal([]byte(dataJSON), &event.Data)
		log.Printf("Event: %s, Data: %s", event.EventName, dataJSON)
		events = append(events, &event)
	}
	
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}

	return events, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
