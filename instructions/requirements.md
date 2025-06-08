# Blockchain System Requirements

## General Requirements

Build a simple blockchain system simulating monetary transactions between two users: **Alice** and **Bob**. The system must ensure the following properties:

### 1. Digital Signature of Transactions using ECDSA

Ensuring the integrity and non-repudiation of transactions is core to blockchain technology.

  * **Each user (Alice, Bob)** will have their own key pair (private/public) using the **ECDSA** (Elliptic Curve Digital Signature Algorithm). **Important Note**: The original problem statement mentioned ECDH, but for digitally signing transactions, **ECDSA** is the appropriate algorithm. ECDH is used for secure key exchange. Ensure you use **ECDSA** for signing and verifying transaction signatures.
  * A **transaction** includes:
      * `Sender`: Public key (or address derived from it) of the sender.
      * `Receiver`: Public key (or address derived from it) of the receiver.
      * `Amount`: The amount of currency being transferred.
      * `Timestamp`: The time the transaction was created.
      * `Signature`: The digital signature of the transaction, created by the sender's **private key**.
  * The system must **verify the signature** upon receiving a transaction to ensure the sender is the true owner of the funds and that the transaction has not been altered.

### 2. Data Storage on LevelDB, Validation with Merkle Tree

Each validator node needs to maintain a consistent and reliable copy of the blockchain.

  * Each validator node will store **blocks** in **LevelDB**. LevelDB is a simple and efficient local key-value database.
  * Each **block** includes:
      * `Transaction List`: A set of valid transactions included in this block.
      * `Merkle Tree Root`: The root hash of the Merkle tree constructed from the list of transactions.
      * `Previous Block Hash`: Links the current block to the preceding block, forming the chain.
      * `Current Block Hash`: Hash of the entire block (including all the fields above).
  * **Merkle Tree Implementation**: This is a crucial part for efficiently verifying the integrity of transactions within a block. Ensure that the **Merkle Tree root correctly reflects the content of all transactions in the block**, and any minor change in a transaction will alter the Merkle Root.

### 3. Consensus Mechanism among 3 Validator Nodes (running on Docker)

The blockchain requires a mechanism for nodes in the network to agree on the common state of the ledger.

  * **Initialize 3 validator nodes** running on Docker. Each node will be an independent service.
  * **Leader-Follower Mechanism**: One node will act as the **Leader** (can be randomly elected, the first node to start, or statically configured). The Leader is responsible for:
      * Creating new blocks from pending transactions.
      * Sending the proposed block to the other nodes (Followers).
  * **Voting Mechanism**: Followers, after receiving the proposed block from the Leader, will:
      * Validate the block's legitimacy (check transaction signatures, Merkle Root, PreviousBlockHash, etc.).
      * Send back a vote (accept or reject) to the Leader.
  * **Consensus**: After achieving a sufficient number of votes (e.g., **majority vote**, meaning 2/3 of the total nodes, which is 2 out of 3 nodes in this case), the block is considered agreed upon.
  * **Commit Block**: After consensus, the Leader notifies all nodes to **commit** (save) the block to their local LevelDB.

### 4. Node Validator Recovery Mechanism upon Disconnection

The system must be fault-tolerant and capable of self-recovery.

  * If a validator node is shut down/loses connection (e.g., Docker container stopped):
      * Upon restart, the node must automatically **reconnect** with other nodes in the network (e.g., via a pre-configured list of peer nodes or a service discovery mechanism).
      * The node needs to **download missed blocks** from the other nodes. This can be done via an API/gRPC endpoint allowing the node to request blocks from a specific `blockHeight` onwards.
      * After **full synchronization** (catching up with the longest and most valid blockchain), the node resumes participation in the consensus process and transaction handling.

-----

## Submission Requirements

For a full evaluation, you need to provide:

  * **Complete Golang Source Code**: The source code must be clean, readable, well-structured, and adhere to Golang programming principles.
  * **`docker-compose.yml` File**: To easily and quickly launch the 3 validators. This file should include network configuration, ports, and necessary environment variables for each node.
  * **CLI Script or REST API**:
      * Create users (Alice, Bob): Allow creation of ECDSA key pairs and their secure storage (e.g., in a simple JSON file or a small database).
      * Send transaction from Alice to Bob: Allow users to create and send a transaction (signed by Alice) to one of the validator nodes.
      * Check node/block/consensus status: API or CLI to check the current blockchain length, view the content of a specific block, or the status of nodes in the consensus process.
  * **System Operation Guide and Architecture Description**: A detailed `README.md` file explaining how to set up, run, and interact with the system. Clearly state the main architectural decisions, libraries used, and reasons for choosing them.

-----

## Suggestions / Detailed Guidance

To help you start and complete the task effectively:

### 1. Golang Project Structure

  * **Organize code by package**:
      * `pkg/blockchain`: Contains definitions for `Block`, `Transaction`, hashing logic, Merkle Tree.
      * `pkg/wallet`: Contains logic for creating and managing ECDSA key pairs, signing, and verifying signatures.
      * `pkg/p2p`: Handles communication between nodes (gRPC or HTTP), including transaction propagation, block proposal/voting.
      * `pkg/consensus`: Implements the consensus mechanism (e.g., Leader election, voting).
      * `pkg/storage`: Interacts with LevelDB.
      * `cmd/cli`: Handles CLI commands.
      * `cmd/node`: Entry point for each validator node.
  * **Use Go Modules**: Ensure effective dependency management.

### 2. ECDSA Implementation

  * **Key Pair Generation**: Use `crypto/elliptic` (e.g., `elliptic.P256()`) and `crypto/ecdsa`.
    ```go
    import (
        "crypto/ecdsa"
        "crypto/elliptic"
        "crypto/rand"
        "fmt"
    )

    func GenerateKeyPair() (*ecdsa.PrivateKey, error) {
        privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
        if err != nil {
            return nil, fmt.Errorf("failed to generate key pair: %w", err)
        }
        return privKey, nil
    }

    func PublicKeyToAddress(pubKey *ecdsa.PublicKey) []byte {
        // Convert public key to an address format (e.g., hash of the public key)
        // Ensure the address is unique and fixed
        // TODO: Implement this
        return nil
    }
    ```
  * **Sign Transaction**: Use `ecdsa.Sign()`. You need to hash the transaction content (sender, receiver, amount, timestamp) before signing.
    ```go
    import (
        "crypto/sha256"
        "encoding/json"
        "fmt"
        "math/big"
    )

    type Transaction struct {
        Sender    []byte // Public Key or Address
        Receiver  []byte // Public Key or Address
        Amount    float64
        Timestamp int64
        Signature []byte // R and S concatenated
    }

    func (t *Transaction) Hash() []byte {
        // Create a hashable representation of the transaction
        txCopy := *t
        txCopy.Signature = nil // Exclude signature from hash
        data, _ := json.Marshal(txCopy) // Error handling for Marshal can be added
        hash := sha256.Sum256(data)
        return hash[:]
    }

    func SignTransaction(tx *Transaction, privKey *ecdsa.PrivateKey) error {
        txHash := tx.Hash()
        r, s, err := ecdsa.Sign(rand.Reader, privKey, txHash)
        if err != nil {
            return fmt.Errorf("failed to sign transaction: %w", err)
        }
        // Store R and S as a concatenated byte slice
        tx.Signature = append(r.Bytes(), s.Bytes()...)
        return nil
    }
    ```
  * **Verify Signature**: Use `ecdsa.Verify()`. Need to separate R and S from the signature.
    ```go
    func VerifyTransaction(tx *Transaction, pubKey *ecdsa.PublicKey) bool {
        txHash := tx.Hash()
        // Assume signature is R and S concatenated, parse them back to big.Int
        // It's crucial that the length used for splitting is correct.
        // Consider storing R and S lengths or using a more robust serialization format.
        sigLen := len(tx.Signature)
        if sigLen == 0 { // Or some other check for an empty/invalid signature
            return false
        }
        rBytes := tx.Signature[:sigLen/2]
        sBytes := tx.Signature[sigLen/2:]

        r := new(big.Int).SetBytes(rBytes)
        s := new(big.Int).SetBytes(sBytes)
        return ecdsa.Verify(pubKey, txHash, r, s)
    }
    ```

### 3. Merkle Tree

  * **Input**: List of hashes of each transaction in the block.
  * **Construction Process**: Pair up hashes, hash them together, and repeat until a single hash (Merkle Root) remains. If the number of hashes is odd, duplicate the last hash.
  * **Library**: You can implement it yourself or find open-source Golang libraries for Merkle Trees (e.g., `github.com/cbergoon/merkletree`). However, self-implementation will demonstrate your deep understanding.

### 4. LevelDB

  * **Key-value store**: LevelDB stores data as key-value byte pairs.
  * **Store Block**: Key can be `block_hash` or `block_height`. Value is the byte representation of the block.
    ```go
    import (
        "encoding/json"
        "github.com/syndtr/goleveldb/leveldb"
        // "github.com/your_project/pkg/blockchain" // Assuming Block struct is defined elsewhere
    )

    // Assuming Block struct is defined, e.g.:
    // type Block struct {
    //     // ... fields ...
    //     CurrentBlockHash []byte // Or string
    // }
    // func (b *Block) Hash() []byte { return b.CurrentBlockHash }


    func SaveBlock(db *leveldb.DB, block *Block) error { // Assuming Block has a Hash() method returning []byte
        blockBytes, err := json.Marshal(block)
        if err != nil {
            return err
        }
        // block.Hash() should be a method of the Block struct returning its hash
        // For example, if block.CurrentBlockHash is the hash:
        // return db.Put(block.CurrentBlockHash, blockBytes, nil)
        // Or if you have a generic Hash() method on the block:
        // return db.Put(block.Hash(), blockBytes, nil)
        // This placeholder needs to be replaced with the actual key.
        return db.Put([]byte("placeholder_block_hash_key"), blockBytes, nil) // FIXME: Use actual block hash
    }

    func GetBlock(db *leveldb.DB, hash []byte) (*Block, error) { // Assuming Block is the correct type
        blockBytes, err := db.Get(hash, nil)
        if err != nil {
            return nil, err
        }
        var block Block // Assuming Block is the correct type
        if err := json.Unmarshal(blockBytes, &block); err != nil {
            return nil, err
        }
        return &block, nil
    }
    ```
  * **Error Handling**: Always check for errors when interacting with LevelDB.

### 5. Inter-Node Communication (P2P)

  * **gRPC**: An excellent choice for high-performance communication between services in Docker. You will define `.proto` files for messages and services like:
      * `SendTransaction(Transaction)`
      * `ProposeBlock(Block)`
      * `Vote(VoteMessage)` // Renamed for clarity if Vote is a common noun
      * `GetBlockByHeight(BlockHeightRequest)` // More descriptive
      * `GetLatestBlock(GetLatestBlockRequest)` // More descriptive
  * **HTTP/REST API**: Simpler to start with but potentially less efficient than gRPC for large-scale block synchronization tasks.
  * **Connection**: Each node needs to know the addresses of other peer nodes (can be configured in `docker-compose.yml` or environment variables).

### 6. Simple Consensus Mechanism (Majority Vote)

  * **Leader Election**: Can be as simple as hardcoding (e.g., node 1 is the leader) or using a simple election mechanism on startup.
  * **State Machine**: Each validator node will have a state (e.g., `Leader`, `Follower`, `Syncing`).
  * **Pending Transaction Management**: The Leader collects unconfirmed (pending) transactions.
  * **Periodic Block Creation**: The Leader can create a new block at a fixed interval (e.g., every 5 seconds) or when there are enough transactions.
  * **Vote Processing**: The Leader will receive votes from Followers. After receiving enough approval votes, the block is considered finalized.
  * **Block Propagation**: When a block is finalized, the Leader will notify all nodes to save that block.

### 7. Node Recovery

  * **Polling/Periodic Check**: A disconnected node, upon restarting, can periodically check the status of other nodes in the network.
  * **Get Latest Block**: When reconnected, the node will call an API like `GetLatestBlock()` from other nodes to know the current `blockHeight` of the network.
  * **Syncing**: Then, the node will repeatedly call `GetBlockByHeight(height)` for each block it missed until it reaches the latest `blockHeight`.
  * **Consistency Assurance**: During synchronization, verify the blockchain (previous hash must match, Merkle Root must be valid) to avoid fraudulent chains.

-----

## Suggested Docker Setup

You can start with a simple `docker-compose.yml` like this:

```yaml
version: '3.8'

services:
  node1:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      NODE_ID: node1
      # Configure other peers. Can be passed as environment variables or config file
      PEERS: "node2:50051,node3:50051" # Ensure quotes if there are special characters or spaces
      # If using static Leader
      IS_LEADER: "true" # Only node1 is initially the leader
    ports:
      - "50051:50051" # gRPC/HTTP port for node-to-node communication
      - "8080:8080"   # API/CLI port if exposed
    volumes:
      - node1_data:/app/data # For persistent LevelDB storage
    networks:
      - blockchain_net

  node2:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      NODE_ID: node2
      PEERS: "node1:50051,node3:50051"
      IS_LEADER: "false"
    ports:
      - "50052:50051" # Host port : Container port
    volumes:
      - node2_data:/app/data
    networks:
      - blockchain_net

  node3:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      NODE_ID: node3
      PEERS: "node1:50051,node2:50051"
      IS_LEADER: "false"
    ports:
      - "50053:50051" # Host port : Container port
    volumes:
      - node3_data:/app/data
    networks:
      - blockchain_net

volumes:
  node1_data:
  node2_data:
  node3_data:

networks:
  blockchain_net:
    driver: bridge
```

And a basic `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
# Ensure the main package for the node is in cmd/node/main.go or similar
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go-blockchain ./cmd/node

# Run stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /go-blockchain .
# Create data directory for LevelDB
RUN mkdir -p /app/data
# Expose the port the application will listen on (if not using host mode network in docker-compose)
# EXPOSE 50051
CMD ["./go-blockchain"]
```
