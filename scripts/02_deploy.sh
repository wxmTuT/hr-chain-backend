#!/bin/bash
# ============================================================
# 02_deploy.sh —— 用 fabric-sdk-go 自动化部署 HR 链码
# 流程：打包(Package) -> 安装(Install, Org1/Org2) ->
#       审批(Approve, Org1/Org2) -> 提交(Commit) -> 初始化账本
# ============================================================
set -e
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct

cd ~/hr-project/chaincode
go mod tidy >/dev/null 2>&1
go mod vendor >/dev/null 2>&1

cd ~/hr-project/deploy
go build -o deploy .
./deploy
echo "===== 链码部署完成 ====="
