package blockchain

import (
	"crypto/sha256"
	"fmt"
)

// MerkleNode represents a node in the Merkle tree
type MerkleNode struct {
	Left  *MerkleNode
	Right *MerkleNode
	Data  []byte
}

// MerkleTree represents a Merkle tree
type MerkleTree struct {
	Root *MerkleNode
}

// NewMerkleNode creates a new Merkle node
func NewMerkleNode(left, right *MerkleNode, data []byte) *MerkleNode {
	node := &MerkleNode{}

	if left == nil && right == nil {
		// Leaf node
		hash := sha256.Sum256(data)
		node.Data = hash[:]
	} else {
		// Internal node
		prevHashes := append(left.Data, right.Data...)
		hash := sha256.Sum256(prevHashes)
		node.Data = hash[:]
	}

	node.Left = left
	node.Right = right

	return node
}

// NewMerkleTree creates a new Merkle tree from transaction data
func NewMerkleTree(data [][]byte) *MerkleTree {
	if len(data) == 0 {
		return &MerkleTree{}
	}

	var nodes []*MerkleNode

	// Create leaf nodes
	for _, datum := range data {
		node := NewMerkleNode(nil, nil, datum)
		nodes = append(nodes, node)
	}

	// Build the tree from bottom up
	for len(nodes) > 1 {
		var newLevel []*MerkleNode

		for i := 0; i < len(nodes); i += 2 {
			var left, right *MerkleNode
			left = nodes[i]

			if i+1 < len(nodes) {
				right = nodes[i+1]
			} else {
				// If odd number of nodes, duplicate the last one
				right = nodes[i]
			}

			node := NewMerkleNode(left, right, nil)
			newLevel = append(newLevel, node)
		}

		nodes = newLevel
	}

	return &MerkleTree{Root: nodes[0]}
}

// GetMerkleRoot returns the root hash of the Merkle tree
func (mt *MerkleTree) GetMerkleRoot() []byte {
	if mt.Root == nil {
		return nil
	}
	return mt.Root.Data
}

// CalculateMerkleRoot calculates the Merkle root from a list of transactions
func CalculateMerkleRoot(transactions []*Transaction) []byte {
	if len(transactions) == 0 {
		// Return hash of empty string for empty block
		hash := sha256.Sum256([]byte(""))
		return hash[:]
	}

	var txHashes [][]byte
	for _, tx := range transactions {
		txHashes = append(txHashes, tx.Hash())
	}

	tree := NewMerkleTree(txHashes)
	return tree.GetMerkleRoot()
}

// VerifyMerkleProof verifies a Merkle proof for a given transaction
func VerifyMerkleProof(txHash []byte, proof [][]byte, root []byte, index int) bool {
	currentHash := txHash

	for _, siblingHash := range proof {
		var combinedHash []byte

		// Determine the order based on the index
		if index%2 == 0 {
			// Current node is left child
			combinedHash = append(currentHash, siblingHash...)
		} else {
			// Current node is right child
			combinedHash = append(siblingHash, currentHash...)
		}

		hash := sha256.Sum256(combinedHash)
		currentHash = hash[:]
		index = index / 2
	}

	// Compare with the expected root
	if len(currentHash) != len(root) {
		return false
	}

	for i := range currentHash {
		if currentHash[i] != root[i] {
			return false
		}
	}

	return true
}

// String returns a string representation of the Merkle tree
func (mt *MerkleTree) String() string {
	if mt.Root == nil {
		return "Empty Merkle Tree"
	}
	return fmt.Sprintf("Merkle Tree Root: %x", mt.Root.Data)
}
