package blockchain

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// Transaction represents a blockchain transaction
type Transaction struct {
	Sender    []byte  `json:"sender"`    // Public key or address of sender
	Receiver  []byte  `json:"receiver"`  // Public key or address of receiver
	Amount    float64 `json:"amount"`    // Amount to transfer
	Timestamp int64   `json:"timestamp"` // Unix timestamp
	Signature []byte  `json:"signature"` // ECDSA signature (R and S concatenated)
}

// NewTransaction creates a new transaction
func NewTransaction(sender, receiver []byte, amount float64) *Transaction {
	return &Transaction{
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		Timestamp: time.Now().Unix(),
	}
}

// Hash creates a hash of the transaction (excluding signature)
func (t *Transaction) Hash() []byte {
	txCopy := *t
	txCopy.Signature = nil // Exclude signature from hash

	data, err := json.Marshal(txCopy)
	if err != nil {
		// In production, this should be handled properly
		return nil
	}

	hash := sha256.Sum256(data)
	return hash[:]
}

// Sign signs the transaction with the given private key
func (t *Transaction) Sign(privKey *ecdsa.PrivateKey) error {
	txHash := t.Hash()
	if txHash == nil {
		return fmt.Errorf("failed to create transaction hash")
	}

	r, s, err := ecdsa.Sign(rand.Reader, privKey, txHash)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Store R and S as concatenated byte slices
	// We need to ensure consistent byte length for proper reconstruction
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Pad to 32 bytes for secp256r1 curve
	if len(rBytes) < 32 {
		rPadded := make([]byte, 32)
		copy(rPadded[32-len(rBytes):], rBytes)
		rBytes = rPadded
	}
	if len(sBytes) < 32 {
		sPadded := make([]byte, 32)
		copy(sPadded[32-len(sBytes):], sBytes)
		sBytes = sPadded
	}

	t.Signature = append(rBytes, sBytes...)
	return nil
}

// Verify verifies the transaction signature
func (t *Transaction) Verify(pubKey *ecdsa.PublicKey) bool {
	if len(t.Signature) != 64 { // 32 bytes for R + 32 bytes for S
		return false
	}

	txHash := t.Hash()
	if txHash == nil {
		return false
	}

	// Extract R and S from signature
	rBytes := t.Signature[:32]
	sBytes := t.Signature[32:]

	r := new(big.Int).SetBytes(rBytes)
	s := new(big.Int).SetBytes(sBytes)

	return ecdsa.Verify(pubKey, txHash, r, s)
}

// String returns a string representation of the transaction
func (t *Transaction) String() string {
	return fmt.Sprintf("Transaction{Sender: %x, Receiver: %x, Amount: %.2f, Timestamp: %d}",
		t.Sender, t.Receiver, t.Amount, t.Timestamp)
}
