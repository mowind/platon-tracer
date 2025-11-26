package main

import (
	"fmt"
	"math/big"
	"os"
	"strings"
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
	begin := time.Now()
	for {
		block, err := client.Eth().GetBlockByNumber(ethgo.BlockNumber(number), false)
		if err != nil {
			panic(err)
		}
		if block == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if (block.Number)%500 == 0 {
			fmt.Printf("Get block #%d, txs: %d, duration: %s\n", block.Number, len(block.TransactionsHashes), time.Since(begin))
			begin = time.Now()
		}
		if len(block.TransactionsHashes) > 0 {
			fmt.Printf("\nGet block #%d, txs: %d, duration: %s\n", block.Number, len(block.TransactionsHashes), time.Since(begin))
			fmt.Printf("Tracing transactions: \n")
			for i, hash := range block.TransactionsHashes {
				receipt, err := client.Eth().GetTransactionReceipt(hash)
				if err != nil {
					panic(err)
				}
				traceTx(client, i, receipt, hash)
			}
			fmt.Println()
		}
		number++
	}
}

func traceTx(client *jsonrpc.Client, txIdx int, receipt *ethgo.Receipt, hash ethgo.Hash) {
	for range 3 {
		res, err := client.Debug().TraceTransaction(hash)
		if err != nil {
			if strings.Contains(err.Error(), "execution timeout") {
				continue
			}
			panic(err)
		}
		fmt.Printf("  Tx #%d %s: \n\tGas: %d\n\tReturnValue: %s\n\tLogs: %d\n", txIdx, hash, res.Gas, res.ReturnValue, len(res.StructLogs))
		if receipt.GasUsed != res.Gas {
			fmt.Printf("  Tx #%d %s: invalid gas used(receipt: %d, trace: %d)\n", txIdx, hash, receipt.GasUsed, res.Gas)
			panic("invalid gas used")
		}
		return
	}
}
