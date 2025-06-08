package p2p

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/blockchain/proto"
)

// Client represents a gRPC client for communicating with other nodes
type Client struct {
	conn   *grpc.ClientConn
	client pb.BlockchainServiceClient
	addr   string
}

// NewClient creates a new P2P client
func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	client := pb.NewBlockchainServiceClient(conn)

	return &Client{
		conn:   conn,
		client: client,
		addr:   addr,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	return c.conn.Close()
}

// SendTransaction sends a transaction to the peer
func (c *Client) SendTransaction(ctx context.Context, tx *pb.Transaction) error {
	req := &pb.SendTransactionRequest{
		Transaction: tx,
	}

	resp, err := c.client.SendTransaction(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send transaction to %s: %w", c.addr, err)
	}

	if !resp.Success {
		return fmt.Errorf("transaction rejected by %s: %s", c.addr, resp.Message)
	}

	log.Printf("Transaction sent successfully to %s", c.addr)
	return nil
}

// ProposeBlock proposes a block to the peer
func (c *Client) ProposeBlock(ctx context.Context, block *pb.Block, proposerID string) error {
	req := &pb.ProposeBlockRequest{
		Block:      block,
		ProposerId: proposerID,
	}

	resp, err := c.client.ProposeBlock(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to propose block to %s: %w", c.addr, err)
	}

	if !resp.Success {
		return fmt.Errorf("block proposal rejected by %s: %s", c.addr, resp.Message)
	}

	log.Printf("Block proposed successfully to %s", c.addr)
	return nil
}

// SendVote sends a vote to the peer
func (c *Client) SendVote(ctx context.Context, vote *pb.Vote) error {
	req := &pb.VoteRequest{
		Vote: vote,
	}

	resp, err := c.client.Vote(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send vote to %s: %w", c.addr, err)
	}

	if !resp.Success {
		return fmt.Errorf("vote rejected by %s: %s", c.addr, resp.Message)
	}

	log.Printf("Vote sent successfully to %s", c.addr)
	return nil
}

// GetLatestBlock gets the latest block from the peer
func (c *Client) GetLatestBlock(ctx context.Context) (*pb.Block, error) {
	req := &pb.GetLatestBlockRequest{}

	resp, err := c.client.GetLatestBlock(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block from %s: %w", c.addr, err)
	}

	if !resp.Found {
		return nil, fmt.Errorf("no blocks found on %s", c.addr)
	}

	return resp.Block, nil
}

// GetBlockByHeight gets a block by height from the peer
func (c *Client) GetBlockByHeight(ctx context.Context, height int32) (*pb.Block, error) {
	req := &pb.GetBlockByHeightRequest{
		Height: height,
	}

	resp, err := c.client.GetBlockByHeight(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get block %d from %s: %w", height, c.addr, err)
	}

	if !resp.Found {
		return nil, fmt.Errorf("block %d not found on %s", height, c.addr)
	}

	return resp.Block, nil
}

// GetBlocksFromHeight gets blocks starting from a specific height
func (c *Client) GetBlocksFromHeight(ctx context.Context, startHeight int32) ([]*pb.Block, error) {
	req := &pb.GetBlocksFromHeightRequest{
		StartHeight: startHeight,
	}

	resp, err := c.client.GetBlocksFromHeight(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocks from height %d from %s: %w", startHeight, c.addr, err)
	}

	return resp.Blocks, nil
}

// GetChainInfo gets chain information from the peer
func (c *Client) GetChainInfo(ctx context.Context) (*pb.GetChainInfoResponse, error) {
	req := &pb.GetChainInfoRequest{}

	resp, err := c.client.GetChainInfo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain info from %s: %w", c.addr, err)
	}

	return resp, nil
}

// Sync synchronizes blockchain data with the peer
func (c *Client) Sync(ctx context.Context, nodeID string, currentHeight int32) ([]*pb.Block, error) {
	req := &pb.SyncRequest{
		NodeId:        nodeID,
		CurrentHeight: currentHeight,
	}

	resp, err := c.client.Sync(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sync with %s: %w", c.addr, err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("sync rejected by %s: %s", c.addr, resp.Message)
	}

	return resp.Blocks, nil
}

// IsHealthy checks if the peer is responsive
func (c *Client) IsHealthy() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.GetChainInfo(ctx)
	return err == nil
}
