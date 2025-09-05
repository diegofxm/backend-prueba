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

// Contract representa un contrato estatal con flujo completo de validación
type Contract struct {
	ID              string             `json:"id"`
	EntityCode      string             `json:"entity_code"`
	EntityName      string             `json:"entity_name"`
	ContractType    string             `json:"contract_type"`
	Description     string             `json:"description"`
	Amount          float64            `json:"amount"`
	Status          ContractStatus     `json:"status"`
	CreatedBy       string             `json:"created_by"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	ValidationSteps []ValidationStep   `json:"validation_steps"`
	CurrentStep     int                `json:"current_step"`
	RequiredRoles   []string           `json:"required_roles"`
	AuditTrail      []AuditEntry       `json:"audit_trail"`
}

// ContractStatus define los estados del contrato en el flujo SECOP
type ContractStatus string

const (
	StatusDraft                   ContractStatus = "DRAFT"
	StatusTechnicalReview         ContractStatus = "TECHNICAL_REVIEW"
	StatusTechnicalApproved       ContractStatus = "TECHNICAL_APPROVED"
	StatusLegalReview             ContractStatus = "LEGAL_REVIEW"
	StatusLegalApproved           ContractStatus = "LEGAL_APPROVED"
	StatusContractsReview         ContractStatus = "CONTRACTS_REVIEW"
	StatusContractsApproved       ContractStatus = "CONTRACTS_APPROVED"
	StatusAdminReview             ContractStatus = "ADMIN_REVIEW"
	StatusAdminApproved           ContractStatus = "ADMIN_APPROVED"
	StatusBudgetReview            ContractStatus = "BUDGET_REVIEW"
	StatusAuthorizedForPublication ContractStatus = "AUTHORIZED_FOR_PUBLICATION"
	StatusPublished               ContractStatus = "PUBLISHED"
	StatusProposalsReceived       ContractStatus = "PROPOSALS_RECEIVED"
	StatusEvaluated               ContractStatus = "EVALUATED"
	StatusAwarded                 ContractStatus = "AWARDED"
	StatusExecuted                ContractStatus = "EXECUTED"
	StatusCompleted               ContractStatus = "COMPLETED"
	// Estados de control (no bloquean el proceso)
	StatusUnderAudit              ContractStatus = "UNDER_AUDIT"
	StatusAuditObservations       ContractStatus = "AUDIT_OBSERVATIONS"
	StatusRejected                ContractStatus = "REJECTED"
)

// ValidationStep representa un paso de validación en el flujo
type ValidationStep struct {
	StepNumber    int                    `json:"step_number"`
	Role          AdminRole              `json:"role"`
	ValidatorID   string                 `json:"validator_id"`
	ValidatorName string                 `json:"validator_name"`
	Status        ValidationStatus       `json:"status"`
	Timestamp     time.Time              `json:"timestamp"`
	Comments      string                 `json:"comments"`
	Required      bool                   `json:"required"`
	DigitalSign   string                 `json:"digital_sign"`
	Documents     []string               `json:"documents"`
}

// AdminRole define los roles administrativos internos
type AdminRole string

const (
	RoleProjectDeveloper  AdminRole = "PROJECT_DEVELOPER"
	RoleTechnicalCommission AdminRole = "TECHNICAL_COMMISSION"
	RoleLegalCommission   AdminRole = "LEGAL_COMMISSION"
	RoleContractsChief    AdminRole = "CONTRACTS_CHIEF"
	RoleAdminChief        AdminRole = "ADMIN_CHIEF"
	RoleBudgetAuthority   AdminRole = "BUDGET_AUTHORITY"
	// Roles de control externo (solo auditoría)
	RoleComptroller       AdminRole = "COMPTROLLER"
	RoleProsecutor        AdminRole = "PROSECUTOR"
	RoleCitizen           AdminRole = "CITIZEN"
)

// ValidationStatus define el estado de una validación
type ValidationStatus string

const (
	ValidationPending   ValidationStatus = "PENDING"
	ValidationApproved  ValidationStatus = "APPROVED"
	ValidationRejected  ValidationStatus = "REJECTED"
	ValidationInReview  ValidationStatus = "IN_REVIEW"
)

// AuditEntry representa una entrada de auditoría
type AuditEntry struct {
	ID          string    `json:"id"`
	Action      string    `json:"action"`
	UserID      string    `json:"user_id"`
	UserRole    AdminRole `json:"user_role"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	IPAddress   string    `json:"ip_address"`
	BlockHash   string    `json:"block_hash"`
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
