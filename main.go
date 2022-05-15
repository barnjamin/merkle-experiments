package main

import (
	"bytes"
	"context"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"log"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/types"
)

var (
	ErrRootMismatch = errors.New("root Mismatch")

	algodAddress = "https://mainnet-api.algonode.cloud"
	algodToken   = ""

	merkleArrayDs = "MA"
	merkleLeafDs  = "TL"
	hashSize      = 32
)

type layerItem struct {
	pos  uint64
	hash []byte
}

func main() {

	// Create an algod client
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		fmt.Printf("failed to make algod client: %s\n", err)
		return
	}

	// Hardcode for now
	round := uint64(20998353)

	// Fetch block
	block, err := algodClient.Block(round).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get block: %+v", err)
	}

	// Iterate over payset
	for _, txn := range block.Payset[len(block.Payset)-10:] {
		// Get txid to get proof for
		txid := GetTxIdString(txn, block.GenesisHash, block.GenesisID)
		response, err := algodClient.GetProof(round, txid).Do(context.Background())
		if err != nil {
			log.Fatalf("Failed to get proof: %+v", err)
		}

		// Get the same txid but as bytes
		tid := GetTxIdBytes(txn, block.GenesisHash, block.GenesisID)

		// compute the hash for the txn in the merkle tree
		merkleHash := GetMerkleHash(tid, response)

		// Check that the path matches
		if err = Verify(block.TxnRoot[:], merkleHash, response); err != nil {
			log.Fatalf("Failzore: %+v", err)
		}

		// sweet
		log.Printf("Verfied: %s", txid)
	}
}

func NewHasher() hash.Hash {
	//TODO: add more hashers
	return sha512.New512_256()
}

func Verify(root []byte, hash []byte, proof models.ProofResponse) error {
	layer := layerItem{hash: hash, pos: proof.Idx}

	// Break up concat'd path
	var path = make([][]byte, proof.Treedepth)
	for x := 0; x < int(proof.Treedepth); x++ {
		path[x] = proof.Proof[x*hashSize : (x+1)*hashSize]
	}

	// While we have elements in the path, do hash and move up tree
	for len(path) > 0 {
		layer = NextLayer(layer, path[0])
		path = path[1:]
	}

	// Check that the pos is 0 and hash is equal to root
	if layer.pos != 0 || !bytes.Equal(layer.hash, root) {
		return ErrRootMismatch
	}

	return nil
}

func NextLayer(li layerItem, siblingHash []byte) layerItem {
	var (
		left, right []byte
	)

	// Determine which side each hash belongs
	if li.pos&1 == 0 {
		left = li.hash
		right = siblingHash
	} else {
		left = siblingHash
		right = li.hash
	}

	// Move up one layer and compute new hash
	return layerItem{
		pos:  li.pos / 2,
		hash: GetMerkleArrayHash(left, right),
	}
}

// GetMerkleHash returns the hash that is used in the merkle tree base leaf
// It is the combination of the TxId and the SignedTransactionInBlock hash
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

// GetMerkleArrayHash returns the has for a merkle tree layer
// It is the combination of both left and right elements of a branch
func GetMerkleArrayHash(left, right []byte) []byte {
	// Domain Separator Length
	dsLen := len(merkleArrayDs)
	// Buffer to hold stuff to hash
	buf := make([]byte, dsLen+2*hashSize)
	// Add domain sep
	copy(buf[:], []byte(merkleArrayDs))
	// Add left hash
	copy(buf[dsLen:], left[:])
	// Add right hash
	copy(buf[dsLen+len(left):], right[:])

	// Hash'em
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
