package main

import (
	"fmt"
	"log"

	"github.com/algorand/go-algorand-sdk/client/algod"
)

const (
	algodAddress = "https://testnet-api.algonode.cloud"
	algodToken   = ""
)

func main() {

	// Create an algod client
	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		fmt.Printf("failed to make algod client: %s\n", err)
		return
	}

	block, err := algodClient.BlockRaw(123)
	if err != nil {
		log.Fatalf("Failed to get block: %+v", err)
	}

	log.Printf("%+v", block)
}
