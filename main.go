package main

import (
	"context"
	"fmt"
	"log"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
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

	for _, txn := range block.Payset[10:11] {

		txn.Txn.GenesisHash = block.GenesisHash
		txn.Txn.GenesisID = block.GenesisID

		id := crypto.GetTxID(txn.Txn)
		proof, err := algodClient.GetProof(round, id).Do(context.Background())
		if err != nil {
			log.Fatalf("Failed to get proof: %+v", err)
		}

		//log.Printf("%s => %+v", id, proof)
		log.Printf("%+v", Verify(block.TxnRoot[:], proof.Stibhash, proof))
	}

}
