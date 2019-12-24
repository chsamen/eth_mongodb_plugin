package data

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"log"
	"strconv"
)

// Receipt represents the results of a transaction.
type ReceiptInfo struct {
	// Consensus fields: These fields are defined by the Yellow Paper
	PostState         string "bson:`root`"
	Status            uint64 "bson:`status`"
	CumulativeGasUsed uint64 "bson:`cumulativeGasUsed`"
	Bloom             string "bson:`logsBloom`"
	Logs              []LogInfo "bson:`logs`"

	// Implementation fields: These fields are added by geth when processing a transaction.
	// They are stored in the chain database.
	TxHash          string "bson:`transactionHash`"
	ContractAddress string "bson:`contractAddress`"
	GasUsed         uint64 "bson:`gasUsed`"

	// Inclusion information: These fields provide information about the inclusion of the
	// transaction corresponding to this receipt.
	BlockHash        string "bson:`blockHash`"
	BlockNumber      int64 "bson:`blockNumber`"
	TransactionIndex uint "bson:`transactionIndex`"
}

// Log represents a contract log event. These events are generated by the LOG opcode and
// stored/indexed by the node.
type LogInfo struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address string "bson:`address`"
	// list of topics provided by the contract.
	Topics []string "bson:`topics`"
	// supplied by the contract, usually ABI-encoded
	Data string "bson:`data`"

	// Derived fields. These fields are filled in by the node
	// but not secured by consensus.
	// block in which the transaction was included
	BlockNumber uint64 "bson:`blockNumber`"
	// hash of the transaction
	TxHash string "bson:`txHash`"
	// index of the transaction in the block
	TxIndex uint "bson:`txIndex`"
	// hash of the block in which the transaction was included
	BlockHash string "bson:`blockHash`"
	// index of the log in the block
	Index uint "bson:`blockHash`"

	// The Removed field is true if this log was reverted due to a chain reorganisation.
	// You must pay attention to this field if you receive logs through a filter query.
	Removed bool "bson:`blockHash`"
}

type EthClient struct {
	cli *ethclient.Client
}

// NewEthereumClient connects a client to the given URL.
func newEthClient(ethIp string) (c *EthClient, _ error) {
	client, err := ethclient.Dial(ethIp)
	if err != nil {
		log.Fatal(err)
	}
	return &EthClient{client}, err
}

func (c *EthClient)getReceipt(repHash string, re chan *ReceiptInfo) {
	receiptsInfo := c.GetReceiptByTxHash(repHash)
	re <- receiptsInfo
}

//获取Logs
func (c *EthClient)GetReceiptByTxHash(hashHex string) (receiptInfo *ReceiptInfo)  {
	r, err := c.cli.TransactionReceipt(context.Background(), common.HexToHash(hashHex))
	if err!= nil {
		fmt.Println(err)
	}
	var enc ReceiptInfo
	enc.PostState = common.BytesToHash(r.PostState).String()
	enc.Status = r.Status
	enc.CumulativeGasUsed = r.CumulativeGasUsed
	enc.Bloom = common.BytesToHash(r.Bloom.Bytes()).String()
	enc.TxHash = r.TxHash.String()
	enc.ContractAddress = r.ContractAddress.String()
	enc.GasUsed = r.GasUsed
	enc.BlockHash = r.BlockHash.String()
	enc.BlockNumber, _ = strconv.ParseInt(r.BlockNumber.String(), 10, 64)
	enc.TransactionIndex = r.TransactionIndex
	var logs = make([]LogInfo, 0)
	for i := 0; i < len(r.Logs) ; i++ {
		var logInfo LogInfo
		l := r.Logs[i]
		logInfo.BlockNumber = l.BlockNumber
		logInfo.BlockHash = l.BlockHash.String()
		logInfo.Data = common.BytesToHash(l.Data).String()
		logInfo.Address = l.Address.String()
		logInfo.Index = l.Index
		logInfo.TxIndex = l.TxIndex
		logInfo.Removed = l.Removed
		var topicArr = make([]string, 0)
		for k := 0; k<len(l.Topics) ; k++ {
			topicArr = append(topicArr, l.Topics[k].String())
		}
		logInfo.Topics = topicArr
		logs = append(logs, logInfo)
	}
	enc.Logs = logs
	return &enc
}