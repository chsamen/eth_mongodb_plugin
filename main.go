package main

import (
	"context"
	"eth_mongodb_plugin/config"
	"eth_mongodb_plugin/data"
	"eth_mongodb_plugin/log"
	"eth_mongodb_plugin/mongodb"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
	"time"
)

var logger *zap.SugaredLogger


func main() {
	logger = log.InitLogger()
	defer logger.Sync()
	config.Execute()
	config.EmpApp.EmpSetting()
	mong, err := mongodb.NewCollection(config.EmpApp.MongoDBIp,config.EmpApp.DatabaseName)
	if err != nil {
		logger.Errorf("mongodb connect error: %s", err	)
		panic(err)
	}
   	if config.EmpApp.CreateIndex == true {
		mong.BlockIndex()
		mong.LogIndex()
		mong.ReceiptIndex()
		mong.BlockStateIndex()
	}

	mobileCli, _ := data.NewEthMobile(config.EmpApp.EthIp)
	//获取最新区块号
	blockInfo, _, _, _ := mobileCli.GetBlock(-1)
	//从不可逆区块开始拉取
	blockNumber := blockInfo.Number - 8
	blocks := make(chan int64, 100)
	ctx := context.Background()
	go checkBlock(mong, blockNumber - 1, blocks)
	go pullFromChannel(ctx, mong, mobileCli, blocks)

	reversePull(mong, mobileCli, blockNumber)
}

func pullFromChannel(ctx context.Context, mong *mongodb.AllCollection, mobileCli *data.MobileClient, blocks chan int64) {
	for{
		getNumber, ok := <- blocks
		if ok {
			res := insertBlock(ctx, mong, mobileCli, getNumber)
			fmt.Println(res)
			if res {
				logger.Infof("Channel--Insert new block from channel: %s", getNumber)
			} else {
				logger.Infof("Channel--Have inserted this block: %s", getNumber)
			}
		}
	}
}

func reversePull(mong *mongodb.AllCollection, mobileCli *data.MobileClient, blockNumber int64) {
	ctx := context.Background()
	insertRes := insertBlock(ctx, mong, mobileCli, blockNumber)
	time.Sleep(time.Second)
	fmt.Println(blockNumber)
	if !insertRes {
		logger.Infof("Have inserted this latest block: %s", blockNumber)
	}else {
		logger.Infof("Insert new block: %s", blockNumber)
	}
	blockInfo, _, _, _ := mobileCli.GetBlock(-1)
	reversePull(mong, mobileCli, blockInfo.Number - 8)
}

func insertBlock(ctx context.Context, mong *mongodb.AllCollection, mobileCli *data.MobileClient, blockNumber int64) bool {
	blockInfo, receiptsArr, logsArr, err := mobileCli.GetBlock(blockNumber)
	if err != nil {
		return false
	}
	res, err := mong.BlockStateSearch(ctx, blockNumber)
	info := mongodb.BlockState{}
	bson.Unmarshal(res, &info)
	if err != nil {
		mong.BlockStateInsert(ctx, blockNumber)
		//mong.BlockStateUpdate(ctx, blockNumber, 1)
	}else if info.BlockState == 0 {
		//mong.BlockStateUpdate(ctx, blockNumber, 1)
	}else if info.BlockState == 1 {
		//mong.DeleteBlock(ctx, blockNumber)
	}else if info.BlockState == 2 {
		return false
	}
	mong.BlockInsert(ctx, blockInfo)
	mong.ReceiptsInsert(ctx, receiptsArr)
	mong.LogsInsert(ctx, logsArr)
	mong.BlockStateUpdate(ctx, blockNumber, 2)
	return true
}

func checkBlock(mong *mongodb.AllCollection, blockNumber int64, blocks chan int64){
	for {
		ctx := context.Background()
		if len(blocks) < 100 {
			fmt.Println("channel:",len(blocks))
		}
		if blockNumber == 0 {
			close(blocks)
			break
		} else {
			if len(blocks) < 100 {
				res, err := mong.BlockStateSearch(ctx, blockNumber)
				if err != nil {
					mong.BlockStateInsert(ctx, blockNumber)
					logger.Infof("check--Create new block state: %s", blockNumber)
				} else {
					info := mongodb.BlockState{}
					bson.Unmarshal(res, &info)
					if info.BlockState == 2 {
						logger.Infof("check--Have inserted this block: %s", blockNumber)
						blockNumber--
						continue
					} else if info.BlockState == 1 {
						deleteRes, deleteErr := mong.DeleteBlock(ctx, blockNumber)
						if deleteErr != nil {
							logger.Errorf("check--Delete dirty data: %s", deleteErr)
						} else {
							logger.Infof("check--Delete dirty data: %s", deleteRes)
						}
						mong.BlockStateUpdate(ctx, blockNumber, 0)
					}
				}
				blocks <- blockNumber
				blockNumber--
			}
		}
	}
}