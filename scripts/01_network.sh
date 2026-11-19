#!/bin/bash
# ============================================================
# 01_network.sh —— 搭建 Hyperledger Fabric v2.2 测试网络并创建应用通道
# 拓扑：1 个 Raft Orderer + Org1(peer0) + Org2(peer0)，TLS 开启
# 通道：mychannel
# 注意：不要设置 COMPOSE_PROJECT_NAME，test-network 的 compose 工程名
#       默认为 docker（compose 文件所在目录名），与官方 network.sh 保持一致
# ============================================================
set -e
export PATH=$PATH:/usr/local/go/bin
TN=~/fabric/fabric-samples/test-network
export PATH=$TN/../bin:$PATH
export FABRIC_CFG_PATH=$TN/../config

cd "$TN"

echo "===== [1/2] 清理旧网络 ====="
./network.sh down

echo "===== [2/2] 启动网络并创建应用通道 mychannel ====="
./network.sh up createChannel -c mychannel

echo "===== 运行中的节点 ====="
docker ps --format '{{.Names}} | {{.Image}} | {{.Status}}'
echo "===== 清理 SDK 本地证书缓存（避免重建网络后使用旧证书） ====="
rm -rf /tmp/hr-store /tmp/hr-web-store /tmp/hr-webapp
echo "===== 网络与通道搭建完成 ====="
