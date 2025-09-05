package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"secop-blockchain/internal/blockchain"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var bc *blockchain.Blockchain
var p2pNetwork *blockchain.P2PNetwork
var workflowManager *blockchain.WorkflowManager

func main() {
	// Obtener configuración del nodo desde variables de entorno
	nodeID := getEnv("NODE_ID", "DNP-NODE")
	nodeAddress := getEnv("NODE_ADDRESS", "localhost")
	nodePort := getEnv("NODE_PORT", "8080")
	
	fmt.Printf("🚀 Iniciando nodo %s en %s:%s\n", nodeID, nodeAddress, nodePort)

	// Inicializar blockchain
	bc = blockchain.NewBlockchain()
	
	// Inicializar red P2P
	p2pNetwork = blockchain.NewP2PNetwork(nodeID, nodeAddress, nodePort, bc)
	
	// Inicializar workflow manager
	workflowManager = blockchain.NewWorkflowManager(bc)
	
	// Configurar peers iniciales desde variables de entorno (OPCIONAL)
	setupInitialPeers()

	// Configurar Gin
	r := gin.Default()

	// Configurar CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
	}))

	// *** BACKEND SOLO - Sin frontend ***
	// r.Static("/static", "./web/public")
	// r.StaticFile("/", "./web/public/index.html")

	// API Routes existentes
	r.GET("/api/blocks", getBlocks)
	r.GET("/api/contracts", getContracts)
	r.POST("/api/contracts", createContract)
	r.POST("/api/contracts/validate", validateContract)
	r.GET("/api/stats", getStats)

	// Nuevas rutas de flujo de trabajo SECOP
	r.GET("/api/workflow/steps", getWorkflowSteps)
	r.GET("/api/contracts/:id/workflow", getContractWorkflowStatus)
	r.POST("/api/contracts/:id/validate-step", validateContractStep)
	r.POST("/api/contracts/:id/audit", addAuditObservation)
	r.GET("/api/contracts/by-status/:status", getContractsByStatus)
	r.GET("/api/contracts/by-role/:role", getContractsByRole)

	// Nuevas rutas P2P
	r.GET("/api/health", healthCheck)
	r.GET("/api/p2p/peers", getPeers)
	r.POST("/api/p2p/add-peer", addPeer)
	r.GET("/api/p2p/get-chain", getChain)
	r.POST("/api/p2p/receive-block", receiveBlock)
	r.POST("/api/p2p/sync", syncWithPeers)

	// Iniciar sincronización periódica
	go startPeriodicSync()
	
	// Iniciar health check periódico
	go startPeriodicHealthCheck()

	// Crear contratos de ejemplo solo en el nodo DNP
	if nodeID == "DNP-NODE" {
		createExampleContracts()
	}

	fmt.Printf("🌐 Servidor backend iniciado en puerto %s\n", nodePort)
	fmt.Printf("🔗 API disponible en http://%s:%s/api/\n", nodeAddress, nodePort)
	
	r.Run(":" + nodePort)
}

// setupInitialPeers configura los peers iniciales desde variables de entorno (OPCIONAL)
func setupInitialPeers() {
	peers := getEnv("INITIAL_PEERS", "")
	if peers == "" {
		fmt.Printf("🌐 Modo descubrimiento dinámico - sin peers iniciales configurados\n")
		fmt.Printf("💡 Los nodos se conectarán automáticamente usando /api/p2p/add-peer\n")
		return
	}

	fmt.Printf("🔗 Configurando peers iniciales: %s\n", peers)
	
	// Parsear peers en formato: "NODE1:localhost:8081,NODE2:localhost:8082"
	peerList := strings.Split(peers, ",")
	for _, peerInfo := range peerList {
		parts := strings.Split(strings.TrimSpace(peerInfo), ":")
		if len(parts) == 3 {
			nodeID := parts[0]
			address := parts[1]
			port := parts[2]
			
			// Agregar peer a la red
			p2pNetwork.AddPeer(nodeID, address, port)
			fmt.Printf("✅ Peer agregado: %s (%s:%s)\n", nodeID, address, port)
		}
	}
}

// Nuevos handlers P2P

func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"node_id":   p2pNetwork.NodeID,
		"timestamp": time.Now(),
		"blocks":    len(bc.Chain),
		"contracts": len(bc.Contracts),
	})
}

func getPeers(c *gin.Context) {
	peers := p2pNetwork.GetActivePeers()
	c.JSON(http.StatusOK, gin.H{
		"peers": peers,
		"count": len(peers),
	})
}

func addPeer(c *gin.Context) {
	var req struct {
		PeerID  string `json:"peer_id"`
		Address string `json:"address"`
		Port    string `json:"port"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p2pNetwork.AddPeer(req.PeerID, req.Address, req.Port)
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("Peer %s agregado exitosamente", req.PeerID),
	})
}

func getChain(c *gin.Context) {
	// Convertir Chain de []*Block a []Block para JSON
	var blocks []blockchain.Block
	for _, block := range bc.Chain {
		blocks = append(blocks, *block)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"chain":  blocks,
		"length": len(blocks),
		"node_id": p2pNetwork.NodeID,
	})
}

func receiveBlock(c *gin.Context) {
	var block blockchain.Block
	if err := c.ShouldBindJSON(&block); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := p2pNetwork.ReceiveBlock(block)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bloque recibido y procesado exitosamente",
	})
}

func syncWithPeers(c *gin.Context) {
	err := p2pNetwork.SyncWithPeers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Sincronización completada",
		"blocks":  len(bc.Chain),
	})
}

// Funciones de sincronización periódica

func startPeriodicSync() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("🔄 Sincronización periódica iniciada\n")
		p2pNetwork.SyncWithPeers()
	}
}

func startPeriodicHealthCheck() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("💚 Health check periódico iniciado\n")
		p2pNetwork.HealthCheck()
	}
}

// Handlers existentes modificados para P2P

func getBlocks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"blocks_count":    len(bc.Chain),
			"contracts_count": len(bc.Contracts),
			"is_valid":        bc.IsChainValid(),
			"latest_block":    bc.Chain[len(bc.Chain)-1],
		},
	})
}

func getContracts(c *gin.Context) {
	contracts := bc.GetAllContracts()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"count":   len(contracts),
		"data":    contracts,
	})
}

func createContract(c *gin.Context) {
	var contract blockchain.Contract
	if err := c.ShouldBindJSON(&contract); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := bc.AddContract(&contract)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast del nuevo bloque a peers
	if len(bc.Chain) > 0 {
		lastBlock := *bc.Chain[len(bc.Chain)-1]
		fmt.Printf("📡 Broadcasting nuevo contrato a peers\n")
		go p2pNetwork.BroadcastBlock(lastBlock)
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Contrato creado exitosamente",
		"contract_id": contract.ID,
	})
}

func validateContract(c *gin.Context) {
	var req struct {
		ContractID string `json:"contractId"`
		NodeID     string `json:"nodeId"`
		Approved   bool   `json:"approved"`
		Reason     string `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := bc.ValidateContract(req.ContractID, req.NodeID, req.Approved, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast del bloque de validación a peers
	if len(bc.Chain) > 0 {
		lastBlock := *bc.Chain[len(bc.Chain)-1]
		fmt.Printf("📡 Broadcasting validación a peers\n")
		go p2pNetwork.BroadcastBlock(lastBlock)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Validación registrada exitosamente",
	})
}

func getStats(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"blocks_count":    len(bc.Chain),
			"contracts_count": len(bc.Contracts),
			"is_valid":        bc.IsChainValid(),
			"latest_block":    bc.Chain[len(bc.Chain)-1],
		},
	})
}

// Handlers de flujo de trabajo SECOP
func getWorkflowSteps(c *gin.Context) {
	steps := workflowManager.GetWorkflowSteps()
	c.JSON(200, gin.H{"steps": steps})
}

func getContractWorkflowStatus(c *gin.Context) {
	contractID := c.Param("id")
	status, err := workflowManager.GetWorkflowStatus(contractID)
	if err != nil {
		c.JSON(404, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, status)
}

func validateContractStep(c *gin.Context) {
	contractID := c.Param("id")
	
	var req struct {
		StepNumber    int    `json:"step_number"`
		ValidatorID   string `json:"validator_id"`
		ValidatorName string `json:"validator_name"`
		Role          string `json:"role"`
		Approved      bool   `json:"approved"`
		Comments      string `json:"comments"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	role := blockchain.AdminRole(req.Role)
	err := workflowManager.ValidateStep(contractID, req.StepNumber, req.ValidatorID, req.ValidatorName, role, req.Approved, req.Comments)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"message": "Paso validado exitosamente"})
}

func addAuditObservation(c *gin.Context) {
	contractID := c.Param("id")
	
	var req struct {
		AuditorID   string `json:"auditor_id"`
		Role        string `json:"role"`
		Observation string `json:"observation"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	role := blockchain.AdminRole(req.Role)
	err := workflowManager.AddAuditObservation(contractID, req.AuditorID, role, req.Observation)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(200, gin.H{"message": "Observación de auditoría agregada"})
}

func getContractsByStatus(c *gin.Context) {
	status := c.Param("status")
	contracts := bc.GetContractsByStatus(blockchain.ContractStatus(status))
	c.JSON(200, gin.H{"contracts": contracts})
}

func getContractsByRole(c *gin.Context) {
	role := c.Param("role")
	contracts := bc.GetContractsByRole(blockchain.AdminRole(role))
	c.JSON(200, gin.H{"contracts": contracts})
}

// Función auxiliar para obtener variables de entorno
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func createExampleContracts() {
	// Contrato 1: Construcción de puente
	contract1 := blockchain.Contract{
		EntityCode:   "08001",
		EntityName:   "Alcaldía de Medellín",
		ContractType: "OBRA_PUBLICA",
		Description:  "Construcción de puente peatonal en la Comuna 1",
		Amount:       2500000000, // $2.500 millones
		CreatedBy:    "funcionario.obras@medellin.gov.co",
	}

	// Contrato 2: Suministro de computadores
	contract2 := blockchain.Contract{
		EntityCode:   "11001",
		EntityName:   "Secretaría de Educación de Bogotá",
		ContractType: "SUMINISTRO",
		Description:  "Adquisición de 500 computadores para colegios públicos",
		Amount:       800000000, // $800 millones
		CreatedBy:    "compras.educacion@educacionbogota.edu.co",
	}

	bc.AddContract(&contract1)
	bc.AddContract(&contract2)

	fmt.Printf("📝 Contratos de ejemplo creados:\n")
	fmt.Printf("   - Puente peatonal Medellín\n")
	fmt.Printf("   - Computadores Bogotá\n")
}
