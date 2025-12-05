package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/umbracle/ethgo"
	"github.com/umbracle/ethgo/jsonrpc"
	"github.com/urfave/cli/v3"
)

var (
	traceTxCmd = &cli.Command{
		Name:  "tx",
		Usage: "Trace transaction.",
		Flags: []cli.Flag{
			urlFlag,
			txHashFlag,
			startBlockFlag,
		},
		Action: traceTxMain,
	}

	traceBlockCmd = &cli.Command{
		Name:  "block",
		Usage: "Trace block.",
		Flags: []cli.Flag{
			urlFlag,
			startBlockFlag,
		},
		Action: traceBlockMain,
	}

	urlFlag = &cli.StringFlag{
		Name:  "url",
		Usage: "Node RPC endpoint.",
		Value: "http://127.0.0.1:6789",
	}

	txHashFlag = &cli.StringFlag{
		Name:  "tx-hash",
		Usage: "Specify a transaction for tracking.",
	}

	startBlockFlag = &cli.Int64Flag{
		Name:  "start-block",
		Usage: "Block number which tracking start from. If `tx-hash` specified, ignored this flag.",
		Value: 0,
	}
)

func main() {

	root := &cli.Command{
		Commands: []*cli.Command{
			traceBlockCmd,
			traceTxCmd,
		},
	}

	if err := root.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
	}
}

func traceBlockMain(ctx context.Context, cmd *cli.Command) error {
	url := cmd.String(urlFlag.Name)
	client, err := jsonrpc.NewClient(url)
	if err != nil {
		return err
	}

	startBlock := cmd.Int64(startBlockFlag.Name)
	if startBlock <= 0 {
		curNum, err := client.Eth().BlockNumber()
		if err != nil {
			return err
		}
		startBlock = int64(curNum)
	}

	number := startBlock
	for {
		begin := time.Now()
		traces, err := client.Debug().TraceBlockByNumber(ethgo.BlockNumber(number), jsonrpc.TraceTransactionOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "execution timeout") || strings.Contains(err.Error(), "request timed out") {
				fmt.Printf("Trace Block #%d %v, retrying\n", number, err)
				continue
			}
			return err
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

func traceTxMain(ctx context.Context, cmd *cli.Command) error {
	url := cmd.String(urlFlag.Name)
	client, err := jsonrpc.NewClient(url)
	if err != nil {
		return err
	}

	txHash := cmd.String(txHashFlag.Name)
	if txHash != "" {
		traceTx(client, 0, ethgo.HexToHash(txHash))
		return nil
	}

	startBlock := cmd.Int64(startBlockFlag.Name)
	if startBlock <= 0 {
		curNum, err := client.Eth().BlockNumber()
		if err != nil {
			return err
		}
		startBlock = int64(curNum)
	}

	for {
		begin := time.Now()
		block, err := client.Eth().GetBlockByNumber(ethgo.BlockNumber(startBlock), false)
		if err != nil {
			return err
		}
		if block == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if (block.Number)%500 == 0 {
			fmt.Printf("Get block #%d, txs: %d, duration: %s\n", block.Number, len(block.TransactionsHashes), time.Since(begin))
		}
		if len(block.TransactionsHashes) > 0 {
			fmt.Printf("\nGet block #%d, txs: %d, duration: %s\n", block.Number, len(block.TransactionsHashes), time.Since(begin))
			fmt.Printf("Tracing transactions: \n")
			for i, hash := range block.TransactionsHashes {
				if err := traceTx(client, i, hash); err != nil {
					return err
				}
			}
			fmt.Println()
		}
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

func traceTx(client *jsonrpc.Client, txIdx int, hash ethgo.Hash) error {
	for {
		res, err := client.Debug().TraceTransaction(hash, jsonrpc.TraceTransactionOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "execution timeout") || strings.Contains(err.Error(), "request timed out") {
				fmt.Printf("Trace Tx %s %v, retrying\n", hash, err)
				continue
			}
			return err
		}
		fmt.Printf("  Tx #%d %s: \n\tGas: %d\n\tReturnValue: %s\n\tLogs: %d\n", txIdx, hash, res.Gas, res.ReturnValue, len(res.StructLogs))

		receipt, err := client.Eth().GetTransactionReceipt(hash)
		if err != nil {
			return err
		}
		if receipt.GasUsed != res.Gas {
			fmt.Printf("  Tx #%d %s: invalid gas used(receipt: %d, trace: %d)\n", txIdx, hash, receipt.GasUsed, res.Gas)
			return errors.New("invalid gas used")
		}
		return nil
	}
}
