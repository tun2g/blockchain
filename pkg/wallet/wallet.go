package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

// Wallet represents a user's wallet with ECDSA key pair
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey `json:"-"` // Don't serialize private key
	PublicKey  *ecdsa.PublicKey  `json:"-"` // Don't serialize public key
	Address    []byte            `json:"address"`
	Name       string            `json:"name"`
}

// WalletData represents serializable wallet data
type WalletData struct {
	PrivateKeyHex string `json:"private_key"`
	Address       string `json:"address"`
	Name          string `json:"name"`
}

// NewWallet creates a new wallet with a fresh ECDSA key pair
func NewWallet(name string) (*Wallet, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	wallet := &Wallet{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
		Name:       name,
	}

	wallet.Address = wallet.GenerateAddress()

	return wallet, nil
}

// GenerateAddress generates an address from the public key
func (w *Wallet) GenerateAddress() []byte {
	// Convert public key to bytes
	pubKeyBytes := append(w.PublicKey.X.Bytes(), w.PublicKey.Y.Bytes()...)

	// Hash the public key to create address
	hash := sha256.Sum256(pubKeyBytes)

	// Take first 20 bytes (similar to Ethereum)
	return hash[:20]
}

// GetAddressString returns the address as a hex string
func (w *Wallet) GetAddressString() string {
	return hex.EncodeToString(w.Address)
}

// GetPrivateKeyHex returns the private key as a hex string
func (w *Wallet) GetPrivateKeyHex() string {
	return hex.EncodeToString(w.PrivateKey.D.Bytes())
}

// SaveToFile saves the wallet to a JSON file
func (w *Wallet) SaveToFile(filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	walletData := WalletData{
		PrivateKeyHex: w.GetPrivateKeyHex(),
		Address:       w.GetAddressString(),
		Name:          w.Name,
	}

	data, err := json.MarshalIndent(walletData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wallet data: %w", err)
	}

	err = os.WriteFile(filePath, data, 0600) // Restrict permissions for security
	if err != nil {
		return fmt.Errorf("failed to write wallet file: %w", err)
	}

	return nil
}

// LoadFromFile loads a wallet from a JSON file
func LoadFromFile(filePath string) (*Wallet, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wallet file: %w", err)
	}

	var walletData WalletData
	err = json.Unmarshal(data, &walletData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet data: %w", err)
	}

	return LoadFromHex(walletData.PrivateKeyHex, walletData.Name)
}

// LoadFromHex loads a wallet from a hex-encoded private key
func LoadFromHex(privateKeyHex, name string) (*Wallet, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	privKey := new(ecdsa.PrivateKey)
	privKey.PublicKey.Curve = elliptic.P256()
	privKey.D = new(big.Int).SetBytes(privateKeyBytes)
	privKey.PublicKey.X, privKey.PublicKey.Y = privKey.PublicKey.Curve.ScalarBaseMult(privateKeyBytes)

	wallet := &Wallet{
		PrivateKey: privKey,
		PublicKey:  &privKey.PublicKey,
		Name:       name,
	}

	wallet.Address = wallet.GenerateAddress()

	return wallet, nil
}

// Sign signs data with the wallet's private key
func (w *Wallet) Sign(data []byte) ([]byte, error) {
	hash := sha256.Sum256(data)
	r, s, err := ecdsa.Sign(rand.Reader, w.PrivateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	// Pad to 32 bytes and concatenate R and S
	rBytes := r.Bytes()
	sBytes := s.Bytes()

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

	signature := append(rBytes, sBytes...)
	return signature, nil
}

// VerifySignature verifies a signature against the wallet's public key
func (w *Wallet) VerifySignature(data, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}

	hash := sha256.Sum256(data)

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	return ecdsa.Verify(w.PublicKey, hash[:], r, s)
}

// String returns a string representation of the wallet
func (w *Wallet) String() string {
	return fmt.Sprintf("Wallet{Name: %s, Address: %s}", w.Name, w.GetAddressString())
}

// AddressFromPublicKey generates an address from a public key
func AddressFromPublicKey(pubKey *ecdsa.PublicKey) []byte {
	pubKeyBytes := append(pubKey.X.Bytes(), pubKey.Y.Bytes()...)
	hash := sha256.Sum256(pubKeyBytes)
	return hash[:20]
}

// IsValidAddress checks if an address is valid (basic length check)
func IsValidAddress(address []byte) bool {
	return len(address) == 20
}

// GetDefaultWalletPath returns the default path for wallet files
func GetDefaultWalletPath(name string) string {
	return filepath.Join("wallets", name+".json")
}
