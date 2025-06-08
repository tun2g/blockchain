package p2p

import (
	"context"
	"fmt"
	"log"

	"github.com/blockchain/pkg/blockchain"
	"github.com/blockchain/pkg/consensus"
	"github.com/blockchain/pkg/storage"
	pb "github.com/blockchain/proto"
)

// Server represents a gRPC server for blockchain communication
type Server struct {
	pb.UnimplementedBlockchainServiceServer
	nodeID          string
	db              *storage.BlockchainDB
	consensusEngine *consensus.ConsensusEngine

	// Callbacks for handling operations
	onTransactionReceived func(*blockchain.Transaction) error
	onBlockProposed       func(*blockchain.Block, string) error
	onVoteReceived        func(*consensus.Vote) error
}

// NewServer creates a new P2P server
func NewServer(nodeID string, db *storage.BlockchainDB, consensusEngine *consensus.ConsensusEngine) *Server {
	return &Server{
		nodeID:          nodeID,
		db:              db,
		consensusEngine: consensusEngine,
	}
}

// SetCallbacks sets the callback functions for handling operations
func (s *Server) SetCallbacks(
	onTransactionReceived func(*blockchain.Transaction) error,
	onBlockProposed func(*blockchain.Block, string) error,
	onVoteReceived func(*consensus.Vote) error,
) {
	s.onTransactionReceived = onTransactionReceived
	s.onBlockProposed = onBlockProposed
	s.onVoteReceived = onVoteReceived
}

// SendTransaction handles incoming transaction requests
func (s *Server) SendTransaction(ctx context.Context, req *pb.SendTransactionRequest) (*pb.SendTransactionResponse, error) {
	log.Printf("[%s] Received transaction from peer", s.nodeID)

	// Convert protobuf transaction to internal format
	tx := &blockchain.Transaction{
		Sender:    req.Transaction.Sender,
		Receiver:  req.Transaction.Receiver,
		Amount:    req.Transaction.Amount,
		Timestamp: req.Transaction.Timestamp,
		Signature: req.Transaction.Signature,
	}

	// Handle the transaction
	if s.onTransactionReceived != nil {
		if err := s.onTransactionReceived(tx); err != nil {
			return &pb.SendTransactionResponse{
				Success: false,
				Message: fmt.Sprintf("Transaction rejected: %v", err),
			}, nil
		}
	}

	return &pb.SendTransactionResponse{
		Success: true,
		Message: "Transaction accepted",
	}, nil
}

// ProposeBlock handles incoming block proposals
func (s *Server) ProposeBlock(ctx context.Context, req *pb.ProposeBlockRequest) (*pb.ProposeBlockResponse, error) {
	log.Printf("[%s] Received block proposal from %s", s.nodeID, req.ProposerId)

	// Convert protobuf block to internal format
	block := protoToInternalBlock(req.Block)

	// Handle the block proposal
	if s.onBlockProposed != nil {
		if err := s.onBlockProposed(block, req.ProposerId); err != nil {
			return &pb.ProposeBlockResponse{
				Success: false,
				Message: fmt.Sprintf("Block proposal rejected: %v", err),
			}, nil
		}
	}

	return &pb.ProposeBlockResponse{
		Success: true,
		Message: "Block proposal accepted",
	}, nil
}

// Vote handles incoming votes
func (s *Server) Vote(ctx context.Context, req *pb.VoteRequest) (*pb.VoteResponse, error) {
	log.Printf("[%s] Received vote from %s", s.nodeID, req.Vote.NodeId)

	// Convert protobuf vote to internal format
	vote := &consensus.Vote{
		NodeID:    req.Vote.NodeId,
		BlockHash: req.Vote.BlockHash,
		Approve:   req.Vote.Approve,
		Timestamp: req.Vote.Timestamp,
	}

	// Handle the vote
	if s.onVoteReceived != nil {
		if err := s.onVoteReceived(vote); err != nil {
			return &pb.VoteResponse{
				Success: false,
				Message: fmt.Sprintf("Vote rejected: %v", err),
			}, nil
		}
	}

	return &pb.VoteResponse{
		Success: true,
		Message: "Vote accepted",
	}, nil
}

// GetBlockByHeight handles requests for blocks by height
func (s *Server) GetBlockByHeight(ctx context.Context, req *pb.GetBlockByHeightRequest) (*pb.GetBlockByHeightResponse, error) {
	block, err := s.db.GetBlockByIndex(int(req.Height))
	if err != nil {
		return &pb.GetBlockByHeightResponse{
			Found: false,
		}, nil
	}

	protoBlock := internalToProtoBlock(block)
	return &pb.GetBlockByHeightResponse{
		Block: protoBlock,
		Found: true,
	}, nil
}

// GetLatestBlock handles requests for the latest block
func (s *Server) GetLatestBlock(ctx context.Context, req *pb.GetLatestBlockRequest) (*pb.GetLatestBlockResponse, error) {
	block, err := s.db.GetLatestBlock()
	if err != nil {
		return &pb.GetLatestBlockResponse{
			Found: false,
		}, nil
	}

	protoBlock := internalToProtoBlock(block)
	return &pb.GetLatestBlockResponse{
		Block: protoBlock,
		Found: true,
	}, nil
}

// GetBlocksFromHeight handles requests for multiple blocks
func (s *Server) GetBlocksFromHeight(ctx context.Context, req *pb.GetBlocksFromHeightRequest) (*pb.GetBlocksFromHeightResponse, error) {
	blocks, err := s.db.GetBlocksFromHeight(int(req.StartHeight))
	if err != nil {
		return &pb.GetBlocksFromHeightResponse{
			Blocks: []*pb.Block{},
		}, nil
	}

	var protoBlocks []*pb.Block
	for _, block := range blocks {
		protoBlocks = append(protoBlocks, internalToProtoBlock(block))
	}

	return &pb.GetBlocksFromHeightResponse{
		Blocks: protoBlocks,
	}, nil
}

// GetChainInfo handles requests for chain information
func (s *Server) GetChainInfo(ctx context.Context, req *pb.GetChainInfoRequest) (*pb.GetChainInfoResponse, error) {
	info, err := s.db.GetChainInfo()
	if err != nil {
		return &pb.GetChainInfoResponse{
			LatestHeight: -1,
		}, nil
	}

	latestHeight := int32(-1)
	var latestBlockHash []byte
	var latestBlockTimestamp int64
	pendingTransactions := int32(0)

	if height, ok := info["latest_height"].(int); ok {
		latestHeight = int32(height)
	}

	if hash, ok := info["latest_block_hash"].(string); ok {
		latestBlockHash = []byte(hash)
	}

	if timestamp, ok := info["latest_block_timestamp"].(int64); ok {
		latestBlockTimestamp = timestamp
	}

	if pending, ok := info["pending_transactions"].(int); ok {
		pendingTransactions = int32(pending)
	}

	return &pb.GetChainInfoResponse{
		LatestHeight:         latestHeight,
		LatestBlockHash:      latestBlockHash,
		LatestBlockTimestamp: latestBlockTimestamp,
		PendingTransactions:  pendingTransactions,
	}, nil
}

// Sync handles synchronization requests
func (s *Server) Sync(ctx context.Context, req *pb.SyncRequest) (*pb.SyncResponse, error) {
	log.Printf("[%s] Sync request from %s (current height: %d)", s.nodeID, req.NodeId, req.CurrentHeight)

	// Get blocks from the requested height onwards
	blocks, err := s.db.GetBlocksFromHeight(int(req.CurrentHeight + 1))
	if err != nil {
		return &pb.SyncResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to get blocks: %v", err),
		}, nil
	}

	var protoBlocks []*pb.Block
	for _, block := range blocks {
		protoBlocks = append(protoBlocks, internalToProtoBlock(block))
	}

	return &pb.SyncResponse{
		Blocks:  protoBlocks,
		Success: true,
		Message: fmt.Sprintf("Synchronized %d blocks", len(protoBlocks)),
	}, nil
}

// Helper functions to convert between internal and protobuf formats

func protoToInternalBlock(protoBlock *pb.Block) *blockchain.Block {
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

func internalToProtoBlock(block *blockchain.Block) *pb.Block {
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

func internalToProtoVote(vote *consensus.Vote) *pb.Vote {
	return &pb.Vote{
		NodeId:    vote.NodeID,
		BlockHash: vote.BlockHash,
		Approve:   vote.Approve,
		Timestamp: vote.Timestamp,
	}
}
