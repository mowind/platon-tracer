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
		if (block.Number)%500 == 0 {
			fmt.Printf("Get block #%d, txs: %d\n", block.Number, len(block.TransactionsHashes))
		}
		if len(block.TransactionsHashes) > 0 {
			fmt.Printf("\nGet block #%d, txs: %d\n", block.Number, len(block.TransactionsHashes))
			fmt.Printf("Tracing transactions: \n")
			for i, hash := range block.TransactionsHashes {
				res, err := client.Debug().TraceTransaction(hash)
				if err != nil {
					panic(err)
				}
				fmt.Printf("  Tx #%d %s: \n\tGas: %d\n\tReturnValue: %s\n\tLogs: %d\n", i, hash, res.Gas, res.ReturnValue, len(res.StructLogs))
			}
			fmt.Println()
		}
		time.Sleep(10 * time.Millisecond)
		number++
	}
}
