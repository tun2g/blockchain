package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/blockchain/pkg/blockchain"
	"github.com/blockchain/pkg/consensus"
	"github.com/blockchain/pkg/p2p"
	"github.com/blockchain/pkg/storage"
	"github.com/blockchain/pkg/wallet"
	pb "github.com/blockchain/proto"
	"google.golang.org/grpc"
)

// Node represents a blockchain node
type Node struct {
	ID         string
	IsLeader   bool
	Peers      []string
	ListenAddr string
	DataDir    string

	// Core components
	DB              *storage.BlockchainDB
	ConsensusEngine *consensus.ConsensusEngine

	// P2P components
	P2PManager *p2p.Manager
	P2PServer  *p2p.Server

	// gRPC server
	GRPCServer *grpc.Server

	// Node state
	Running        bool
	WalletRegistry map[string]*wallet.Wallet
	WalletMutex    sync.RWMutex

	// Node's own wallet for rewards (optional)
	NodeWallet *wallet.Wallet
}

// NewNode creates a new blockchain node
func NewNode(nodeID string, isLeader bool, peers []string, listenAddr, dataDir string) (*Node, error) {
	// Create data directory
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize database
	db, err := storage.NewBlockchainDB(dataDir + "/blockchain.db")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Create node wallet
	nodeWallet, err := wallet.NewWallet(nodeID + "_node")
	if err != nil {
		return nil, fmt.Errorf("failed to create node wallet: %w", err)
	}

	// Initialize consensus engine
	consensusEngine := consensus.NewConsensusEngine(nodeID, isLeader, peers)

	// Initialize P2P manager
	p2pManager := p2p.NewManager(nodeID, peers)

	node := &Node{
		ID:              nodeID,
		IsLeader:        isLeader,
		Peers:           peers,
		ListenAddr:      listenAddr,
		DataDir:         dataDir,
		DB:              db,
		ConsensusEngine: consensusEngine,
		P2PManager:      p2pManager,
		WalletRegistry:  make(map[string]*wallet.Wallet),
		NodeWallet:      nodeWallet,
	}

	// Initialize P2P server
	node.P2PServer = p2p.NewServer(nodeID, db, consensusEngine)

	// Set P2P callbacks
	node.P2PServer.SetCallbacks(
		node.handleIncomingTransaction,
		node.handleIncomingBlock,
		node.handleIncomingVote,
	)

	// Set consensus callbacks
	consensusEngine.SetCallbacks(
		node.commitBlock,
		node.voteOnBlock,
		node.broadcastBlock,
		node.broadcastVote,
	)

	return node, nil
}

// Start starts the node
func (n *Node) Start() error {
	log.Printf("[%s] Starting node (Leader: %t) on %s", n.ID, n.IsLeader, n.ListenAddr)

	// Initialize blockchain if empty
	if err := n.initializeBlockchain(); err != nil {
		return fmt.Errorf("failed to initialize blockchain: %w", err)
	}

	// Start gRPC server for P2P communication
	lis, err := net.Listen("tcp", n.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	n.GRPCServer = grpc.NewServer()
	pb.RegisterBlockchainServiceServer(n.GRPCServer, n.P2PServer)

	go func() {
		log.Printf("[%s] gRPC server listening on %s", n.ID, n.ListenAddr)
		if err := n.GRPCServer.Serve(lis); err != nil {
			log.Printf("[%s] gRPC server error: %v", n.ID, err)
		}
	}()

	// Start P2P manager
	if err := n.P2PManager.Start(); err != nil {
		return fmt.Errorf("failed to start P2P manager: %w", err)
	}

	// Start consensus if leader
	if n.IsLeader {
		go n.ConsensusEngine.StartBlockProduction(
			n.getPendingTransactions,
			n.getLatestBlock,
		)
	}

	// Start synchronization routine for non-leaders
	if !n.IsLeader {
		go n.syncRoutine()
	}

	n.Running = true
	log.Printf("[%s] Node started successfully", n.ID)

	return nil
}

// Stop stops the node
func (n *Node) Stop() error {
	log.Printf("[%s] Stopping node", n.ID)

	n.Running = false

	if n.P2PManager != nil {
		n.P2PManager.Stop()
	}

	if n.GRPCServer != nil {
		n.GRPCServer.GracefulStop()
	}

	if n.DB != nil {
		n.DB.Close()
	}

	log.Printf("[%s] Node stopped", n.ID)
	return nil
}

// syncRoutine periodically synchronizes with peers
func (n *Node) syncRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !n.Running {
				return
			}
			n.synchronizeWithPeers()
		}
	}
}

// synchronizeWithPeers syncs blockchain data with peers
func (n *Node) synchronizeWithPeers() {
	currentHeight, err := n.DB.GetLatestHeight()
	if err != nil {
		log.Printf("[%s] Failed to get current height: %v", n.ID, err)
		return
	}

	blocks, err := n.P2PManager.SyncWithPeers(currentHeight)
	if err != nil {
		log.Printf("[%s] Failed to sync with peers: %v", n.ID, err)
		return
	}

	for _, block := range blocks {
		if err := n.commitBlock(block); err != nil {
			log.Printf("[%s] Failed to commit synced block %d: %v", n.ID, block.Index, err)
			break
		}
	}

	if len(blocks) > 0 {
		log.Printf("[%s] Successfully synced %d blocks", n.ID, len(blocks))
	}
}

// initializeBlockchain initializes the blockchain with genesis block if empty
func (n *Node) initializeBlockchain() error {
	height, err := n.DB.GetLatestHeight()
	if err != nil {
		return err
	}

	if height < 0 {
		// Create and save genesis block
		genesisBlock := blockchain.NewGenesisBlock()
		if err := n.DB.SaveBlock(genesisBlock); err != nil {
			return fmt.Errorf("failed to save genesis block: %w", err)
		}
		log.Printf("[%s] Genesis block created: %x", n.ID, genesisBlock.Hash())
	}

	return nil
}

// P2P callback handlers

// handleIncomingTransaction handles transactions received from peers
func (n *Node) handleIncomingTransaction(tx *blockchain.Transaction) error {
	log.Printf("[%s] Received transaction from peer", n.ID)
	return n.AddTransaction(tx)
}

// handleIncomingBlock handles block proposals from peers
func (n *Node) handleIncomingBlock(block *blockchain.Block, proposerID string) error {
	log.Printf("[%s] Received block proposal from %s", n.ID, proposerID)
	return n.ConsensusEngine.HandleBlockProposal(block, proposerID)
}

// handleIncomingVote handles votes from peers
func (n *Node) handleIncomingVote(vote *consensus.Vote) error {
	log.Printf("[%s] Received vote from %s", n.ID, vote.NodeID)
	return n.ConsensusEngine.HandleVote(vote)
}

// commitBlock commits a block to the blockchain
func (n *Node) commitBlock(block *blockchain.Block) error {
	log.Printf("[%s] Committing block %d with hash %x", n.ID, block.Index, block.Hash())

	// Validate block before committing
	var previousBlock *blockchain.Block
	if block.Index > 0 {
		var err error
		previousBlock, err = n.DB.GetBlockByIndex(block.Index - 1)
		if err != nil {
			return fmt.Errorf("failed to get previous block: %w", err)
		}
	}

	if !block.IsValid(previousBlock) {
		return fmt.Errorf("invalid block")
	}

	// Save block
	if err := n.DB.SaveBlock(block); err != nil {
		return fmt.Errorf("failed to save block: %w", err)
	}

	// Remove committed transactions from pending
	for _, tx := range block.Transactions {
		n.DB.RemovePendingTransaction(tx)
	}

	log.Printf("[%s] Block %d committed successfully", n.ID, block.Index)
	return nil
}

// voteOnBlock decides whether to vote for a proposed block
func (n *Node) voteOnBlock(block *blockchain.Block) (*consensus.Vote, error) {
	log.Printf("[%s] Voting on block %d", n.ID, block.Index)

	// Basic validation
	var previousBlock *blockchain.Block
	if block.Index > 0 {
		var err error
		previousBlock, err = n.DB.GetBlockByIndex(block.Index - 1)
		if err != nil {
			return nil, fmt.Errorf("failed to get previous block: %w", err)
		}
	}

	// Create vote
	vote := &consensus.Vote{
		NodeID:    n.ID,
		BlockHash: block.Hash(),
		Approve:   block.IsValid(previousBlock),
		Timestamp: time.Now().Unix(),
	}

	log.Printf("[%s] Vote: %t for block %x", n.ID, vote.Approve, vote.BlockHash)
	return vote, nil
}

// broadcastBlock broadcasts a block to all peers using P2P
func (n *Node) broadcastBlock(block *blockchain.Block) error {
	log.Printf("[%s] Broadcasting block %x to peers", n.ID, block.Hash())
	return n.P2PManager.BroadcastBlock(block)
}

// broadcastVote broadcasts a vote to all peers using P2P
func (n *Node) broadcastVote(vote *consensus.Vote) error {
	log.Printf("[%s] Broadcasting vote to peers", n.ID)
	return n.P2PManager.BroadcastVote(vote)
}

// getPendingTransactions returns pending transactions for block creation
func (n *Node) getPendingTransactions() []*blockchain.Transaction {
	transactions, err := n.DB.GetPendingTransactions()
	if err != nil {
		log.Printf("[%s] Failed to get pending transactions: %v", n.ID, err)
		return nil
	}
	return transactions
}

// getLatestBlock returns the latest block
func (n *Node) getLatestBlock() *blockchain.Block {
	block, err := n.DB.GetLatestBlock()
	if err != nil {
		log.Printf("[%s] Failed to get latest block: %v", n.ID, err)
		return nil
	}
	return block
}

// AddTransaction adds a new transaction to the node
func (n *Node) AddTransaction(tx *blockchain.Transaction) error {
	// Add to pending transactions
	if err := n.DB.SaveTransaction(tx); err != nil {
		return fmt.Errorf("failed to add pending transaction: %w", err)
	}

	// Broadcast to peers if we have any
	if n.P2PManager != nil && n.P2PManager.GetPeerCount() > 0 {
		if err := n.P2PManager.BroadcastTransaction(tx); err != nil {
			log.Printf("[%s] Failed to broadcast transaction: %v", n.ID, err)
		}
	}

	log.Printf("[%s] Transaction added and broadcast", n.ID)
	return nil
}

// CreateUser creates a new wallet for a user
func (n *Node) CreateUser(name string) (*wallet.Wallet, error) {
	n.WalletMutex.Lock()
	defer n.WalletMutex.Unlock()

	// Check if user already exists
	if _, exists := n.WalletRegistry[name]; exists {
		return nil, fmt.Errorf("user %s already exists", name)
	}

	// Create new wallet
	w, err := wallet.NewWallet(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Register wallet
	n.WalletRegistry[name] = w

	// Save wallet to file
	walletPath := fmt.Sprintf("%s/wallets/%s.json", n.DataDir, name)
	if err := os.MkdirAll(fmt.Sprintf("%s/wallets", n.DataDir), 0755); err != nil {
		return nil, fmt.Errorf("failed to create wallets directory: %w", err)
	}

	if err := w.SaveToFile(walletPath); err != nil {
		return nil, fmt.Errorf("failed to save wallet: %w", err)
	}

	log.Printf("[%s] Created user: %s with address: %s", n.ID, name, w.GetAddressString())
	return w, nil
}

// GetUser retrieves a user's wallet
func (n *Node) GetUser(name string) (*wallet.Wallet, error) {
	n.WalletMutex.RLock()
	if w, exists := n.WalletRegistry[name]; exists {
		n.WalletMutex.RUnlock()
		return w, nil
	}
	n.WalletMutex.RUnlock()

	// Try to load from file
	walletPath := fmt.Sprintf("%s/wallets/%s.json", n.DataDir, name)
	w, err := wallet.LoadFromFile(walletPath)
	if err != nil {
		return nil, fmt.Errorf("user %s not found", name)
	}

	// Register in memory
	n.WalletMutex.Lock()
	n.WalletRegistry[name] = w
	n.WalletMutex.Unlock()

	return w, nil
}

// SendTransaction creates and sends a transaction
func (n *Node) SendTransaction(senderName, receiverName string, amount float64) error {
	// Get sender wallet
	senderWallet, err := n.GetUser(senderName)
	if err != nil {
		return fmt.Errorf("sender not found: %w", err)
	}

	// Get receiver wallet
	receiverWallet, err := n.GetUser(receiverName)
	if err != nil {
		return fmt.Errorf("receiver not found: %w", err)
	}

	// Create transaction
	tx := &blockchain.Transaction{
		Sender:    []byte(senderWallet.GetAddressString()),
		Receiver:  []byte(receiverWallet.GetAddressString()),
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}

	// Sign transaction
	if err := tx.Sign(senderWallet.PrivateKey); err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Add transaction to node
	return n.AddTransaction(tx)
}

// GetChainInfo returns information about the blockchain
func (n *Node) GetChainInfo() (map[string]interface{}, error) {
	return n.DB.GetChainInfo()
}

func main() {
	// Get environment variables
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "node1"
	}

	isLeader := os.Getenv("IS_LEADER") == "true"

	peers := []string{}
	if peerList := os.Getenv("PEERS"); peerList != "" {
		peers = strings.Split(peerList, ",")
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":50051"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Create and start node
	node, err := NewNode(nodeID, isLeader, peers, listenAddr, dataDir)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	// Keep the process running
	select {}
}
