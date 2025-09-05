package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"secop-blockchain/internal/blockchain"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var bc *blockchain.Blockchain
var p2pNetwork *blockchain.P2PNetwork

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
	
	// Configurar peers iniciales desde variables de entorno
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

// setupInitialPeers configura los peers iniciales desde variables de entorno
func setupInitialPeers() {
	peers := getEnv("INITIAL_PEERS", "")
	if peers == "" {
		return
	}

	// Formato esperado: "NODE1:localhost:8081,NODE2:localhost:8082"
	// Parsear y agregar peers
	fmt.Printf("🔗 Configurando peers iniciales: %s\n", peers)
	
	// Aquí puedes implementar el parsing de peers si lo necesitas
	// Por ahora, configuraremos manualmente según el nodo
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
