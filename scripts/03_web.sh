#!/bin/bash
# ============================================================
# 03_web.sh —— 构建并启动人力资源 Web 管理系统（监听 :8080）
# ============================================================
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
export GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct

cd ~/hr-project/webapp
go build -o webapp .

# 停止旧实例（进程名为 webapp，cmdline 为 ./webapp）
pkill -x webapp 2>/dev/null || true
sleep 2
nohup ./webapp > /tmp/webapp.log 2>&1 &
sleep 3
echo "Web 系统已启动，访问地址: http://localhost:8080"
curl -s --max-time 30 http://localhost:8080/api/stats && echo "" || echo "（启动后请稍候再访问）"
