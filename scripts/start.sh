#!/bin/bash
# start.sh —— 一键：网络/通道 -> 链码部署 -> Web 启动
set -e
DIR="$(cd "$(dirname "$0")" && pwd)"
bash "$DIR/01_network.sh"
bash "$DIR/02_deploy.sh"
bash "$DIR/03_web.sh"
echo ""
echo "================ 全部启动完成 ================"
echo "Web 系统: http://localhost:8080"
echo "=============================================="
