package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/channel"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
)

const (
	channelID = "mychannel"
	ccName    = "hrcc"
	operator  = "web-admin"
)

//go:embed config.yaml
var configYAML []byte

//go:embed static
var staticFS embed.FS

var sdk *fabsdk.FabricSDK
var cc *channel.Client

func main() {
	var err error
	// 使用内嵌配置（写入临时文件以便 SDK 读取）
	cfgPath, err := writeTempConfig()
	if err != nil {
		log.Fatalf("写入配置失败: %v", err)
	}
	sdk, err = fabsdk.New(config.FromFile(cfgPath))
	if err != nil {
		log.Fatalf("初始化 SDK 失败: %v", err)
	}
	defer sdk.Close()

	ctxProvider := sdk.ChannelContext(channelID, fabsdk.WithOrg("Org1"), fabsdk.WithUser("Admin"))
	cc, err = channel.New(ctxProvider)
	if err != nil {
		log.Fatalf("创建通道客户端失败: %v", err)
	}

	mux := http.NewServeMux()
	// API 路由
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/departments", handleDepartments)
	mux.HandleFunc("/api/employees", handleEmployees)
	mux.HandleFunc("/api/employee", handleEmployee)
	mux.HandleFunc("/api/history", handleHistory)

	// 静态页面
	staticSub, _ := fs.Sub(staticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(staticSub)))

	addr := ":8080"
	log.Printf("人力资源区块链管理系统已启动: http://localhost%s", addr)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

// ---------- 查询/执行封装 ----------

func queryChaincode(fcn string, args []string, target string) ([]byte, error) {
	bargs := toBytes(args)
	opts := []channel.RequestOption{channel.WithTargetEndpoints("peer0.org1.example.com")}
	_ = target
	resp, err := cc.Query(channel.Request{ChaincodeID: ccName, Fcn: fcn, Args: bargs}, opts...)
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func executeChaincode(fcn string, args []string) ([]byte, error) {
	resp, err := cc.Execute(channel.Request{
		ChaincodeID: ccName,
		Fcn:         fcn,
		Args:        toBytes(args),
	}, channel.WithTargetEndpoints("peer0.org1.example.com", "peer0.org2.example.com"))
	if err != nil {
		return nil, err
	}
	return resp.Payload, nil
}

func toBytes(args []string) [][]byte {
	b := make([][]byte, len(args))
	for i, a := range args {
		b[i] = []byte(a)
	}
	return b
}

// ---------- 处理器 ----------

func handleStats(w http.ResponseWriter, r *http.Request) {
	data, err := queryChaincode("GetStats", nil, "")
	writeJSON(w, data, err)
}

func handleDepartments(w http.ResponseWriter, r *http.Request) {
	data, err := queryChaincode("QueryDepartments", nil, "")
	writeJSON(w, data, err)
}

func handleEmployees(w http.ResponseWriter, r *http.Request) {
	dept := r.URL.Query().Get("department")
	if dept != "" {
		data, err := queryChaincode("QueryByDepartment", []string{dept}, "")
		writeJSON(w, data, err)
		return
	}
	data, err := queryChaincode("QueryAllEmployees", nil, "")
	writeJSON(w, data, err)
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, nil, fmt.Errorf("缺少员工编号 id"))
		return
	}
	data, err := queryChaincode("GetEmployeeHistory", []string{id}, "")
	writeJSON(w, data, err)
}

type employeeForm struct {
	EmployeeID string  `json:"employeeId"`
	Name       string  `json:"name"`
	Gender     string  `json:"gender"`
	Department string  `json:"department"`
	Position   string  `json:"position"`
	Phone      string  `json:"phone"`
	Salary     float64 `json:"salary"`
	HireDate   string  `json:"hireDate"`
	Status     string  `json:"status"`
}

func handleEmployee(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		id := r.URL.Query().Get("id")
		data, err := queryChaincode("QueryEmployee", []string{id}, "")
		writeJSON(w, data, err)

	case http.MethodPost:
		// 若无 id 字段且为新增；body 中带特殊 action 区分
		action := r.URL.Query().Get("action")
		var f employeeForm
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeJSON(w, nil, err)
			return
		}
		switch action {
		case "status":
			data, err := executeChaincode("UpdateEmployeeStatus", []string{f.EmployeeID, f.Status, operator})
			writeJSON(w, data, err)
		case "salary":
			data, err := executeChaincode("UpdateEmployeeSalary", []string{f.EmployeeID, strconv.FormatFloat(f.Salary, 'f', 2, 64), operator})
			writeJSON(w, data, err)
		default:
			// 新增员工
			data, err := executeChaincode("CreateEmployee", []string{
				f.EmployeeID, f.Name, f.Gender, f.Department, f.Position, f.Phone,
				strconv.FormatFloat(f.Salary, 'f', 2, 64), f.HireDate, f.Status, operator,
			})
			writeJSON(w, data, err)
		}

	case http.MethodPut:
		var f employeeForm
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeJSON(w, nil, err)
			return
		}
		data, err := executeChaincode("UpdateEmployee", []string{
			f.EmployeeID, f.Name, f.Gender, f.Department, f.Position, f.Phone,
			strconv.FormatFloat(f.Salary, 'f', 2, 64), f.HireDate, f.Status, operator,
		})
		writeJSON(w, data, err)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		data, err := executeChaincode("DeleteEmployee", []string{id})
		writeJSON(w, data, err)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ---------- 工具 ----------

func writeJSON(w http.ResponseWriter, data []byte, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		b, _ := json.Marshal(map[string]string{"error": err.Error()})
		w.Write(b)
		return
	}
	if data == nil {
		w.Write([]byte("null"))
		return
	}
	w.Write(data)
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.ServeHTTP(w, r)
	})
}
