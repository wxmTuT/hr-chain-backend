package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

// 部署参数
const (
	ChannelID       = "mychannel"
	CCName          = "hrcc"
	CCVersion       = "1.0"
	CCLabel         = "hrcc_1.0"
	Sequence  int64 = 1
)

// 组织与其 peer 端点（config.yaml 中定义的名字）
type orgTarget struct {
	orgName string
	peer    string
}

var orgs = []orgTarget{
	{"Org1", "peer0.org1.example.com"},
	{"Org2", "peer0.org2.example.com"},
}

func main() {
	home, _ := os.UserHomeDir()
	projectRoot := filepath.Join(home, "hr-project")
	ccPath := filepath.Join(projectRoot, "chaincode")
	configPath := filepath.Join(projectRoot, "deploy", "config.yaml")
	peerBin := filepath.Join(home, "fabric", "fabric-samples", "bin", "peer")
	pkgFile := filepath.Join(projectRoot, "deploy", "hrcc.tar.gz")

	// ============ 步骤 1：打包（Package） ============
	log("【步骤1】打包链码 -> %s", pkgFile)
	if err := packageChaincode(peerBin, ccPath, pkgFile); err != nil {
		fatal("打包失败: %v", err)
	}
	pkgBytes, err := os.ReadFile(pkgFile)
	if err != nil {
		fatal("读取链码包失败: %v", err)
	}
	log("链码包大小: %d 字节", len(pkgBytes))

	// ============ 初始化 SDK ============
	sdk, err := fabsdk.New(config.FromFile(configPath))
	if err != nil {
		fatal("初始化 SDK 失败: %v", err)
	}
	defer sdk.Close()

	// ============ 步骤 2：安装（Install）到两个组织 ============
	packageIDs := map[string]string{}
	for _, o := range orgs {
		log("【步骤2】在 %s 的 %s 上安装链码", o.orgName, o.peer)
		rc, err := resmgmt.New(sdk.Context(fabsdk.WithOrg(o.orgName), fabsdk.WithUser("Admin")))
		if err != nil {
			fatal("%s 创建资源管理客户端失败: %v", o.orgName, err)
		}

		// 已安装则直接查询 packageID，保证脚本可重复执行
		installed, err := rc.LifecycleQueryInstalledCC(resmgmt.WithTargetEndpoints(o.peer))
		if err != nil {
			fatal("%s 查询已安装链码失败: %v", o.orgName, err)
		}
		pid := findPackageID(installed, CCLabel)
		if pid == "" {
			resp, err := rc.LifecycleInstallCC(
				resmgmt.LifecycleInstallCCRequest{Label: CCLabel, Package: pkgBytes},
				resmgmt.WithTargetEndpoints(o.peer),
			)
			if err != nil {
				fatal("%s 安装链码失败: %v", o.orgName, err)
			}
			for _, r := range resp {
				if r.PackageID != "" {
					pid = r.PackageID
				}
			}
			if pid == "" {
				installed, _ = rc.LifecycleQueryInstalledCC(resmgmt.WithTargetEndpoints(o.peer))
				pid = findPackageID(installed, CCLabel)
			}
		} else {
			log("%s 链码已安装，packageID=%s", o.orgName, pid)
		}
		if pid == "" {
			fatal("%s 安装后未能获取 packageID", o.orgName)
		}
		packageIDs[o.orgName] = pid
		log("%s 安装完成，packageID=%s", o.orgName, pid)
	}

	// 两个组织的 packageID 应一致（标签+代码哈希）
	packageID := packageIDs["Org1"]
	log("使用 packageID: %s", packageID)

	// ============ 步骤 3：组织审批（Approve） ============
	approveReq := resmgmt.LifecycleApproveCCRequest{
		Name:      CCName,
		Version:   CCVersion,
		PackageID: packageID,
		Sequence:  Sequence,
	}
	for _, o := range orgs {
		log("【步骤3】%s 审批链码定义", o.orgName)
		rc, err := resmgmt.New(sdk.Context(fabsdk.WithOrg(o.orgName), fabsdk.WithUser("Admin")))
		if err != nil {
			fatal("%s 创建通道资源客户端失败: %v", o.orgName, err)
		}
		// 已审批则跳过
		approved, err := rc.LifecycleQueryApprovedCC(ChannelID,
			resmgmt.LifecycleQueryApprovedCCRequest{Name: CCName, Sequence: Sequence},
			resmgmt.WithTargetEndpoints(o.peer))
		if err == nil && approved.PackageID == packageID && approved.Version == CCVersion {
			log("%s 已审批该定义，跳过", o.orgName)
			continue
		}
		txID, err := rc.LifecycleApproveCC(ChannelID, approveReq, resmgmt.WithTargetEndpoints(o.peer))
		if err != nil {
			fatal("%s 审批失败: %v", o.orgName, err)
		}
		log("%s 审批成功，交易ID=%s", o.orgName, txID)
		time.Sleep(3 * time.Second)
	}

	// ============ 检查提交就绪状态 ============
	rc1, err := resmgmt.New(sdk.Context(fabsdk.WithOrg("Org1"), fabsdk.WithUser("Admin")))
	if err != nil {
		fatal("创建 Org1 通道客户端失败: %v", err)
	}
	readiness, err := rc1.LifecycleCheckCCCommitReadiness(ChannelID,
		resmgmt.LifecycleCheckCCCommitReadinessRequest{
			Name: CCName, Version: CCVersion, Sequence: Sequence,
		},
		resmgmt.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
	if err != nil {
		log("⚠ 检查提交就绪状态失败（继续）: %v", err)
	} else {
		log("提交就绪审批情况: %v", readiness.Approvals)
	}

	// ============ 步骤 4：提交（Commit） ============
	committed, err := rc1.LifecycleQueryCommittedCC(ChannelID,
		resmgmt.LifecycleQueryCommittedCCRequest{Name: CCName},
		resmgmt.WithTargetEndpoints("peer0.org1.example.com"))
	alreadyCommitted := false
	if err == nil {
		for _, d := range committed {
			if d.Name == CCName && d.Version == CCVersion && d.Sequence == Sequence {
				alreadyCommitted = true
			}
		}
	}
	if alreadyCommitted {
		log("【步骤4】链码 %s:%s 已提交，跳过提交", CCName, CCVersion)
	} else {
		log("【步骤4】提交链码定义到通道 %s", ChannelID)
		txID, err := rc1.LifecycleCommitCC(ChannelID,
			resmgmt.LifecycleCommitCCRequest{Name: CCName, Version: CCVersion, Sequence: Sequence},
			resmgmt.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
		if err != nil {
			fatal("提交失败: %v", err)
		}
		log("提交成功，交易ID=%s", txID)
		time.Sleep(5 * time.Second)
	}

	// ============ 步骤 5：初始化账本并校验 ============
	log("【步骤5】初始化示例数据并查询校验")
	cc, err := channel.New(sdk.ChannelContext(ChannelID, fabsdk.WithOrg("Org1"), fabsdk.WithUser("Admin")))
	if err != nil {
		fatal("创建通道客户端失败: %v", err)
	}

	// InitLedger（首次调用会触发链码容器构建，可能需要较长时间）
	if _, err := cc.Execute(channel.Request{
		ChaincodeID: CCName,
		Fcn:         "InitLedger",
		Args:        [][]byte{},
	}, channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com")); err != nil {
		log("⚠ InitLedger 调用返回（可能数据已存在或容器构建中）: %v", err)
	}

	// 轮询查询，等待链码容器就绪
	var result []byte
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		resp, qerr := cc.Query(channel.Request{
			ChaincodeID: CCName,
			Fcn:         "QueryAllEmployees",
			Args:        [][]byte{},
		}, channel.WithTargetEndpoints("peer0.org1.example.com"))
		if qerr == nil {
			result = resp.Payload
			break
		}
		log("链码尚未就绪，10s 后重试... (%v)", qerr)
		time.Sleep(10 * time.Second)
	}
	if result == nil {
		fatal("链码部署后查询超时，请检查链码容器日志")
	}
	log("✔ 部署成功！QueryAllEmployees 返回:")
	fmt.Println(string(result))
}

// packageChaincode 调用 peer 二进制完成链码生命周期打包
func packageChaincode(peerBin, ccPath, outFile string) error {
	// 若已存在先删除，保证重新打包
	_ = os.Remove(outFile)
	// core.yaml 所在目录（fabric-samples/config）
	home, _ := os.UserHomeDir()
	cfgDir := filepath.Join(home, "fabric", "fabric-samples", "config")
	cmd := exec.Command(peerBin, "lifecycle", "chaincode", "package", outFile,
		"--path", ccPath, "--lang", "golang", "--label", CCLabel)
	cmd.Env = append(os.Environ(),
		"FABRIC_CFG_PATH="+cfgDir,
		"GO111MODULE=on",
	)
	cmd.Dir = ccPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v\n%s", err, string(out))
	}
	return nil
}

func findPackageID(installed []resmgmt.LifecycleInstalledCC, label string) string {
	for _, cc := range installed {
		if cc.Label == label {
			return cc.PackageID
		}
	}
	return ""
}

func log(format string, args ...interface{}) {
	fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}
func fatal(format string, args ...interface{}) {
	fmt.Printf("[错误] %s\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}
