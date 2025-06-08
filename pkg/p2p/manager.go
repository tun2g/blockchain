package p2p

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blockchain/pkg/blockchain"
	"github.com/blockchain/pkg/consensus"
	pb "github.com/blockchain/proto"
)

// Manager manages P2P connections and communication
type Manager struct {
	nodeID       string
	peers        map[string]*Client // peer address -> client
	peerAddrs    []string
	peerMutex    sync.RWMutex
	healthTicker *time.Ticker
	stopCh       chan struct{}
}

// NewManager creates a new P2P manager
func NewManager(nodeID string, peerAddrs []string) *Manager {
	return &Manager{
		nodeID:    nodeID,
		peers:     make(map[string]*Client),
		peerAddrs: peerAddrs,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the P2P manager
func (m *Manager) Start() error {
	log.Printf("[%s] Starting P2P manager with %d peers", m.nodeID, len(m.peerAddrs))

	// Connect to all peers
	for _, addr := range m.peerAddrs {
		if err := m.connectToPeer(addr); err != nil {
			log.Printf("[%s] Failed to connect to peer %s: %v", m.nodeID, addr, err)
			// Continue with other peers even if one fails
		}
	}

	// Start health check routine
	m.healthTicker = time.NewTicker(30 * time.Second)
	go m.healthCheckRoutine()

	return nil
}

// Stop stops the P2P manager
func (m *Manager) Stop() {
	log.Printf("[%s] Stopping P2P manager", m.nodeID)

	if m.healthTicker != nil {
		m.healthTicker.Stop()
	}

	close(m.stopCh)

	// Close all peer connections
	m.peerMutex.Lock()
	defer m.peerMutex.Unlock()

	for addr, client := range m.peers {
		if err := client.Close(); err != nil {
			log.Printf("[%s] Error closing connection to %s: %v", m.nodeID, addr, err)
		}
	}

	m.peers = make(map[string]*Client)
}

// connectToPeer establishes a connection to a peer
func (m *Manager) connectToPeer(addr string) error {
	m.peerMutex.Lock()
	defer m.peerMutex.Unlock()

	// Check if already connected
	if _, exists := m.peers[addr]; exists {
		return nil
	}

	// Create new client connection
	client, err := NewClient(addr)
	if err != nil {
		return fmt.Errorf("failed to create client for %s: %w", addr, err)
	}

	// Test the connection
	if !client.IsHealthy() {
		client.Close()
		return fmt.Errorf("peer %s is not healthy", addr)
	}

	m.peers[addr] = client
	log.Printf("[%s] Connected to peer %s", m.nodeID, addr)

	return nil
}

// healthCheckRoutine periodically checks peer health and reconnects if needed
func (m *Manager) healthCheckRoutine() {
	for {
		select {
		case <-m.healthTicker.C:
			m.checkPeerHealth()
		case <-m.stopCh:
			return
		}
	}
}

// checkPeerHealth checks the health of all peers and reconnects if needed
func (m *Manager) checkPeerHealth() {
	m.peerMutex.Lock()
	defer m.peerMutex.Unlock()

	for addr, client := range m.peers {
		if !client.IsHealthy() {
			log.Printf("[%s] Peer %s is unhealthy, attempting reconnection", m.nodeID, addr)
			client.Close()
			delete(m.peers, addr)

			// Try to reconnect
			go func(peerAddr string) {
				time.Sleep(5 * time.Second) // Wait before reconnecting
				if err := m.connectToPeer(peerAddr); err != nil {
					log.Printf("[%s] Failed to reconnect to %s: %v", m.nodeID, peerAddr, err)
				}
			}(addr)
		}
	}
}

// BroadcastTransaction broadcasts a transaction to all connected peers
func (m *Manager) BroadcastTransaction(tx *blockchain.Transaction) error {
	m.peerMutex.RLock()
	defer m.peerMutex.RUnlock()

	if len(m.peers) == 0 {
		return fmt.Errorf("no connected peers")
	}

	// Convert to protobuf format
	protoTx := &pb.Transaction{
		Sender:    tx.Sender,
		Receiver:  tx.Receiver,
		Amount:    tx.Amount,
		Timestamp: tx.Timestamp,
		Signature: tx.Signature,
	}

	var wg sync.WaitGroup
	var errorCount int
	var errorMutex sync.Mutex

	// Broadcast to all peers concurrently
	for addr, client := range m.peers {
		wg.Add(1)
		go func(peerAddr string, peerClient *Client) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := peerClient.SendTransaction(ctx, protoTx); err != nil {
				log.Printf("[%s] Failed to send transaction to %s: %v", m.nodeID, peerAddr, err)
				errorMutex.Lock()
				errorCount++
				errorMutex.Unlock()
			}
		}(addr, client)
	}

	wg.Wait()

	if errorCount == len(m.peers) {
		return fmt.Errorf("failed to send transaction to all peers")
	}

	log.Printf("[%s] Transaction broadcast to %d peers (%d errors)", m.nodeID, len(m.peers), errorCount)
	return nil
}

// BroadcastBlock broadcasts a block proposal to all connected peers
func (m *Manager) BroadcastBlock(block *blockchain.Block) error {
	m.peerMutex.RLock()
	defer m.peerMutex.RUnlock()

	if len(m.peers) == 0 {
		return fmt.Errorf("no connected peers")
	}

	// Convert to protobuf format
	protoBlock := internalToProtoBlockMgr(block)

	var wg sync.WaitGroup
	var errorCount int
	var errorMutex sync.Mutex

	// Broadcast to all peers concurrently
	for addr, client := range m.peers {
		wg.Add(1)
		go func(peerAddr string, peerClient *Client) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := peerClient.ProposeBlock(ctx, protoBlock, m.nodeID); err != nil {
				log.Printf("[%s] Failed to propose block to %s: %v", m.nodeID, peerAddr, err)
				errorMutex.Lock()
				errorCount++
				errorMutex.Unlock()
			}
		}(addr, client)
	}

	wg.Wait()

	if errorCount == len(m.peers) {
		return fmt.Errorf("failed to propose block to all peers")
	}

	log.Printf("[%s] Block proposed to %d peers (%d errors)", m.nodeID, len(m.peers), errorCount)
	return nil
}

// BroadcastVote broadcasts a vote to all connected peers
func (m *Manager) BroadcastVote(vote *consensus.Vote) error {
	m.peerMutex.RLock()
	defer m.peerMutex.RUnlock()

	if len(m.peers) == 0 {
		return fmt.Errorf("no connected peers")
	}

	// Convert to protobuf format
	protoVote := &pb.Vote{
		NodeId:    vote.NodeID,
		BlockHash: vote.BlockHash,
		Approve:   vote.Approve,
		Timestamp: vote.Timestamp,
	}

	var wg sync.WaitGroup
	var errorCount int
	var errorMutex sync.Mutex

	// Broadcast to all peers concurrently
	for addr, client := range m.peers {
		wg.Add(1)
		go func(peerAddr string, peerClient *Client) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := peerClient.SendVote(ctx, protoVote); err != nil {
				log.Printf("[%s] Failed to send vote to %s: %v", m.nodeID, peerAddr, err)
				errorMutex.Lock()
				errorCount++
				errorMutex.Unlock()
			}
		}(addr, client)
	}

	wg.Wait()

	if errorCount == len(m.peers) {
		return fmt.Errorf("failed to send vote to all peers")
	}

	log.Printf("[%s] Vote broadcast to %d peers (%d errors)", m.nodeID, len(m.peers), errorCount)
	return nil
}

// SyncWithPeers synchronizes blockchain data with peers
func (m *Manager) SyncWithPeers(currentHeight int) ([]*blockchain.Block, error) {
	m.peerMutex.RLock()
	defer m.peerMutex.RUnlock()

	if len(m.peers) == 0 {
		return nil, fmt.Errorf("no connected peers for synchronization")
	}

	// Try to sync with each peer until successful
	for addr, client := range m.peers {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		blocks, err := client.Sync(ctx, m.nodeID, int32(currentHeight))
		cancel()

		if err != nil {
			log.Printf("[%s] Failed to sync with %s: %v", m.nodeID, addr, err)
			continue
		}

		// Convert protobuf blocks to internal format
		var internalBlocks []*blockchain.Block
		for _, protoBlock := range blocks {
			block := protoToInternalBlockMgr(protoBlock)
			internalBlocks = append(internalBlocks, block)
		}

		log.Printf("[%s] Synchronized %d blocks from %s", m.nodeID, len(internalBlocks), addr)
		return internalBlocks, nil
	}

	return nil, fmt.Errorf("failed to synchronize with any peer")
}

// GetPeerCount returns the number of connected peers
func (m *Manager) GetPeerCount() int {
	m.peerMutex.RLock()
	defer m.peerMutex.RUnlock()
	return len(m.peers)
}

// GetConnectedPeers returns the addresses of connected peers
func (m *Manager) GetConnectedPeers() []string {
	m.peerMutex.RLock()
	defer m.peerMutex.RUnlock()

	var peers []string
	for addr := range m.peers {
		peers = append(peers, addr)
	}
	return peers
}

// Helper functions for format conversion

func internalToProtoBlockMgr(block *blockchain.Block) *pb.Block {
	var protoTransactions []*pb.Transaction
	for _, tx := range block.Transactions {
		protoTx := &pb.Transaction{
			Sender:    tx.Sender,
			Receiver:  tx.Receiver,
			Amount:    tx.Amount,
			Timestamp: tx.Timestamp,
			Signature: tx.Signature,
		}
		protoTransactions = append(protoTransactions, protoTx)
	}

	return &pb.Block{
		Index:            int32(block.Index),
		Timestamp:        block.Timestamp,
		Transactions:     protoTransactions,
		MerkleRoot:       block.MerkleRoot,
		PreviousHash:     block.PreviousHash,
		CurrentBlockHash: block.CurrentBlockHash,
		Nonce:            int32(block.Nonce),
	}
}

func protoToInternalBlockMgr(protoBlock *pb.Block) *blockchain.Block {
	var transactions []*blockchain.Transaction
	for _, protoTx := range protoBlock.Transactions {
		tx := &blockchain.Transaction{
			Sender:    protoTx.Sender,
			Receiver:  protoTx.Receiver,
			Amount:    protoTx.Amount,
			Timestamp: protoTx.Timestamp,
			Signature: protoTx.Signature,
		}
		transactions = append(transactions, tx)
	}

	return &blockchain.Block{
		Index:            int(protoBlock.Index),
		Timestamp:        protoBlock.Timestamp,
		Transactions:     transactions,
		MerkleRoot:       protoBlock.MerkleRoot,
		PreviousHash:     protoBlock.PreviousHash,
		CurrentBlockHash: protoBlock.CurrentBlockHash,
		Nonce:            int(protoBlock.Nonce),
	}
}
