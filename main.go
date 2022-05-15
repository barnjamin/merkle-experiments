package main

import (
	"context"
	"fmt"
	"log"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/types"
)

const (
	algodAddress = "https://mainnet-api.algonode.cloud"
	algodToken   = ""

	merkleLeafDs = "TL"
	round        = uint64(20998353)
)

func main() {

	// Create an algod client
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		fmt.Printf("failed to make algod client: %s\n", err)
		return
	}

	// Fetch block
	block, err := algodClient.Block(round).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get block: %+v", err)
	}

	// Iterate over payset
	for idx, txn := range block.Payset[len(block.Payset)-10:] {
		// Get txid to get proof for
		txid := GetTxIdString(txn, block.GenesisHash, block.GenesisID)
		response, err := algodClient.GetProof(round, txid).Do(context.Background())
		if err != nil {
			log.Fatalf("Failed to get proof: %+v", err)
		}

		// Get the same txid but as bytes
		tid := GetTxIdBytes(txn, block.GenesisHash, block.GenesisID)

		// compute the hash for the merkle tree
		merkleHash := GetMerkleHash(tid, response)

		// Check that the path matches
		if err = Verify(block.TxnRoot[:], merkleHash, response); err != nil {
			log.Fatalf("Failzore: %+v", err)
		}

		// sweet
		log.Printf("Verfied: %d", idx)
	}
}

func GetMerkleHash(txid []byte, proof models.ProofResponse) []byte {
	//Domain separator length
	dsLen := len(merkleLeafDs)
	// Buffer to hold the stuff we need to hash
	buf := make([]byte, 64+dsLen)
	// Domain sep first
	copy(buf[:], merkleLeafDs[:])
	// Add txid after domain sep
	copy(buf[dsLen:], txid[:])
	// Add stibhash after domain sep & txid
	copy(buf[dsLen+32:], proof.Stibhash[:])

	// Write out the hash
	nh := NewHasher()
	nh.Write(buf)
	return nh.Sum(nil)
}

func GetTxIdString(stib types.SignedTxnInBlock, gh types.Digest, gid string) string {
	stib.Txn.GenesisHash = gh
	stib.Txn.GenesisID = gid
	return crypto.GetTxID(stib.Txn)
}

func GetTxIdBytes(stib types.SignedTxnInBlock, gh types.Digest, gid string) []byte {
	stib.Txn.GenesisHash = gh
	stib.Txn.GenesisID = gid
	return crypto.TransactionID(stib.Txn)
}
