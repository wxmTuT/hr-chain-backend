# HR Chain Backend · 基于联盟链的人力资源存证与管理系统

Go 后端 + Hyperledger Fabric 的学生项目：员工信息、薪资调整、状态变更等人事操作全部写入联盟链，由 Go Web 服务统一封装链上能力，提供 RESTful 接口与内置管理页面，支持完整变更链路溯源审计。

![Go](https://img.shields.io/badge/Go-1.18%2B-00ADD8?logo=go&logoColor=white)
![Fabric](https://img.shields.io/badge/Hyperledger%20Fabric-2.2-2F3134)
![Docker](https://img.shields.io/badge/Docker-compose-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)

## 技术栈

| 层 | 技术 |
|---|---|
| 区块链网络 | Hyperledger Fabric v2.2（test-network，双组织 Org1/Org2，TLS，Raft 排序，通道 `mychannel`） |
| 链码 | Go，`fabric-contract-api-go`，复合键索引 + 历史查询（`GetHistoryForKey`） |
| 后端服务 | Go 原生 `net/http` + `fabric-sdk-go`（包 / 安装 / 审批 / 提交 / 初始化全自动） |
| 前端 | 原生 HTML/JS/CSS（`embed` 内嵌进二进制，单文件部署） |
| 部署 | Docker + Shell 脚本，一键起网、部署链码、启动服务 |

## 系统结构

```
浏览器 ──▶ Go Web 服务 (:8080) ──▶ fabric-sdk-go ──▶ peer0.org1 / peer0.org2
                │
                └── deploy（SDK 客户端）：链码打包→安装→审批→提交→初始化账本
                        │
                        链码 hrcc（Go）
                        员工增删改查 · 部门索引 · 薪资/状态变更
                        历史溯源（交易ID+时间戳+快照） · 统计下沉
```

## 核心设计

- **防篡改存证**：每次员工信息变更都作为一次链上交易，账本自动保留全部历史版本，链下无法直接改写。
- **复合键索引**：员工主键 `employee~id`，部门索引 `departmentEmployee~dept~id`，按部门检索无需全量扫描。
- **统计下沉**：人员分布、薪资汇总在链码内完成，前端一次请求拿结果，减少网络往返。
- **溯源审计**：`/api/history?id=` 返回指定员工的完整变更链路（交易标识、时间戳、数据快照）。

## REST API

| 方法与路径 | 功能 | 对应链码函数 |
|---|---|---|
| `GET /api/stats` | 人员分布与薪资汇总 | `GetStats` |
| `GET /api/departments` | 部门列表 | `QueryDepartments` |
| `GET /api/employees[?department=]` | 全部/按部门查询员工 | `QueryAllEmployees` / `QueryByDepartment` |
| `GET /api/employee?id=` | 查询单个员工 | `QueryEmployee` |
| `POST /api/employee` | 新增员工 | `CreateEmployee` |
| `POST /api/employee?action=status` | 状态变更（入职/离职/调岗） | `UpdateEmployeeStatus` |
| `POST /api/employee?action=salary` | 薪资调整 | `UpdateEmployeeSalary` |
| `PUT /api/employee` | 修改员工信息 | `UpdateEmployee` |
| `DELETE /api/employee?id=` | 删除员工（历史仍可溯源） | `DeleteEmployee` |
| `GET /api/history?id=` | 员工变更历史审计 | `GetEmployeeHistory` |
| `GET /` | 内置管理页面 | — |

所有响应统一 `{ code, msg, data }` 结构，异常集中封装。

## 目录结构

```
chaincode/   Go 链码（合同、模型、入口）
webapp/      Go Web 服务（REST 接口、SDK 封装、内嵌前端、连接配置）
deploy/      链码自动化部署程序（package → install → approve → commit → init）
scripts/     01_network.sh 起网建通道 / 02_deploy.sh 部署链码 / 03_web.sh 构建并启动 Web
             start.sh 一键全流程 / stop.sh 停止并清网
```

## 快速开始

环境要求：Ubuntu 20.04+、Docker 与 docker-compose、Go ≥ 1.18、Fabric 2.2 二进制。

```bash
# 1. 准备 fabric-samples（脚本默认路径 ~/fabric/fabric-samples）
git clone https://github.com/hyperledger/fabric-samples.git ~/fabric/fabric-samples
cd ~/fabric/fabric-samples && ./install-fabric.sh binary docker

# 2. 放置本项目到 ~/hr-project（与 scripts/ 内路径保持一致，或自行修改脚本）

# 3. 一键启动：起网建通道 → 部署链码 → 构建并启动 Web 服务
cd ~/hr-project/scripts
./start.sh
```

启动完成后访问 `http://localhost:8080/` 打开管理页面。分步执行时可单独运行 `01_network.sh`、`02_deploy.sh`、`03_web.sh`；停止并清理网络执行 `./stop.sh`。

## 接口验证

```bash
# 人事统计
curl http://localhost:8080/api/stats

# 新增员工
curl -X POST http://localhost:8080/api/employee \
  -H 'Content-Type: application/json' \
  -d '{"id":"E1001","name":"张三","department":"研发部","position":"后端工程师","salary":12000}'

# 薪资调整
curl -X POST 'http://localhost:8080/api/employee?action=salary' \
  -H 'Content-Type: application/json' \
  -d '{"id":"E1001","salary":13500}'

# 变更历史溯源
curl 'http://localhost:8080/api/history?id=E1001'
```

## 常见问题

| 问题 | 排查方向 |
|---|---|
| 起网失败、容器反复重启 | 确认 Docker 正常、`fabric-samples` 路径与脚本一致、端口 7050/7051 未被占用 |
| 链码安装失败 | 检查 `chaincode/` 是否可 `go build`、Go 版本与 vendor 依赖、镜像内是否已有编译产物 |
| 审批/提交失败 | 双组织都要 approve，序列号（sequence）需与上一次提交保持一致递增 |
| Web 服务连接 peer 报 TLS 错误 | 确认证书路径指向 `organizations/peerOrganizations/...`，容器内外地址不同需改连接配置 |
| 账本写入异常 | 查看 peer 日志 `docker logs peer0.org1.example.com`，重点看背书失败与 MVCC 冲突 |

## 安全说明

仓库只保留源码与脚本，**不包含任何证书、私钥与 msp 材料**。`crypto-config/`、`*.pem`、`*.key`、`organizations/` 下的身份文件均已通过 `.gitignore` 排除，运行环境本地生成即可，切勿提交到公开仓库。

## License

MIT
