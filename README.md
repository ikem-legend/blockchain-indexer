# Blockchain Indexer

A lightweight Ethereum event indexer written in Go. It polls an Ethereum JSON-RPC endpoint for logs emitted by a configured contract, decodes them against a local ABI, persists them to SQLite, and exposes the indexed events over an HTTP API.

By default the project ships with the USDT ABI (`decoder/abi/usdt.abi`) and is wired up against Ethereum mainnet via Infura, but any contract + ABI pair can be swapped in.

## Architecture

The application is composed of five packages that are wired together in `main.go`:

```
                       ┌──────────────┐
                       │    config    │  loads .env + defaults
                       └──────┬───────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
  │   listener   │───▶│   decoder    │───▶│   storage    │
  │ (poll RPC)   │    │ (parse ABI)  │    │   (SQLite)   │
  └──────────────┘    └──────────────┘    └──────┬───────┘
                                                 │
                                                 ▼
                                          ┌──────────────┐
                                          │     api      │  HTTP server
                                          └──────────────┘
```

- **`config`** — Loads configuration from `.env` and environment variables. Builds the RPC URL by combining a provider base URL (`config/constants.go`) with the user's API key.
- **`listener`** — Owns the `ethclient.Client` connection. On a fixed `PollInterval` tick it calls `eth_getLogs` (via `FilterLogs`) for the configured contract, hands each log to the decoder, and persists the decoded event via storage. Tracks the last processed block in memory.
- **`decoder`** — Parses an ABI file once at startup and builds a map keyed by event signature hash (`topics[0]`). On each log it looks up the event, unpacks indexed args from `topics[1:]` and non-indexed args from `data`, returning a `DecodedEvent`.
- **`storage`** — `Storage` interface with a SQLite implementation. Creates the `events` table on first run, marshals the decoded `data` map to JSON for storage, and reads it back on query.
- **`api`** — `gorilla/mux` HTTP server exposing read-only endpoints over the indexed events. Runs in its own goroutine alongside the listener.
- **`models`** — Shared data structs (`RawLog`, `DecodedEvent`) used across packages to avoid circular dependencies.

### Data flow

1. Listener ticks every `POLL_INTERVAL` seconds.
2. Fetches logs for the configured contract between the last processed block and the current target block.
3. Each raw log is decoded against the loaded ABI into a `DecodedEvent`.
4. The decoded event is inserted into SQLite.
5. The HTTP API serves the persisted events from SQLite on demand.

## Configuration

Configuration is read from a `.env` file at the project root (or from process environment variables). Copy `.env.example` and fill in the values:

```bash
cp .env.example .env
```

| Variable           | Required | Default        | Description                                                                 |
| ------------------ | -------- | -------------- | --------------------------------------------------------------------------- |
| `INFURA_API_KEY`   | Yes      | —              | Infura project key. Appended to the Infura mainnet base URL.                |
| `CONTRACT_ADDRESS` | Yes      | —              | Hex address of the contract whose logs should be indexed (e.g. USDT).       |
| `HTTP_PORT`        | No       | `8080`         | Port the API server listens on.                                             |
| `DB_PATH`          | No       | `./events.db`  | Path to the SQLite database file. Created automatically if absent.          |
| `POLL_INTERVAL`    | No       | `30`           | Seconds between RPC polls.                                                  |

The RPC provider is currently hardcoded to Infura mainnet in `config/constants.go`. To target a different network or provider, update `Providers` there.

### ABI

The decoder loads its ABI from `./decoder/abi/usdt.abi` (path is hardcoded in `main.go`). To index a different contract, replace that file with the target contract's ABI JSON, or edit the path passed to `decoder.NewEventDecoder`.

## Running the app

### Prerequisites

- Go 1.25+
- A C toolchain (required by `mattn/go-sqlite3`, which is CGo-based) — Xcode Command Line Tools on macOS, `build-essential` on Debian/Ubuntu.
- An Infura API key (or another RPC provider configured in `config/constants.go`).

### Install dependencies

```bash
go mod download
```

### Run

```bash
go run .
```

Or build a binary:

```bash
go build -o blockchain-indexer .
./blockchain-indexer
```

On startup the indexer will:
- Open (or create) the SQLite DB at `DB_PATH`.
- Load the ABI and build the event signature cache.
- Dial the configured RPC endpoint.
- Start the HTTP API on `HTTP_PORT`.
- Begin polling for logs every `POLL_INTERVAL` seconds.

## HTTP API

| Method | Path       | Description                                              |
| ------ | ---------- | -------------------------------------------------------- |
| GET    | `/`        | Service banner.                                          |
| GET    | `/health`  | Liveness probe; returns `{"status":"ok"}`.               |
| GET    | `/events`  | Returns indexed events as a JSON array (limit 100).      |

Example:

```bash
curl http://localhost:8080/events
```

## Database schema

A single `events` table is created on first run:

```sql
CREATE TABLE IF NOT EXISTS events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    contract_address TEXT,
    event_name       TEXT,
    block_number     INTEGER,
    tx_hash          TEXT,
    data             TEXT,        -- decoded args, JSON-encoded
    timestamp        DATETIME
);
```

The `data` column stores the decoded indexed and non-indexed arguments as a JSON object keyed by argument name.

## Project layout

```
.
├── main.go               # wires config, storage, decoder, listener, and api
├── api/                  # HTTP server (gorilla/mux)
├── config/               # env loading + RPC provider constants
├── decoder/              # ABI loading + log decoding
│   └── abi/usdt.abi      # default ABI shipped with the project
├── listener/             # RPC polling loop
├── models/               # shared structs (RawLog, DecodedEvent)
├── storage/              # Storage interface + SQLite implementation
├── go.mod / go.sum
├── .env.example
└── LICENSE.md
```

## Notes & limitations

- The starting block is currently hardcoded in `listener.Start` (`lastBlockNumber`/`recentBlockNumber`). For a long-running deployment you'd typically persist the last processed block in storage and resume from it on startup.
- The decoder path (`./decoder/abi/usdt.abi`) is relative to the working directory the binary is launched from.
- Only one ABI/contract is indexed per process.

## License

See [LICENSE.md](./LICENSE.md).
