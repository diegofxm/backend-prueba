package blockchain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Blockchain representa la cadena de bloques SECOP
type Blockchain struct {
	Chain           []*Block             `json:"chain"`
	Contracts       map[string]*Contract `json:"contracts"`
	WorkflowManager *WorkflowManager     `json:"-"`
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

	bc := &Blockchain{
		Chain:     []*Block{genesisBlock},
		Contracts: make(map[string]*Contract),
	}
	
	// Inicializar el gestor de flujo de trabajo
	bc.WorkflowManager = NewWorkflowManager(bc)
	
	return bc
}

// AddContract agrega un nuevo contrato a la blockchain con flujo de trabajo
func (bc *Blockchain) AddContract(contract *Contract) error {
	// Validar contrato
	if err := bc.validateContract(contract); err != nil {
		return err
	}

	// Generar ID único si no existe
	if contract.ID == "" {
		contract.ID = uuid.New().String()
	}

	// Establecer timestamp y estado inicial
	contract.CreatedAt = time.Now()
	contract.UpdatedAt = time.Now()
	contract.Status = StatusDraft

	// Inicializar flujo de trabajo
	if err := bc.WorkflowManager.InitializeContractWorkflow(contract); err != nil {
		return fmt.Errorf("error inicializando flujo de trabajo: %v", err)
	}

	// Agregar a la blockchain
	bc.Contracts[contract.ID] = contract

	// Crear bloque para el contrato
	blockData := map[string]interface{}{
		"type":        "CONTRACT_CREATION",
		"contract_id": contract.ID,
		"entity_code": contract.EntityCode,
		"entity_name": contract.EntityName,
		"amount":      contract.Amount,
		"created_by":  contract.CreatedBy,
		"timestamp":   contract.CreatedAt,
	}

	return bc.AddBlock(blockData)
}

// ValidateContractStep valida un paso del flujo de trabajo
func (bc *Blockchain) ValidateContractStep(contractID string, stepNumber int, validatorID string, validatorName string, role AdminRole, approved bool, comments string) error {
	return bc.WorkflowManager.ValidateStep(contractID, stepNumber, validatorID, validatorName, role, approved, comments)
}

// AddAuditObservation agrega una observación de auditoría
func (bc *Blockchain) AddAuditObservation(contractID string, auditorID string, role AdminRole, observation string) error {
	return bc.WorkflowManager.AddAuditObservation(contractID, auditorID, role, observation)
}

// GetContractWorkflowStatus obtiene el estado del flujo de trabajo de un contrato
func (bc *Blockchain) GetContractWorkflowStatus(contractID string) (*WorkflowStatus, error) {
	return bc.WorkflowManager.GetContractWorkflowStatus(contractID)
}

// GetContractsByStatus obtiene contratos por estado
func (bc *Blockchain) GetContractsByStatus(status ContractStatus) []*Contract {
	var contracts []*Contract
	for _, contract := range bc.Contracts {
		if contract.Status == status {
			contracts = append(contracts, contract)
		}
	}
	return contracts
}

// GetContractsByRole obtiene contratos que requieren validación de un rol específico
func (bc *Blockchain) GetContractsByRole(role AdminRole) []*Contract {
	var contracts []*Contract
	for _, contract := range bc.Contracts {
		if contract.CurrentStep <= len(contract.ValidationSteps) {
			currentStepRole := contract.ValidationSteps[contract.CurrentStep-1].Role
			if currentStepRole == role && contract.ValidationSteps[contract.CurrentStep-1].Status == ValidationPending {
				contracts = append(contracts, contract)
			}
		}
	}
	return contracts
}

// ValidateContract valida un contrato por parte de un nodo
func (bc *Blockchain) ValidateContract(contractID string, nodeID string, approved bool, reason string) error {
	contract, exists := bc.Contracts[contractID]
	if !exists {
		return errors.New("contrato no encontrado")
	}

	// Crear bloque de validación
	validationData := map[string]interface{}{
		"type":        "VALIDATION",
		"contract_id": contractID,
		"node_id":     nodeID,
		"approved":    approved,
		"reason":      reason,
		"timestamp":   time.Now(),
	}

	// Actualizar estado del contrato basado en el flujo de trabajo
	if approved {
		// El estado se maneja ahora a través del WorkflowManager
		fmt.Printf("✅ Validación aprobada para contrato %s por nodo %s\n", contractID, nodeID)
	} else {
		contract.Status = StatusRejected
		fmt.Printf("❌ Validación rechazada para contrato %s por nodo %s: %s\n", contractID, nodeID, reason)
	}

	return bc.AddBlock(validationData)
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

// AddBlock agrega un nuevo bloque a la cadena con datos
func (bc *Blockchain) AddBlock(blockData map[string]interface{}) error {
	// Crear el bloque con los datos proporcionados
	block := NewBlock(blockData, bc.getLatestBlock().Hash)
	block.Index = len(bc.Chain)
	
	// Establecer tipo de bloque si está especificado
	if blockType, ok := blockData["type"].(string); ok {
		block.Type = blockType
	}
	
	// Recalcular hash con el índice correcto
	block.Hash = block.calculateHash()

	// Verificar que el bloque sea válido
	if !bc.IsValidBlock(*block) {
		return errors.New("bloque inválido")
	}

	// Agregar a la cadena
	bc.Chain = append(bc.Chain, block)
	fmt.Printf("✅ Bloque %d agregado a la cadena\n", block.Index)
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
