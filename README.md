# Blockchain System

A simple blockchain implementation in Go with ECDSA transaction signing, Merkle Tree validation, LevelDB storage, and leader-follower consensus mechanism.

## Features

- **ECDSA Digital Signatures**: Secure transaction signing using Elliptic Curve Digital Signature Algorithm
- **Merkle Tree**: Efficient transaction validation and integrity verification
- **LevelDB Storage**: Persistent blockchain data storage
- **Leader-Follower Consensus**: Majority voting mechanism among 3 validator nodes
- **Docker Support**: Containerized deployment with Docker Compose
- **CLI Interface**: Command-line tool for blockchain interaction
- **Wallet Management**: ECDSA key pair generation and wallet functionality

## Architecture

### Core Components

1. **Transaction** (`pkg/blockchain/transaction.go`)
   - ECDSA signature generation and verification
   - Transaction hashing and serialization
   - Supports sender, receiver, amount, and timestamp

2. **Block** (`pkg/blockchain/block.go`)
   - Block creation with transaction list
   - Merkle root calculation
   - Block validation and hash generation

3. **Merkle Tree** (`pkg/blockchain/merkle.go`)
   - Binary tree construction from transactions
   - Root hash calculation for block integrity
   - Merkle proof verification

4. **Wallet** (`pkg/wallet/wallet.go`)
   - ECDSA key pair generation (P256 curve)
   - Address derivation from public keys
   - Transaction signing and verification

5. **Storage** (`pkg/storage/leveldb.go`)
   - LevelDB integration for persistent storage
   - Block and transaction management
   - Chain validation and synchronization

6. **Consensus** (`pkg/consensus/consensus.go`)
   - Leader-follower voting mechanism
   - Block proposal and voting
   - 2/3 majority consensus requirement

7. **Node** (`cmd/node/main.go`)
   - Blockchain node implementation
   - gRPC server for inter-node communication
   - Leader election and block production

8. **CLI** (`cmd/cli/main.go`)
   - Command-line interface for blockchain interaction
   - Wallet creation and transaction management
   - Blockchain status and validation

## Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- Git

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd blockchain
```

2. Install dependencies:
```bash
go mod download
```

3. Build the binaries:
```bash
go build -o blockchain-node ./cmd/node
go build -o blockchain-cli ./cmd/cli
```

## Usage

### Running with Docker Compose (Recommended)

1. Start the 3-node blockchain network:
```bash
docker-compose up -d
```

This will start:
- `node1` (Leader) on port 50051
- `node2` (Follower) on port 50052
- `node3` (Follower) on port 50053

2. Check node status:
```bash
docker-compose logs node1
docker-compose logs node2
docker-compose logs node3
```

### Running Locally

1. Start a single node:
```bash
export NODE_ID=node1
export IS_LEADER=true
export LISTEN_ADDR=:50051
export DATA_DIR=./data
./blockchain-node
```

### Using the CLI

#### Create Users (Alice and Bob)

```bash
# Create Alice's wallet
./blockchain-cli create-user alice

# Create Bob's wallet
./blockchain-cli create-user bob

# List all wallets
./blockchain-cli list-wallets
```

#### Send Transactions

```bash
# Send 10 coins from Alice to Bob
./blockchain-cli send alice bob 10.0

# Check blockchain status
./blockchain-cli status

# List all blocks
./blockchain-cli list-blocks

# Get specific block
./blockchain-cli get-block 0
```

#### Blockchain Operations

```bash
# Validate the entire blockchain
./blockchain-cli validate

# Check blockchain status
./blockchain-cli status --data-dir ./data
```

## Configuration

### Environment Variables

- `NODE_ID`: Unique identifier for the node (default: "node1")
- `IS_LEADER`: Whether this node is the leader (default: "false")
- `PEERS`: Comma-separated list of peer addresses (e.g., "node2:50051,node3:50051")
- `LISTEN_ADDR`: Address to listen on (default: ":50051")
- `DATA_DIR`: Directory for blockchain data storage (default: "./data")

### CLI Flags

- `--data-dir`: Specify data directory path (default: "./data")
- `--node`: Node address for communication (default: "localhost:50051")

## System Design

### Consensus Mechanism

The system implements a simple leader-follower consensus:

1. **Leader Election**: One node is designated as leader (configurable)
2. **Block Proposal**: Leader creates blocks from pending transactions
3. **Voting**: Followers validate and vote on proposed blocks
4. **Consensus**: Blocks are committed with 2/3 majority (2 out of 3 nodes)
5. **Synchronization**: All nodes maintain synchronized blockchain state

### Transaction Flow

1. User creates transaction using CLI
2. Transaction is signed with sender's private key
3. Transaction is added to pending pool
4. Leader includes transaction in new block
5. Block is proposed to follower nodes
6. Followers validate and vote on block
7. Block is committed after reaching consensus
8. Transaction is removed from pending pool

### Security Features

- **ECDSA Signatures**: Cryptographic transaction authentication
- **Merkle Trees**: Tamper-evident transaction integrity
- **Hash Chaining**: Immutable block linkage
- **Consensus Validation**: Multi-node agreement requirement

## Development

### Project Structure

```
blockchain/
├── cmd/
│   ├── cli/           # CLI application
│   └── node/          # Node application
├── pkg/
│   ├── blockchain/    # Core blockchain logic
│   ├── consensus/     # Consensus mechanism
│   ├── storage/       # LevelDB integration
│   └── wallet/        # Wallet and cryptography
├── proto/             # gRPC protocol definitions
├── instructions/      # Requirements and documentation
├── docker-compose.yml # Docker deployment configuration
├── Dockerfile         # Container build configuration
├── go.mod            # Go module definition
└── README.md         # This file
```

### Testing

Run tests for individual packages:

```bash
go test ./pkg/blockchain/
go test ./pkg/wallet/
go test ./pkg/storage/
```

### Building for Production

Build optimized binaries:

```bash
CGO_ENABLED=1 go build -ldflags="-w -s" -o blockchain-node ./cmd/node
CGO_ENABLED=1 go build -ldflags="-w -s" -o blockchain-cli ./cmd/cli
```

## Troubleshooting

### Common Issues

1. **Port conflicts**: Ensure ports 50051-50053 are available
2. **Permission errors**: Check file permissions for data directories
3. **Network connectivity**: Verify Docker network configuration
4. **Database locks**: Stop all nodes before restarting

### Debugging

Enable verbose logging:

```bash
export LOG_LEVEL=debug
./blockchain-node
```

Check container logs:

```bash
docker-compose logs -f node1
```

## Limitations

- **Simplified Consensus**: Basic majority voting (production systems need more robust consensus)
- **No Network Security**: Missing TLS/encryption for node communication
- **Limited Scalability**: Designed for 3 nodes (can be extended)
- **No Transaction Fees**: Simplified economic model
- **Basic Wallet**: No advanced wallet features (HD wallets, multi-sig)

## Future Enhancements

- gRPC implementation for full inter-node communication
- Advanced consensus algorithms (PBFT, Raft)
- Transaction pool optimization
- Network security and encryption
- REST API for external integration
- Comprehensive test coverage
- Performance monitoring and metrics

## License

This project is created for educational and demonstration purposes.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## Architecture Decisions

### Why ECDSA?
- Industry standard for blockchain applications
- Smaller key sizes compared to RSA
- Excellent security-to-performance ratio

### Why LevelDB?
- Embedded key-value store (no external dependencies)
- High performance for read-heavy workloads
- Used by Bitcoin Core and other blockchain projects

### Why Leader-Follower?
- Simple to implement and understand
- Suitable for small networks (3 nodes)
- Clear separation of responsibilities

### Why Go?
- Excellent concurrency support
- Strong standard library
- Fast compilation and deployment
- Growing blockchain ecosystem 