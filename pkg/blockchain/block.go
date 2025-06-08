package blockchain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Block represents a block in the blockchain
type Block struct {
	Index            int            `json:"index"`              // Block number
	Timestamp        int64          `json:"timestamp"`          // Unix timestamp
	Transactions     []*Transaction `json:"transactions"`       // List of transactions
	MerkleRoot       []byte         `json:"merkle_root"`        // Merkle tree root hash
	PreviousHash     []byte         `json:"previous_hash"`      // Hash of previous block
	CurrentBlockHash []byte         `json:"current_block_hash"` // Hash of current block
	Nonce            int            `json:"nonce"`              // For potential PoW (unused in this implementation)
}

// NewBlock creates a new block
func NewBlock(index int, transactions []*Transaction, previousHash []byte) *Block {
	block := &Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: transactions,
		PreviousHash: previousHash,
		Nonce:        0,
	}

	// Calculate Merkle root
	block.MerkleRoot = CalculateMerkleRoot(transactions)

	// Calculate block hash
	block.CurrentBlockHash = block.CalculateHash()

	return block
}

// NewGenesisBlock creates the first block in the blockchain
func NewGenesisBlock() *Block {
	// Genesis block has no previous hash and no transactions
	return NewBlock(0, []*Transaction{}, []byte{})
}

// CalculateHash calculates the hash of the block
func (b *Block) CalculateHash() []byte {
	// Create a copy without the current hash to avoid circular dependency
	blockData := struct {
		Index        int            `json:"index"`
		Timestamp    int64          `json:"timestamp"`
		Transactions []*Transaction `json:"transactions"`
		MerkleRoot   []byte         `json:"merkle_root"`
		PreviousHash []byte         `json:"previous_hash"`
		Nonce        int            `json:"nonce"`
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		MerkleRoot:   b.MerkleRoot,
		PreviousHash: b.PreviousHash,
		Nonce:        b.Nonce,
	}

	data, err := json.Marshal(blockData)
	if err != nil {
		// In production, this should be handled properly
		return nil
	}

	hash := sha256.Sum256(data)
	return hash[:]
}

// Hash returns the current block hash
func (b *Block) Hash() []byte {
	if b.CurrentBlockHash == nil {
		b.CurrentBlockHash = b.CalculateHash()
	}
	return b.CurrentBlockHash
}

// IsValid validates the block
func (b *Block) IsValid(previousBlock *Block) bool {
	// Check if block index is correct
	if previousBlock != nil && b.Index != previousBlock.Index+1 {
		return false
	}

	// Check if previous hash is correct
	if previousBlock != nil {
		prevHash := previousBlock.Hash()
		if len(b.PreviousHash) != len(prevHash) {
			return false
		}
		for i := range b.PreviousHash {
			if b.PreviousHash[i] != prevHash[i] {
				return false
			}
		}
	}

	// Check if current hash is correct
	calculatedHash := b.CalculateHash()
	if len(b.CurrentBlockHash) != len(calculatedHash) {
		return false
	}
	for i := range b.CurrentBlockHash {
		if b.CurrentBlockHash[i] != calculatedHash[i] {
			return false
		}
	}

	// Check if Merkle root is correct
	calculatedMerkleRoot := CalculateMerkleRoot(b.Transactions)
	if len(b.MerkleRoot) != len(calculatedMerkleRoot) {
		return false
	}
	for i := range b.MerkleRoot {
		if b.MerkleRoot[i] != calculatedMerkleRoot[i] {
			return false
		}
	}

	// Validate all transactions
	for _, tx := range b.Transactions {
		// Note: In a real implementation, you would need to pass the correct public key
		// This is a simplified validation that just checks if signature exists
		if len(tx.Signature) == 0 {
			return false
		}
	}

	return true
}

// AddTransaction adds a transaction to the block (for building blocks)
func (b *Block) AddTransaction(tx *Transaction) {
	b.Transactions = append(b.Transactions, tx)
	// Recalculate Merkle root and block hash
	b.MerkleRoot = CalculateMerkleRoot(b.Transactions)
	b.CurrentBlockHash = b.CalculateHash()
}

// GetTransactionCount returns the number of transactions in the block
func (b *Block) GetTransactionCount() int {
	return len(b.Transactions)
}

// String returns a string representation of the block
func (b *Block) String() string {
	return fmt.Sprintf("Block{Index: %d, Timestamp: %d, TxCount: %d, Hash: %x, PrevHash: %x}",
		b.Index, b.Timestamp, len(b.Transactions), b.CurrentBlockHash, b.PreviousHash)
}

// ToJSON converts the block to JSON
func (b *Block) ToJSON() ([]byte, error) {
	return json.MarshalIndent(b, "", "  ")
}

// FromJSON creates a block from JSON data
func FromJSON(data []byte) (*Block, error) {
	var block Block
	err := json.Unmarshal(data, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}
