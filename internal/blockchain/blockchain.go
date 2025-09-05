package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Blockchain representa la cadena de bloques SECOP
type Blockchain struct {
	Chain     []*Block             `json:"chain"`
	Contracts map[string]*Contract `json:"contracts"`
}

// NewBlockchain crea una nueva blockchain con bloque génesis
func NewBlockchain() *Blockchain {
	genesisBlock := &Block{
		Index:        0,
		Timestamp:    time.Now(),
		Data:         map[string]interface{}{"message": "SECOP Blockchain Genesis Block"},
		PreviousHash: "",
		Nonce:        0,
	}
	genesisBlock.Hash = genesisBlock.calculateHash()

	return &Blockchain{
		Chain:     []*Block{genesisBlock},
		Contracts: make(map[string]*Contract),
	}
}

// AddContract agrega un nuevo contrato a la blockchain
func (bc *Blockchain) AddContract(contract *Contract) error {
	// Validar contrato
	if err := bc.validateContract(contract); err != nil {
		return err
	}

	// Generar ID único si no existe
	if contract.ID == "" {
		contract.ID = uuid.New().String()
	}

	// Establecer timestamp
	contract.CreatedAt = time.Now()
	contract.Status = "CREADO"

	// Crear bloque para el contrato
	blockData := map[string]interface{}{
		"type":     "CONTRACT_CREATION",
		"contract": contract,
	}

	block := NewBlock(blockData, bc.getLatestBlock().Hash)
	block.Index = len(bc.Chain)
	block.Type = "CONTRACT_CREATION" // Establecer tipo de bloque
	block.Hash = block.calculateHash()

	// Agregar bloque a la cadena
	bc.Chain = append(bc.Chain, block)
	bc.Contracts[contract.ID] = contract

	fmt.Printf("✅ Contrato %s agregado al bloque %d\n", contract.ID, block.Index)
	return nil
}

// ValidateContract valida un contrato por parte de un nodo
func (bc *Blockchain) ValidateContract(contractID string, nodeID string, approved bool, reason string) error {
	contract, exists := bc.Contracts[contractID]
	if !exists {
		return errors.New("contrato no encontrado")
	}

	// Crear evento de validación
	validationData := map[string]interface{}{
		"type":        "CONTRACT_VALIDATION",
		"contract_id": contractID,
		"node_id":     nodeID,
		"approved":    approved,
		"reason":      reason,
		"timestamp":   time.Now(),
	}

	block := NewBlock(validationData, bc.getLatestBlock().Hash)
	block.Index = len(bc.Chain)
	bc.Chain = append(bc.Chain, block)

	// Actualizar estado del contrato
	if approved {
		contract.ValidationNodes = append(contract.ValidationNodes, nodeID)
		if len(contract.ValidationNodes) >= 4 { // Requiere 4 validaciones internas
			contract.Status = "APROBADO_INTERNAMENTE"
		}
	}

	status := "RECHAZADO"
	if approved {
		status = "APROBADO"
	}

	fmt.Printf("✅ Validación %s para contrato %s por nodo %s\n", status, contractID, nodeID)
	return nil
}

// GetContract obtiene un contrato por ID
func (bc *Blockchain) GetContract(contractID string) (*Contract, error) {
	contract, exists := bc.Contracts[contractID]
	if !exists {
		return nil, errors.New("contrato no encontrado")
	}
	return contract, nil
}

// GetAllContracts obtiene todos los contratos
func (bc *Blockchain) GetAllContracts() []*Contract {
	contracts := make([]*Contract, 0, len(bc.Contracts))
	for _, contract := range bc.Contracts {
		contracts = append(contracts, contract)
	}
	return contracts
}

// IsChainValid verifica la integridad de la blockchain
func (bc *Blockchain) IsChainValid() bool {
	for i := 1; i < len(bc.Chain); i++ {
		currentBlock := bc.Chain[i]
		previousBlock := bc.Chain[i-1]

		// Verificar hash del bloque actual
		if !currentBlock.IsValid() {
			return false
		}

		// Verificar enlace con bloque anterior
		if currentBlock.PreviousHash != previousBlock.Hash {
			return false
		}
	}
	return true
}

// getLatestBlock obtiene el último bloque de la cadena
func (bc *Blockchain) getLatestBlock() *Block {
	return bc.Chain[len(bc.Chain)-1]
}

// validateContract valida los datos del contrato
func (bc *Blockchain) validateContract(contract *Contract) error {
	if contract.EntityCode == "" {
		return errors.New("código de entidad requerido")
	}
	if contract.EntityName == "" {
		return errors.New("nombre de entidad requerido")
	}
	if contract.Description == "" {
		return errors.New("descripción requerida")
	}
	if contract.Amount <= 0 {
		return errors.New("monto debe ser mayor a cero")
	}
	if contract.CreatedBy == "" {
		return errors.New("creador requerido")
	}
	return nil
}

// IsValidBlock valida si un bloque es válido
func (bc *Blockchain) IsValidBlock(block Block) bool {
	// Verificar que el hash no esté vacío
	if block.Hash == "" {
		return false
	}
	
	// Verificar que el timestamp sea razonable
	if block.Timestamp.IsZero() {
		return false
	}
	
	// Verificar que tenga un hash previo válido (excepto el bloque génesis)
	if len(bc.Chain) > 0 && block.PreviousHash != bc.Chain[len(bc.Chain)-1].Hash {
		return false
	}
	
	return true
}

// HasBlock verifica si ya tenemos un bloque con el hash dado
func (bc *Blockchain) HasBlock(hash string) bool {
	for _, block := range bc.Chain {
		if block.Hash == hash {
			return true
		}
	}
	return false
}

// AddBlock agrega un bloque existente a la cadena
func (bc *Blockchain) AddBlock(block Block) error {
	// Verificar que el bloque sea válido
	if !bc.IsValidBlock(block) {
		return errors.New("bloque inválido")
	}
	
	// Verificar que no tengamos ya este bloque
	if bc.HasBlock(block.Hash) {
		return errors.New("bloque ya existe")
	}
	
	// Agregar el bloque
	bc.Chain = append(bc.Chain, &block)
	
	// Si es un bloque de contrato, agregarlo al mapa de contratos
	if block.Type == "CONTRACT_CREATION" {
		var contract Contract
		err := json.Unmarshal([]byte(fmt.Sprintf("%v", block.Data)), &contract)
		if err == nil {
			bc.Contracts[contract.ID] = &contract
		}
	}
	
	return nil
}

// IsValidChain valida si una cadena completa es válida
func (bc *Blockchain) IsValidChain(chain []Block) bool {
	if len(chain) == 0 {
		return false
	}
	
	// Verificar cada bloque en la cadena
	for i, block := range chain {
		// Verificar hash del bloque
		if block.Hash == "" {
			return false
		}
		
		// Verificar enlace con bloque anterior (excepto el primero)
		if i > 0 {
			if block.PreviousHash != chain[i-1].Hash {
				return false
			}
		}
	}
	
	return true
}
