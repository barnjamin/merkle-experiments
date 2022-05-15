package main

import (
	"bytes"
	"context"
	"fmt"
	"log"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/encoding/msgpack"
	"github.com/algorand/go-algorand-sdk/types"
	gcrypto "github.com/algorand/go-algorand/crypto"
	"github.com/algorand/go-algorand/protocol"
)

const (
	algodAddress = "https://mainnet-api.algonode.cloud"
	algodToken   = ""
)

func main() {

	// Create an algod client
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		fmt.Printf("failed to make algod client: %s\n", err)
		return
	}

	round := uint64(20998353)
	block, err := algodClient.Block(round).Do(context.Background())
	if err != nil {
		log.Fatalf("Failed to get block: %+v", err)
	}

	for _, txn := range block.Payset[:1] {

		txid := GetTxIdString(txn, block.GenesisHash, block.GenesisID)
		response, err := algodClient.GetProof(round, txid).Do(context.Background())
		if err != nil {
			log.Fatalf("Failed to get proof: %+v", err)
		}

		proof := ResponseToProof(response)

		log.Printf("Produced same hash?: %t", CheckHash(hashableTxn(txn), proof.HashFactory, response.Stibhash))

		var stibhash gcrypto.Digest
		copy(stibhash[:], response.Stibhash)

		var txidhash gcrypto.Digest
		var txidbytes = GetTxIdBytes(txn, block.GenesisHash, block.GenesisID)
		copy(txidhash[:], txidbytes[:])
		merkleElem := &TxnMerkleElemRaw{Txn: txidhash, Stib: stibhash}

		h := Hash(merkleElem, proof.HashFactory)

		//if err = merklearray.Verify(block.TxnRoot[:], map[uint64]gcrypto.Hashable{response.Idx: merkleElem}, &proof); err != nil {
		//	log.Fatalf("Coundn't verify: %+v", err)
		//} else {
		//	log.Printf("that worked?")
		//}
		if err = Verify(block.TxnRoot[:], h, response); err != nil {
			log.Fatalf("Failzore: %+v", err)
		}
		log.Printf("That worked?")

	}
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
func Hash(h gcrypto.Hashable, hf gcrypto.HashFactory) []byte {
	nh := hf.NewHash()
	prefix, b := h.ToBeHashed()
	nh.Write(append([]byte(string(prefix))[:], b...))
	return nh.Sum(nil)
}

func CheckHash(htxn gcrypto.Hashable, hf gcrypto.HashFactory, hash []byte) bool {
	htxnHash := Hash(htxn, hf)
	return bytes.Equal(htxnHash, hash)
}

type stib struct {
	types.SignedTxnInBlock
}

func (s *stib) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.SignedTxnInBlock, msgpack.Encode(s)
}

func hashableTxn(txn types.SignedTxnInBlock) gcrypto.Hashable {
	return &stib{txn}
}

// TxnMerkleElemRaw this struct helps creates a hashable struct from the bytes
type TxnMerkleElemRaw struct {
	Txn  gcrypto.Digest // txn id
	Stib gcrypto.Digest // hash value of transactions.SignedTxnInBlock
}

func txnMerkleToRaw(txid [gcrypto.DigestSize]byte, stib [gcrypto.DigestSize]byte) (buf []byte) {
	buf = make([]byte, 2*gcrypto.DigestSize)
	copy(buf[:], txid[:])
	copy(buf[gcrypto.DigestSize:], stib[:])
	return
}

// ToBeHashed implements the crypto.Hashable interface.
func (tme *TxnMerkleElemRaw) ToBeHashed() (protocol.HashID, []byte) {
	return protocol.TxnMerkleLeaf, txnMerkleToRaw(tme.Txn, tme.Stib)
}
