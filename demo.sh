#!/bin/bash

# Blockchain Demo Script
# This script demonstrates the basic functionality of the blockchain system

set -e

echo "🔗 Blockchain System Demo"
echo "=========================="

# Build the CLI
echo "📦 Building blockchain CLI..."
go build -o blockchain-cli ./cmd/cli

echo ""
echo "👥 Creating users (Alice and Bob)..."

# Create Alice's wallet
echo "Creating Alice's wallet..."
./blockchain-cli create-user alice

echo ""

# Create Bob's wallet  
echo "Creating Bob's wallet..."
./blockchain-cli create-user bob

echo ""
echo "💼 Listing all wallets..."
./blockchain-cli list-wallets

echo ""
echo "📊 Checking initial blockchain status..."
./blockchain-cli status

echo ""
echo "💸 Creating a transaction: Alice sends 10.5 coins to Bob..."
./blockchain-cli send alice bob 10.5

echo ""
echo "💸 Creating another transaction: Alice sends 5.0 coins to Bob..."
./blockchain-cli send alice bob 5.0

echo ""
echo "💸 Creating third transaction: Bob sends 2.5 coins to Alice..."
./blockchain-cli send bob alice 2.5

echo ""
echo "📊 Checking blockchain status after transactions..."
./blockchain-cli status

echo ""
echo "🔍 Listing all blocks in the blockchain..."
./blockchain-cli list-blocks

echo ""
echo "🔐 Validating blockchain integrity..."
./blockchain-cli validate

echo ""
echo "✅ Demo completed successfully!"
echo ""
echo "📝 Summary:"
echo "- Created wallets for Alice and Bob with ECDSA key pairs"
echo "- Generated 3 signed transactions"
echo "- All transactions are stored as pending (waiting for leader to create blocks)"
echo "- Blockchain structure and integrity verified"
echo ""
echo "🚀 To run with full consensus (3 nodes):"
echo "   docker-compose up -d"
echo ""
echo "📖 Check the README.md for more detailed instructions." 