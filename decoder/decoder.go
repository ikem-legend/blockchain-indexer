package decoder

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ikem-legend/blockchain-indexer/models"
)

type Decoder interface {
	Decode(rawLog *models.RawLog) (*models.DecodedEvent, error)
}

type EventDecoder struct {
	abi abi.ABI // Parsed ABI
	abiPath string // Location of ABI file
	cache   map[string]abi.Event // Event signature cache
}

// Constructor creates a new decoder with state
func NewEventDecoder(abiPath string) (*EventDecoder, error) {
    // Load and parse ABI
	abiBytes, err := os.ReadFile(abiPath)
	if err != nil {
		return nil, fmt.Errorf("Error reading ABI file: %w", err)
	}
	contractABI, err := abi.JSON(strings.NewReader(string(abiBytes)))
	if err != nil {
		return nil, fmt.Errorf("Error parsing ABI data: %w", err)
	}
    // Initialize cache
	eventCache := make(map[string]abi.Event)
	for _, event := range contractABI.Events {
		sigHash := event.ID.Hex()
		eventCache[sigHash] = event
	}
    return &EventDecoder{
		abi: contractABI,
		abiPath: abiPath,
		cache: eventCache,
	}, nil
}

func (d *EventDecoder) Decode(rawLog *models.RawLog) (*models.DecodedEvent, error) {
	if len(rawLog.Topics) == 0 {
		return nil, fmt.Errorf("Event topics not found")
	}
	if _, hexErr := common.ParseHexOrString(rawLog.Data); hexErr != nil {
		return nil, fmt.Errorf("Data is not hex encoded: %w", hexErr)
	}

	eventSigHash := rawLog.Topics[0]
	val, ok := d.cache[eventSigHash]
	if !ok {
		return nil, fmt.Errorf("Unknown event signature: %s", eventSigHash)
	}

	// Decode indexed args only
	var indexedArgs []abi.Argument
	for _, input := range val.Inputs {
		if input.Indexed {
			indexedArgs = append(indexedArgs, input)
		}
	}

	decodedData := make(map[string]interface{})
	
	// Match each indexed arg to its topic
	// topics[0] is the event signature, so indexed args start at topics[1]
	for i, arg := range indexedArgs {
		if i + 1 >= len(rawLog.Topics) {
			return nil, fmt.Errorf("topic missing for indexed arg %s (need index %d, have %d topics)", arg.Name, i+1, len(rawLog.Topics))
		}
		topicData := rawLog.Topics[i+1]
		topicBytes := common.HexToHash(topicData).Bytes()

		// Unpack the value according to its ABI type
		// Since the topic bytes are already the raw value, the indexed arguments are flipped to false,
		argCopy := arg
		argCopy.Indexed = false
		tempArgs := abi.Arguments{argCopy}

		values, err := tempArgs.Unpack(topicBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse indexed arg %s: %w", arg.Name, err)
		}
		decodedData[arg.Name] = values[0]
	}

	// Decoded non-indexed args
	var nonIndexedArgs abi.Arguments
	for _, input := range val.Inputs {
		if !input.Indexed {
			nonIndexedArgs = append(nonIndexedArgs, input)
		}
	}

	if len(nonIndexedArgs) > 0 && len(rawLog.Data) > 2 {
		dataBytes := common.FromHex(rawLog.Data)
		values, err := nonIndexedArgs.Unpack(dataBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode non-indexed args: %w", err)
		}
		for i, arg := range nonIndexedArgs {
			decodedData[arg.Name] = values[i]
		}
	}

	return &models.DecodedEvent{
		ContractAddr: rawLog.Address,
		EventName: val.Name, 
		BlockNumber: rawLog.BlockNumber,
		TxHash: rawLog.TxHash,
		Data: decodedData,
		Timestamp: time.Now(),
	}, nil
}
