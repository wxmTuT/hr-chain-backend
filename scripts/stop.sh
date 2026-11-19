#!/bin/bash
# stop.sh —— 停止 Web 应用并关闭 Fabric 网络
echo "===== 停止 Web 应用 ====="
pkill -x webapp 2>/dev/null || true
echo "===== 关闭 Fabric 网络 ====="
TN=~/fabric/fabric-samples/test-network
export PATH=$TN/../bin:$PATH
export FABRIC_CFG_PATH=$TN/../config
cd "$TN"
./network.sh down
echo "===== 已停止 ====="
