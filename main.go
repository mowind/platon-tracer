package main

import (
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprint(os.Stderr, "platon-tracer <rpc-url> <start-block-number>\n")
		os.Exit(0)
	}

	rpcURL := os.Args[1]
	startBlock, valid := new(big.Int).SetString(os.Args[2], 10)
	if !valid {
		fmt.Fprintf(os.Stderr, "start block number %s must be a valid number\n", os.Args[2])
		os.Exit(0)
	}

	client, err := jsonrpc.NewClient(rpcURL)
	if err != nil {
		panic(err)
	}

	number := startBlock.Int64()
	for {
		block, err := client.Eth().GetBlockByNumber(ethgo.BlockNumber(number), false)
		if err != nil {
			panic(err)
		}
		fmt.Printf("get block #%d, txs: %d\n", block.Number, len(block.Transactions))
		if len(block.Transactions) > 0 {
			for _, tx := range block.Transactions {
				res, err := client.Debug().TraceTransaction(tx.Hash)
				if err != nil {
					panic(err)
				}
				fmt.Printf("Tx %s trace: \n\tGas: %d\n\tReturnValue: %s\n", tx.Hash, res.Gas, res.ReturnValue)
			}
		}
		time.Sleep(200 * time.Millisecond)
		number++
	}
}
