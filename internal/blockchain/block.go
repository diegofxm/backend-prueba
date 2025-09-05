package blockchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Block representa un bloque en la blockchain SECOP
type Block struct {
	Index        int                    `json:"index"`
	Timestamp    time.Time              `json:"timestamp"`
	Data         map[string]interface{} `json:"data"`
	PreviousHash string                 `json:"previous_hash"`
	Hash         string                 `json:"hash"`
	Nonce        int                    `json:"nonce"`
	Type         string                 `json:"type"` // Tipo de bloque: CONTRACT_CREATION, VALIDATION, etc.
}

// Contract representa un contrato estatal
type Contract struct {
	ID              string  `json:"id"`
	EntityCode      string  `json:"entity_code"`
	EntityName      string  `json:"entity_name"`
	ContractType    string  `json:"contract_type"`
	Description     string  `json:"description"`
	Amount          float64 `json:"amount"`
	Status          string  `json:"status"`
	CreatedBy       string  `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	ValidationNodes []string `json:"validation_nodes"`
}

// NewBlock crea un nuevo bloque
func NewBlock(data map[string]interface{}, previousHash string) *Block {
	block := &Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         data,
		PreviousHash: previousHash,
		Nonce:        0,
	}
	
	block.Hash = block.calculateHash()
	return block
}

// calculateHash calcula el hash SHA-256 del bloque
func (b *Block) calculateHash() string {
	record := map[string]interface{}{
		"index":         b.Index,
		"timestamp":     b.Timestamp.Unix(),
		"data":          b.Data,
		"previous_hash": b.PreviousHash,
		"nonce":         b.Nonce,
		"type":          b.Type,
	}
	
	recordBytes, _ := json.Marshal(record)
	hash := sha256.Sum256(recordBytes)
	return hex.EncodeToString(hash[:])
}

// IsValid verifica si el bloque es válido
func (b *Block) IsValid() bool {
	return b.Hash == b.calculateHash()
}
