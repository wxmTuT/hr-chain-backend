# HR Chain Backend · 基于联盟链的人力资源存证与管理系统

Go 后端 + Hyperledger Fabric 的学生项目：员工信息、薪资调整、状态变更等人事操作全部写入联盟链，
由 Go Web 服务统一封装链上能力，提供 RESTful 接口与内置管理页面，支持完整变更链路溯源审计。

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
              │                                            │
              └── deploy（SDK 客户端）：链码打包→安装→审批→提交→初始化账本
                                                         │
                                              链码 hrcc（Go）
                              员工增删改查 · 部门索引 · 薪资/状态变更
                              历史溯源（交易ID+时间戳+快照） · 统计下沉
```

- **防篡改存证**：每次员工信息变更都作为一次链上交易，账本自动保留全部历史版本。
- **复合键索引**：员工主键 `employee~id`，部门索引 `departmentEmployee~dept~id`，按部门检索无需全量扫描。
- **统计下沉**：人员分布、薪资汇总在链码内完成，前端一次请求拿结果。
- **溯源审计**：`/api/history?id=` 返回指定员工的完整变更链路（交易标识、时间戳、数据快照）。

## REST API

| 方法与路径 | 功能 | 对应链码函数 |
|---|---|---|
| `GET  /api/stats` | 人员分布与薪资汇总 | `GetStats` |
| `GET  /api/departments` | 部门列表 | `QueryDepartments` |
| `GET  /api/employees[?department=]` | 全部/按部门查询员工 | `QueryAllEmployees` / `QueryByDepartment` |
| `GET  /api/employee?id=` | 查询单个员工 | `QueryEmployee` |
| `POST /api/employee` | 新增员工 | `CreateEmployee` |
| `POST /api/employee?action=status` | 状态变更（入职/离职/调岗） | `UpdateEmployeeStatus` |
| `POST /api/employee?action=salary` | 薪资调整 | `UpdateEmployeeSalary` |
| `PUT  /api/employee` | 修改员工信息 | `UpdateEmployee` |
| `DELETE /api/employee?id=` | 删除员工（历史仍可溯源） | `DeleteEmployee` |
| `GET  /api/history?id=` | 员工变更历史审计 | `GetEmployeeHistory` |
| `GET  /` | 内置管理页面 | — |

所有响应统一 `{ code, msg, data }` 结构，异常集中封装。

## 目录

```
chaincode/   Go 链码（合同、模型、入口）
webapp/      Go Web 服务（REST 接口、SDK 封装、内嵌前端、连接配置）
deploy/      链码自动化部署程序（package → install → approve → commit → init）
scripts/     01_network.sh 起网建通道 / 02_deploy.sh 部署链码 / 03_web.sh 构建并启动 Web
             start.sh 一键全流程 / stop.sh 停止并清网
```

## 快速开始（Ubuntu + Docker + Go）

1. **环境准备**
   - 安装 Docker、docker-compose、Go（≥1.18）与 Fabric 二进制；
   - 克隆 [fabric-samples](https://github.com/hyperledger/fabric-samples)，脚本默认其在 `~/fabric/fabric-samples`；
   - 本项目建议放在 `~/hr-project`（与 `scripts/` 内路径一致，或自行修改脚本）。

2. **修改连接配置**：`webapp/config.yaml` 与 `deploy/config.yaml` 中
   `cryptoconfig.path`、TLS 证书路径改成你的 fabric-samples 实际路径；
   在 `/etc/hosts` 将 `peer0.org1/org2.example.com`、`orderer.example.com` 指向 `127.0.0.1`。

3. **一键启动**

   ```bash
   bash scripts/start.sh        # 组网 → 部署链码 → 启动 Web
   curl http://localhost:8080/api/stats
   ```

   分步执行：`01_network.sh`、`02_deploy.sh`、`03_web.sh`；停止：`scripts/stop.sh`。

4. **常见问题**
   - 链码安装失败：确认两组织的 peer 容器健康、`go mod vendor` 能访问 GOPROXY；
   - 重建网络后接口报证书错误：SDK 会缓存证书，删除 `/tmp/hr-store`、`/tmp/hr-web-store`（`01_network.sh` 已自动清理）；
   - 账本写入异常：检查通道是否已 `createChannel`、链码是否完成 approve/commit。

## 说明

本项目为个人学习与课程实践作品，网络拓扑基于 Fabric 官方 test-network。

## License

MIT
