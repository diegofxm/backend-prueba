package blockchain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// Peer representa un nodo peer en la red
type Peer struct {
	ID       string `json:"id"`
	Address  string `json:"address"`
	Port     string `json:"port"`
	LastSeen time.Time `json:"last_seen"`
	Active   bool   `json:"active"`
}

// P2PNetwork maneja la comunicación entre nodos
type P2PNetwork struct {
	NodeID     string
	Address    string
	Port       string
	Peers      map[string]*Peer
	Blockchain *Blockchain
	mutex      sync.RWMutex
}

// NewP2PNetwork crea una nueva instancia de red P2P
func NewP2PNetwork(nodeID, address, port string, blockchain *Blockchain) *P2PNetwork {
	return &P2PNetwork{
		NodeID:     nodeID,
		Address:    address,
		Port:       port,
		Peers:      make(map[string]*Peer),
		Blockchain: blockchain,
	}
}

// AddPeer agrega un nuevo peer a la red
func (p2p *P2PNetwork) AddPeer(peerID, address, port string) {
	p2p.mutex.Lock()
	defer p2p.mutex.Unlock()
	
	p2p.Peers[peerID] = &Peer{
		ID:       peerID,
		Address:  address,
		Port:     port,
		LastSeen: time.Now(),
		Active:   true,
	}
	
	fmt.Printf("🔗 Peer agregado: %s (%s:%s)\n", peerID, address, port)
}

// BroadcastBlock envía un nuevo bloque a todos los peers
func (p2p *P2PNetwork) BroadcastBlock(block Block) {
	p2p.mutex.RLock()
	defer p2p.mutex.RUnlock()
	
	fmt.Printf("📡 Broadcasting bloque %s a %d peers\n", block.Hash, len(p2p.Peers))
	
	for peerID, peer := range p2p.Peers {
		if !peer.Active {
			continue
		}
		
		go func(peerID string, peer *Peer) {
			err := p2p.sendBlockToPeer(peer, block)
			if err != nil {
				fmt.Printf("❌ Error enviando bloque a %s: %v\n", peerID, err)
				p2p.markPeerInactive(peerID)
			} else {
				fmt.Printf("✅ Bloque enviado a %s\n", peerID)
			}
		}(peerID, peer)
	}
}

// sendBlockToPeer envía un bloque a un peer específico
func (p2p *P2PNetwork) sendBlockToPeer(peer *Peer, block Block) error {
	url := fmt.Sprintf("http://%s:%s/api/p2p/receive-block", peer.Address, peer.Port)
	
	blockData, err := json.Marshal(block)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(blockData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer respondió con status %d", resp.StatusCode)
	}
	
	return nil
}

// ReceiveBlock procesa un bloque recibido de otro peer
func (p2p *P2PNetwork) ReceiveBlock(block Block) error {
	fmt.Printf("📥 Bloque recibido de peer: %s\n", block.Hash)
	
	// Validar el bloque
	if !p2p.Blockchain.IsValidBlock(block) {
		return fmt.Errorf("bloque inválido recibido")
	}
	
	// Verificar si ya tenemos este bloque
	if p2p.Blockchain.HasBlock(block.Hash) {
		fmt.Printf("⚠️ Bloque %s ya existe, ignorando\n", block.Hash)
		return nil
	}
	
	// Agregar el bloque a nuestra cadena
	blockData := map[string]interface{}{
		"type":          block.Type,
		"data":          block.Data,
		"timestamp":     block.Timestamp,
		"previous_hash": block.PreviousHash,
		"nonce":         block.Nonce,
	}
	
	err := p2p.Blockchain.AddBlock(blockData)
	if err != nil {
		return fmt.Errorf("error agregando bloque: %v", err)
	}
	
	fmt.Printf("✅ Bloque %s agregado exitosamente\n", block.Hash)
	return nil
}

// SyncWithPeers sincroniza la blockchain con todos los peers
func (p2p *P2PNetwork) SyncWithPeers() error {
	p2p.mutex.RLock()
	defer p2p.mutex.RUnlock()
	
	fmt.Printf("🔄 Iniciando sincronización con %d peers\n", len(p2p.Peers))
	
	for peerID, peer := range p2p.Peers {
		if !peer.Active {
			continue
		}
		
		chain, err := p2p.requestChainFromPeer(peer)
		if err != nil {
			fmt.Printf("❌ Error obteniendo cadena de %s: %v\n", peerID, err)
			continue
		}
		
		// Si el peer tiene una cadena más larga y válida, la adoptamos
		if len(chain) > len(p2p.Blockchain.Chain) && p2p.Blockchain.IsValidChain(chain) {
			fmt.Printf("🔄 Adoptando cadena más larga de %s (%d bloques)\n", peerID, len(chain))
			// Convertir []Block a []*Block
			p2p.Blockchain.Chain = make([]*Block, len(chain))
			for i, block := range chain {
				blockCopy := block
				p2p.Blockchain.Chain[i] = &blockCopy
			}
			p2p.rebuildContractsFromChain()
		}
	}
	
	return nil
}

// requestChainFromPeer solicita la blockchain completa de un peer
func (p2p *P2PNetwork) requestChainFromPeer(peer *Peer) ([]Block, error) {
	url := fmt.Sprintf("http://%s:%s/api/p2p/get-chain", peer.Address, peer.Port)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer respondió con status %d", resp.StatusCode)
	}
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var response struct {
		Chain []Block `json:"chain"`
	}
	
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	
	return response.Chain, nil
}

// rebuildContractsFromChain reconstruye el mapa de contratos desde la cadena
func (p2p *P2PNetwork) rebuildContractsFromChain() {
	p2p.Blockchain.Contracts = make(map[string]*Contract)
	
	for _, block := range p2p.Blockchain.Chain {
		if block.Type == "CONTRACT_CREATION" {
			var contract Contract
			err := json.Unmarshal([]byte(fmt.Sprintf("%v", block.Data)), &contract)
			if err == nil {
				p2p.Blockchain.Contracts[contract.ID] = &contract
			}
		}
	}
	
	fmt.Printf("🔄 Contratos reconstruidos: %d\n", len(p2p.Blockchain.Contracts))
}

// markPeerInactive marca un peer como inactivo
func (p2p *P2PNetwork) markPeerInactive(peerID string) {
	p2p.mutex.Lock()
	defer p2p.mutex.Unlock()
	
	if peer, exists := p2p.Peers[peerID]; exists {
		peer.Active = false
		fmt.Printf("⚠️ Peer %s marcado como inactivo\n", peerID)
	}
}

// GetActivePeers retorna la lista de peers activos
func (p2p *P2PNetwork) GetActivePeers() []*Peer {
	p2p.mutex.RLock()
	defer p2p.mutex.RUnlock()
	
	var activePeers []*Peer
	for _, peer := range p2p.Peers {
		if peer.Active {
			activePeers = append(activePeers, peer)
		}
	}
	
	return activePeers
}

// HealthCheck verifica el estado de todos los peers
func (p2p *P2PNetwork) HealthCheck() {
	p2p.mutex.Lock()
	defer p2p.mutex.Unlock()
	
	for peerID, peer := range p2p.Peers {
		url := fmt.Sprintf("http://%s:%s/api/health", peer.Address, peer.Port)
		
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(url)
		
		if err != nil || resp.StatusCode != http.StatusOK {
			peer.Active = false
			fmt.Printf("💔 Peer %s no responde\n", peerID)
		} else {
			peer.Active = true
			peer.LastSeen = time.Now()
			fmt.Printf("💚 Peer %s activo\n", peerID)
		}
		
		if resp != nil {
			resp.Body.Close()
		}
	}
}
