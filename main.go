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
	if len(os.Args) != 2 {
		fmt.Fprint(os.Stderr, "platon-tracer <rpc-url> <start-block-number>\n")
		os.Exit(0)
	}

	var startBlock *big.Int
	var valid bool

	rpcURL := os.Args[1]
	client, err := jsonrpc.NewClient(rpcURL)
	if err != nil {
		panic(err)
	}

	if len(os.Args) > 2 {
		startBlock, valid = new(big.Int).SetString(os.Args[2], 10)
		if !valid {
			fmt.Fprintf(os.Stderr, "start block number %s must be a valid number\n", os.Args[2])
			os.Exit(0)
		}
	} else {
		curNum, err := client.Eth().BlockNumber()
		if err != nil {
			panic(err)
		}
		startBlock = new(big.Int).SetUint64(curNum)
	}
	fmt.Println("start block", startBlock)

	number := startBlock.Int64()
	for {
		begin := time.Now()
		traces, err := client.Debug().TraceBlockByNumber(ethgo.BlockNumber(number), jsonrpc.TraceTransactionOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "execution timeout") || strings.Contains(err.Error(), "request timed out") {
				fmt.Printf("Trace Block #%d %v, retrying\n", number, err)
				continue
			}
			panic(err)
		}

		if number%500 == 0 {
			fmt.Printf("Trace block #%d, Txs: %d, duration: %s\n", number, len(traces), time.Since(begin))
		}
		if len(traces) > 0 {
			fmt.Printf("Trace block #%d, Txs: %d, duration: %s\n", number, len(traces), time.Since(begin))
			printTraceTx(client, traces)
		} else {
			time.Sleep(20 * time.Millisecond)
		}
		number++
	}
}

func printTraceTx(client *jsonrpc.Client, blockTrace []*jsonrpc.BlockTrace) {
	for i, txTrace := range blockTrace {
		if txTrace.Error != "" {
			panic(txTrace.Error)
		}
		receipt, err := client.Eth().GetTransactionReceipt(txTrace.TxHash)
		if err != nil {
			panic(err)
		}
		res := txTrace.Result
		fmt.Printf("  Tx #%d %s: \n\tGas: %d\n\tReturnValue: %s\n\tLogs: %d\n", i, txTrace.TxHash, res.Gas, res.ReturnValue, len(res.StructLogs))
		if receipt.GasUsed != res.Gas {
			fmt.Printf("  Tx #%d %s: invalid gas used(receipt: %d, trace: %d)\n", i, txTrace.TxHash, receipt.GasUsed, res.Gas)
			panic("invalid gas used")
		}
	}
}

func traceTx(client *jsonrpc.Client, txIdx int, receipt *ethgo.Receipt, hash ethgo.Hash) {
	for {
		res, err := client.Debug().TraceTransaction(hash, jsonrpc.TraceTransactionOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "execution timeout") || strings.Contains(err.Error(), "request timed out") {
				fmt.Printf("Trace Tx %s %v, retrying\n", hash, err)
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
