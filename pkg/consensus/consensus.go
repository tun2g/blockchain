package consensus

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blockchain/pkg/blockchain"
)

// NodeState represents the state of a consensus node
type NodeState int

const (
	Follower NodeState = iota
	Leader
	Syncing
)

func (ns NodeState) String() string {
	switch ns {
	case Follower:
		return "Follower"
	case Leader:
		return "Leader"
	case Syncing:
		return "Syncing"
	default:
		return "Unknown"
	}
}

// Vote represents a vote for a proposed block
type Vote struct {
	NodeID    string
	BlockHash []byte
	Approve   bool
	Timestamp int64
}

// ConsensusEngine manages the consensus mechanism
type ConsensusEngine struct {
	nodeID           string
	isLeader         bool
	state            NodeState
	peers            []string
	pendingVotes     map[string][]*Vote           // blockHash -> votes
	blockProposals   map[string]*blockchain.Block // blockHash -> block
	voteMutex        sync.RWMutex
	proposalMutex    sync.RWMutex
	consensusTimeout time.Duration
	blockInterval    time.Duration

	// Callbacks
	onBlockCommitted func(*blockchain.Block) error
	onVoteNeeded     func(*blockchain.Block) (*Vote, error)
	broadcastBlock   func(*blockchain.Block) error
	broadcastVote    func(*Vote) error
}

// NewConsensusEngine creates a new consensus engine
func NewConsensusEngine(nodeID string, isLeader bool, peers []string) *ConsensusEngine {
	state := Follower
	if isLeader {
		state = Leader
	}

	return &ConsensusEngine{
		nodeID:           nodeID,
		isLeader:         isLeader,
		state:            state,
		peers:            peers,
		pendingVotes:     make(map[string][]*Vote),
		blockProposals:   make(map[string]*blockchain.Block),
		consensusTimeout: 10 * time.Second,
		blockInterval:    5 * time.Second,
	}
}

// SetCallbacks sets the callback functions
func (ce *ConsensusEngine) SetCallbacks(
	onBlockCommitted func(*blockchain.Block) error,
	onVoteNeeded func(*blockchain.Block) (*Vote, error),
	broadcastBlock func(*blockchain.Block) error,
	broadcastVote func(*Vote) error,
) {
	ce.onBlockCommitted = onBlockCommitted
	ce.onVoteNeeded = onVoteNeeded
	ce.broadcastBlock = broadcastBlock
	ce.broadcastVote = broadcastVote
}

// GetState returns the current node state
func (ce *ConsensusEngine) GetState() NodeState {
	return ce.state
}

// IsLeader returns true if this node is the leader
func (ce *ConsensusEngine) IsLeader() bool {
	return ce.state == Leader
}

// ProposeBlock proposes a new block (only for leader)
func (ce *ConsensusEngine) ProposeBlock(block *blockchain.Block) error {
	if ce.state != Leader {
		return fmt.Errorf("only leader can propose blocks")
	}

	blockHashStr := fmt.Sprintf("%x", block.Hash())

	ce.proposalMutex.Lock()
	ce.blockProposals[blockHashStr] = block
	ce.proposalMutex.Unlock()

	ce.voteMutex.Lock()
	ce.pendingVotes[blockHashStr] = []*Vote{}
	ce.voteMutex.Unlock()

	log.Printf("[%s] Proposing block %s with %d transactions",
		ce.nodeID, blockHashStr[:8], len(block.Transactions))

	// Broadcast the block to all peers
	if ce.broadcastBlock != nil {
		if err := ce.broadcastBlock(block); err != nil {
			return fmt.Errorf("failed to broadcast block: %w", err)
		}
	}

	// Start consensus timer
	go ce.waitForConsensus(blockHashStr)

	return nil
}

// HandleBlockProposal handles a block proposal from the leader
func (ce *ConsensusEngine) HandleBlockProposal(block *blockchain.Block, proposerID string) error {
	if ce.state == Leader {
		return fmt.Errorf("leader should not receive block proposals")
	}

	blockHashStr := fmt.Sprintf("%x", block.Hash())

	ce.proposalMutex.Lock()
	ce.blockProposals[blockHashStr] = block
	ce.proposalMutex.Unlock()

	log.Printf("[%s] Received block proposal %s from %s",
		ce.nodeID, blockHashStr[:8], proposerID)

	// Request vote decision
	if ce.onVoteNeeded != nil {
		vote, err := ce.onVoteNeeded(block)
		if err != nil {
			return fmt.Errorf("failed to get vote decision: %w", err)
		}

		// Broadcast vote
		if ce.broadcastVote != nil {
			if err := ce.broadcastVote(vote); err != nil {
				return fmt.Errorf("failed to broadcast vote: %w", err)
			}
		}
	}

	return nil
}

// HandleVote handles a vote from a peer
func (ce *ConsensusEngine) HandleVote(vote *Vote) error {
	blockHashStr := fmt.Sprintf("%x", vote.BlockHash)

	ce.voteMutex.Lock()
	defer ce.voteMutex.Unlock()

	// Check if we have the corresponding block proposal
	ce.proposalMutex.RLock()
	_, exists := ce.blockProposals[blockHashStr]
	ce.proposalMutex.RUnlock()

	if !exists {
		return fmt.Errorf("received vote for unknown block %s", blockHashStr[:8])
	}

	// Add vote to pending votes
	if _, exists := ce.pendingVotes[blockHashStr]; !exists {
		ce.pendingVotes[blockHashStr] = []*Vote{}
	}

	// Check if vote already exists from this node
	for _, existingVote := range ce.pendingVotes[blockHashStr] {
		if existingVote.NodeID == vote.NodeID {
			return nil // Vote already recorded
		}
	}

	ce.pendingVotes[blockHashStr] = append(ce.pendingVotes[blockHashStr], vote)

	log.Printf("[%s] Received vote from %s for block %s: %t",
		ce.nodeID, vote.NodeID, blockHashStr[:8], vote.Approve)

	// Check if consensus is reached
	if ce.state == Leader {
		go ce.checkConsensus(blockHashStr)
	}

	return nil
}

// waitForConsensus waits for consensus on a proposed block
func (ce *ConsensusEngine) waitForConsensus(blockHashStr string) {
	timer := time.NewTimer(ce.consensusTimeout)
	defer timer.Stop()

	<-timer.C

	// Timeout reached, check final consensus
	ce.checkConsensus(blockHashStr)
}

// checkConsensus checks if consensus is reached for a block
func (ce *ConsensusEngine) checkConsensus(blockHashStr string) {
	ce.voteMutex.RLock()
	votes, exists := ce.pendingVotes[blockHashStr]
	ce.voteMutex.RUnlock()

	if !exists {
		return
	}

	ce.proposalMutex.RLock()
	block, blockExists := ce.blockProposals[blockHashStr]
	ce.proposalMutex.RUnlock()

	if !blockExists {
		return
	}

	totalNodes := len(ce.peers) + 1       // peers + this node
	requiredVotes := (totalNodes * 2) / 3 // 2/3 majority

	approvals := 0
	for _, vote := range votes {
		if vote.Approve {
			approvals++
		}
	}

	log.Printf("[%s] Consensus check for block %s: %d/%d approvals (%d required)",
		ce.nodeID, blockHashStr[:8], approvals, len(votes), requiredVotes)

	if approvals >= requiredVotes {
		// Consensus reached, commit the block
		log.Printf("[%s] Consensus reached for block %s, committing",
			ce.nodeID, blockHashStr[:8])

		if ce.onBlockCommitted != nil {
			if err := ce.onBlockCommitted(block); err != nil {
				log.Printf("[%s] Failed to commit block: %v", ce.nodeID, err)
			}
		}

		// Clean up
		ce.voteMutex.Lock()
		delete(ce.pendingVotes, blockHashStr)
		ce.voteMutex.Unlock()

		ce.proposalMutex.Lock()
		delete(ce.blockProposals, blockHashStr)
		ce.proposalMutex.Unlock()
	}
}

// StartBlockProduction starts the block production process (for leader)
func (ce *ConsensusEngine) StartBlockProduction(getTransactions func() []*blockchain.Transaction,
	getPreviousBlock func() *blockchain.Block) {

	if ce.state != Leader {
		return
	}

	ticker := time.NewTicker(ce.blockInterval)
	defer ticker.Stop()

	log.Printf("[%s] Starting block production with interval %v", ce.nodeID, ce.blockInterval)

	for {
		select {
		case <-ticker.C:
			// Get pending transactions
			transactions := getTransactions()
			if len(transactions) == 0 {
				continue // No transactions to include
			}

			// Get previous block
			previousBlock := getPreviousBlock()
			var previousHash []byte
			var index int

			if previousBlock != nil {
				previousHash = previousBlock.Hash()
				index = previousBlock.Index + 1
			} else {
				// Genesis block
				previousHash = []byte{}
				index = 0
			}

			// Create new block
			newBlock := blockchain.NewBlock(index, transactions, previousHash)

			// Propose the block
			if err := ce.ProposeBlock(newBlock); err != nil {
				log.Printf("[%s] Failed to propose block: %v", ce.nodeID, err)
			}
		}
	}
}

// SetSyncing sets the node to syncing state
func (ce *ConsensusEngine) SetSyncing() {
	ce.state = Syncing
	log.Printf("[%s] Node state changed to Syncing", ce.nodeID)
}

// SetOperational sets the node back to operational state
func (ce *ConsensusEngine) SetOperational() {
	if ce.isLeader {
		ce.state = Leader
	} else {
		ce.state = Follower
	}
	log.Printf("[%s] Node state changed to %s", ce.nodeID, ce.state)
}

// GetPendingProposals returns pending block proposals
func (ce *ConsensusEngine) GetPendingProposals() map[string]*blockchain.Block {
	ce.proposalMutex.RLock()
	defer ce.proposalMutex.RUnlock()

	proposals := make(map[string]*blockchain.Block)
	for hash, block := range ce.blockProposals {
		proposals[hash] = block
	}

	return proposals
}

// GetPendingVotes returns pending votes
func (ce *ConsensusEngine) GetPendingVotes() map[string][]*Vote {
	ce.voteMutex.RLock()
	defer ce.voteMutex.RUnlock()

	votes := make(map[string][]*Vote)
	for hash, voteList := range ce.pendingVotes {
		votes[hash] = make([]*Vote, len(voteList))
		copy(votes[hash], voteList)
	}

	return votes
}
