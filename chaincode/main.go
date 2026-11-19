package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	chaincode, err := contractapi.NewChaincode(&HRContract{})
	if err != nil {
		log.Panicf("创建 HR 链码失败: %v", err)
	}
	if err := chaincode.Start(); err != nil {
		log.Panicf("启动 HR 链码失败: %v", err)
	}
}
