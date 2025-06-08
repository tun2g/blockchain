package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/blockchain/pkg/blockchain"
	"github.com/blockchain/pkg/storage"
	"github.com/blockchain/pkg/wallet"
	"github.com/spf13/cobra"
)

var (
	dataDir  string
	nodeAddr string
)

var rootCmd = &cobra.Command{
	Use:   "blockchain-cli",
	Short: "Blockchain CLI tool",
	Long:  "A command line interface for interacting with the blockchain",
}

var createUserCmd = &cobra.Command{
	Use:   "create-user [name]",
	Short: "Create a new user wallet",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		userName := args[0]

		// Create wallet
		userWallet, err := wallet.NewWallet(userName)
		if err != nil {
			log.Fatalf("Failed to create wallet: %v", err)
		}

		// Save wallet to file
		walletPath := wallet.GetDefaultWalletPath(userName)
		if err := userWallet.SaveToFile(walletPath); err != nil {
			log.Fatalf("Failed to save wallet: %v", err)
		}

		fmt.Printf("Created user: %s\n", userName)
		fmt.Printf("Address: %s\n", userWallet.GetAddressString())
		fmt.Printf("Private Key: %s\n", userWallet.GetPrivateKeyHex())
		fmt.Printf("Wallet saved to: %s\n", walletPath)
	},
}

var sendTxCmd = &cobra.Command{
	Use:   "send [from] [to] [amount]",
	Short: "Send a transaction",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		fromUser := args[0]
		toUser := args[1]
		amount, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			log.Fatalf("Invalid amount: %v", err)
		}

		// Load sender wallet
		senderWalletPath := wallet.GetDefaultWalletPath(fromUser)
		senderWallet, err := wallet.LoadFromFile(senderWalletPath)
		if err != nil {
			log.Fatalf("Failed to load sender wallet: %v", err)
		}

		// Load receiver wallet
		receiverWalletPath := wallet.GetDefaultWalletPath(toUser)
		receiverWallet, err := wallet.LoadFromFile(receiverWalletPath)
		if err != nil {
			log.Fatalf("Failed to load receiver wallet: %v", err)
		}

		// Create transaction
		tx := blockchain.NewTransaction(
			senderWallet.Address,
			receiverWallet.Address,
			amount,
		)

		// Sign transaction
		if err := tx.Sign(senderWallet.PrivateKey); err != nil {
			log.Fatalf("Failed to sign transaction: %v", err)
		}

		// Initialize database to save pending transaction
		db, err := storage.NewBlockchainDB(dataDir + "/blockchain.db")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Save transaction as pending
		if err := db.SaveTransaction(tx); err != nil {
			log.Fatalf("Failed to save transaction: %v", err)
		}

		fmt.Printf("Transaction created and saved as pending\n")
		fmt.Printf("From: %s (%s)\n", fromUser, senderWallet.GetAddressString())
		fmt.Printf("To: %s (%s)\n", toUser, receiverWallet.GetAddressString())
		fmt.Printf("Amount: %.2f\n", amount)
		fmt.Printf("Transaction Hash: %x\n", tx.Hash())
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show blockchain status",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize database
		db, err := storage.NewBlockchainDB(dataDir + "/blockchain.db")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Get chain info
		info, err := db.GetChainInfo()
		if err != nil {
			log.Fatalf("Failed to get chain info: %v", err)
		}

		fmt.Println("Blockchain Status:")
		fmt.Printf("Latest Height: %v\n", info["latest_height"])
		if hash, ok := info["latest_block_hash"]; ok {
			fmt.Printf("Latest Block Hash: %v\n", hash)
		}
		if timestamp, ok := info["latest_block_timestamp"]; ok {
			fmt.Printf("Latest Block Timestamp: %v\n", timestamp)
		}
		if pending, ok := info["pending_transactions"]; ok {
			fmt.Printf("Pending Transactions: %v\n", pending)
		}
	},
}

var listBlocksCmd = &cobra.Command{
	Use:   "list-blocks",
	Short: "List all blocks",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize database
		db, err := storage.NewBlockchainDB(dataDir + "/blockchain.db")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Get latest height
		height, err := db.GetLatestHeight()
		if err != nil {
			log.Fatalf("Failed to get latest height: %v", err)
		}

		if height < 0 {
			fmt.Println("No blocks found")
			return
		}

		fmt.Printf("Blockchain has %d blocks:\n\n", height+1)

		// List all blocks
		for i := 0; i <= height; i++ {
			block, err := db.GetBlockByIndex(i)
			if err != nil {
				log.Printf("Failed to get block %d: %v", i, err)
				continue
			}

			fmt.Printf("Block #%d:\n", block.Index)
			fmt.Printf("  Hash: %x\n", block.Hash())
			fmt.Printf("  Previous Hash: %x\n", block.PreviousHash)
			fmt.Printf("  Timestamp: %d\n", block.Timestamp)
			fmt.Printf("  Transactions: %d\n", len(block.Transactions))
			fmt.Printf("  Merkle Root: %x\n", block.MerkleRoot)

			if len(block.Transactions) > 0 {
				fmt.Println("  Transaction Details:")
				for j, tx := range block.Transactions {
					fmt.Printf("    TX #%d:\n", j+1)
					fmt.Printf("      From: %x\n", tx.Sender)
					fmt.Printf("      To: %x\n", tx.Receiver)
					fmt.Printf("      Amount: %.2f\n", tx.Amount)
					fmt.Printf("      Timestamp: %d\n", tx.Timestamp)
					fmt.Printf("      Hash: %x\n", tx.Hash())
				}
			}
			fmt.Println()
		}
	},
}

var getBlockCmd = &cobra.Command{
	Use:   "get-block [index]",
	Short: "Get a specific block by index",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		index, err := strconv.Atoi(args[0])
		if err != nil {
			log.Fatalf("Invalid block index: %v", err)
		}

		// Initialize database
		db, err := storage.NewBlockchainDB(dataDir + "/blockchain.db")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Get block
		block, err := db.GetBlockByIndex(index)
		if err != nil {
			log.Fatalf("Failed to get block: %v", err)
		}

		// Convert to JSON for pretty printing
		blockJSON, err := json.MarshalIndent(block, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal block: %v", err)
		}

		fmt.Printf("Block #%d:\n%s\n", index, string(blockJSON))
	},
}

var listWalletsCmd = &cobra.Command{
	Use:   "list-wallets",
	Short: "List all wallets",
	Run: func(cmd *cobra.Command, args []string) {
		walletDir := "wallets"

		// Read wallet directory
		files, err := os.ReadDir(walletDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No wallets found")
				return
			}
			log.Fatalf("Failed to read wallet directory: %v", err)
		}

		fmt.Println("Available wallets:")
		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Load wallet
			walletPath := walletDir + "/" + file.Name()
			userWallet, err := wallet.LoadFromFile(walletPath)
			if err != nil {
				log.Printf("Failed to load wallet %s: %v", file.Name(), err)
				continue
			}

			fmt.Printf("  %s:\n", userWallet.Name)
			fmt.Printf("    Address: %s\n", userWallet.GetAddressString())
			fmt.Printf("    File: %s\n", walletPath)
		}
	},
}

var validateChainCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the blockchain",
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize database
		db, err := storage.NewBlockchainDB(dataDir + "/blockchain.db")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

		// Validate chain
		if err := db.ValidateChain(); err != nil {
			fmt.Printf("Blockchain validation failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Blockchain validation successful!")
	},
}

func init() {
	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "./data", "Data directory path")
	rootCmd.PersistentFlags().StringVar(&nodeAddr, "node", "localhost:50051", "Node address")

	// Add commands
	rootCmd.AddCommand(createUserCmd)
	rootCmd.AddCommand(sendTxCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listBlocksCmd)
	rootCmd.AddCommand(getBlockCmd)
	rootCmd.AddCommand(listWalletsCmd)
	rootCmd.AddCommand(validateChainCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
