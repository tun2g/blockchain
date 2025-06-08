package storage

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/blockchain/pkg/blockchain"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// BlockchainDB represents the LevelDB storage for blockchain data
type BlockchainDB struct {
	db *leveldb.DB
}

// NewBlockchainDB creates a new blockchain database
func NewBlockchainDB(dbPath string) (*BlockchainDB, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open LevelDB: %w", err)
	}

	return &BlockchainDB{db: db}, nil
}

// Close closes the database
func (bdb *BlockchainDB) Close() error {
	return bdb.db.Close()
}

// SaveBlock saves a block to the database
func (bdb *BlockchainDB) SaveBlock(block *blockchain.Block) error {
	// Save by block hash
	blockBytes, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	// Save block by hash
	hashKey := fmt.Sprintf("block_hash_%x", block.Hash())
	err = bdb.db.Put([]byte(hashKey), blockBytes, nil)
	if err != nil {
		return fmt.Errorf("failed to save block by hash: %w", err)
	}

	// Save block by index
	indexKey := fmt.Sprintf("block_index_%d", block.Index)
	err = bdb.db.Put([]byte(indexKey), blockBytes, nil)
	if err != nil {
		return fmt.Errorf("failed to save block by index: %w", err)
	}

	// Update latest block height
	err = bdb.db.Put([]byte("latest_height"), []byte(strconv.Itoa(block.Index)), nil)
	if err != nil {
		return fmt.Errorf("failed to update latest height: %w", err)
	}

	return nil
}

// GetBlockByHash retrieves a block by its hash
func (bdb *BlockchainDB) GetBlockByHash(hash []byte) (*blockchain.Block, error) {
	key := fmt.Sprintf("block_hash_%x", hash)
	blockBytes, err := bdb.db.Get([]byte(key), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by hash: %w", err)
	}

	var block blockchain.Block
	err = json.Unmarshal(blockBytes, &block)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

// GetBlockByIndex retrieves a block by its index
func (bdb *BlockchainDB) GetBlockByIndex(index int) (*blockchain.Block, error) {
	key := fmt.Sprintf("block_index_%d", index)
	blockBytes, err := bdb.db.Get([]byte(key), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get block by index: %w", err)
	}

	var block blockchain.Block
	err = json.Unmarshal(blockBytes, &block)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal block: %w", err)
	}

	return &block, nil
}

// GetLatestBlock retrieves the latest block in the chain
func (bdb *BlockchainDB) GetLatestBlock() (*blockchain.Block, error) {
	height, err := bdb.GetLatestHeight()
	if err != nil {
		return nil, err
	}

	return bdb.GetBlockByIndex(height)
}

// GetLatestHeight returns the height of the latest block
func (bdb *BlockchainDB) GetLatestHeight() (int, error) {
	heightBytes, err := bdb.db.Get([]byte("latest_height"), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return -1, nil // No blocks yet
		}
		return -1, fmt.Errorf("failed to get latest height: %w", err)
	}

	height, err := strconv.Atoi(string(heightBytes))
	if err != nil {
		return -1, fmt.Errorf("failed to parse height: %w", err)
	}

	return height, nil
}

// BlockExists checks if a block exists by hash
func (bdb *BlockchainDB) BlockExists(hash []byte) bool {
	key := fmt.Sprintf("block_hash_%x", hash)
	_, err := bdb.db.Get([]byte(key), nil)
	return err == nil
}

// GetBlocksFromHeight returns blocks starting from a specific height
func (bdb *BlockchainDB) GetBlocksFromHeight(startHeight int) ([]*blockchain.Block, error) {
	latestHeight, err := bdb.GetLatestHeight()
	if err != nil {
		return nil, err
	}

	if startHeight > latestHeight {
		return []*blockchain.Block{}, nil
	}

	var blocks []*blockchain.Block
	for i := startHeight; i <= latestHeight; i++ {
		block, err := bdb.GetBlockByIndex(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get block at height %d: %w", i, err)
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// SaveTransaction saves a transaction to the database (for pending transactions)
func (bdb *BlockchainDB) SaveTransaction(tx *blockchain.Transaction) error {
	txBytes, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	key := fmt.Sprintf("pending_tx_%x", tx.Hash())
	err = bdb.db.Put([]byte(key), txBytes, nil)
	if err != nil {
		return fmt.Errorf("failed to save pending transaction: %w", err)
	}

	return nil
}

// GetPendingTransactions retrieves all pending transactions
func (bdb *BlockchainDB) GetPendingTransactions() ([]*blockchain.Transaction, error) {
	var transactions []*blockchain.Transaction

	iter := bdb.db.NewIterator(util.BytesPrefix([]byte("pending_tx_")), nil)
	defer iter.Release()

	for iter.Next() {
		var tx blockchain.Transaction
		err := json.Unmarshal(iter.Value(), &tx)
		if err != nil {
			continue // Skip invalid transactions
		}
		transactions = append(transactions, &tx)
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("failed to iterate pending transactions: %w", err)
	}

	return transactions, nil
}

// RemovePendingTransaction removes a pending transaction
func (bdb *BlockchainDB) RemovePendingTransaction(tx *blockchain.Transaction) error {
	key := fmt.Sprintf("pending_tx_%x", tx.Hash())
	err := bdb.db.Delete([]byte(key), nil)
	if err != nil {
		return fmt.Errorf("failed to remove pending transaction: %w", err)
	}

	return nil
}

// GetChainInfo returns basic information about the blockchain
func (bdb *BlockchainDB) GetChainInfo() (map[string]interface{}, error) {
	height, err := bdb.GetLatestHeight()
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"latest_height": height,
	}

	if height >= 0 {
		latestBlock, err := bdb.GetLatestBlock()
		if err != nil {
			return nil, err
		}

		info["latest_block_hash"] = fmt.Sprintf("%x", latestBlock.Hash())
		info["latest_block_timestamp"] = latestBlock.Timestamp
		info["total_transactions"] = 0 // Could calculate this by iterating through all blocks
	}

	pendingTxs, err := bdb.GetPendingTransactions()
	if err == nil {
		info["pending_transactions"] = len(pendingTxs)
	}

	return info, nil
}

// ValidateChain validates the entire blockchain for consistency
func (bdb *BlockchainDB) ValidateChain() error {
	height, err := bdb.GetLatestHeight()
	if err != nil {
		return err
	}

	if height < 0 {
		return nil // Empty chain is valid
	}

	// Check genesis block
	genesisBlock, err := bdb.GetBlockByIndex(0)
	if err != nil {
		return fmt.Errorf("failed to get genesis block: %w", err)
	}

	if len(genesisBlock.PreviousHash) != 0 {
		return fmt.Errorf("genesis block should have empty previous hash")
	}

	// Validate chain integrity
	var previousBlock *blockchain.Block = genesisBlock

	for i := 1; i <= height; i++ {
		currentBlock, err := bdb.GetBlockByIndex(i)
		if err != nil {
			return fmt.Errorf("failed to get block at height %d: %w", i, err)
		}

		if !currentBlock.IsValid(previousBlock) {
			return fmt.Errorf("invalid block at height %d", i)
		}

		previousBlock = currentBlock
	}

	return nil
}
