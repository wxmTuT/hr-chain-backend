package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// HRContract 人力资源智能合约
type HRContract struct {
	contractapi.Contract
}

// ===================== 辅助函数 =====================

// employeeKey 员工主复合键： employee~employeeId
func employeeKey(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	return ctx.GetStub().CreateCompositeKey(EmployeeIndex, []string{id})
}

// departmentKey 部门索引复合键： departmentEmployee~department~employeeId
func departmentKey(ctx contractapi.TransactionContextInterface, dept, id string) (string, error) {
	return ctx.GetStub().CreateCompositeKey(DepartmentIndex, []string{dept, id})
}

// nowText 当前时间（可读）
func nowText() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// getEmployee 读取单个员工，不存在返回 nil, nil
func getEmployee(ctx contractapi.TransactionContextInterface, id string) (*Employee, error) {
	key, err := employeeKey(ctx, id)
	if err != nil {
		return nil, err
	}
	bytes, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, err
	}
	if bytes == nil {
		return nil, nil
	}
	var e Employee
	if err := json.Unmarshal(bytes, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// getAllEmployees 遍历员工主复合键，返回全部员工
func getAllEmployees(ctx contractapi.TransactionContextInterface) ([]Employee, error) {
	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(EmployeeIndex, []string{})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var result []Employee
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}
		var e Employee
		if err := json.Unmarshal(kv.Value, &e); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

// toJSON 序列化辅助
func toJSON(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ===================== 初始化账本 =====================

// InitLedger 初始化一批示例员工数据
func (c *HRContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	seeds := []Employee{
		{EmployeeID: "E001", Name: "张伟", Gender: "男", Department: "研发部", Position: "高级工程师", Phone: "13800000001", Salary: 18000, HireDate: "2021-03-15", Status: StatusActive, UpdatedBy: "system"},
		{EmployeeID: "E002", Name: "李娜", Gender: "女", Department: "研发部", Position: "测试工程师", Phone: "13800000002", Salary: 13000, HireDate: "2022-07-01", Status: StatusActive, UpdatedBy: "system"},
		{EmployeeID: "E003", Name: "王强", Gender: "男", Department: "市场部", Position: "市场经理", Phone: "13800000003", Salary: 15000, HireDate: "2020-11-20", Status: StatusActive, UpdatedBy: "system"},
		{EmployeeID: "E004", Name: "刘洋", Gender: "女", Department: "人力资源部", Position: "HRBP", Phone: "13800000004", Salary: 11000, HireDate: "2023-02-10", Status: StatusActive, UpdatedBy: "system"},
		{EmployeeID: "E005", Name: "陈静", Gender: "女", Department: "财务部", Position: "会计", Phone: "13800000005", Salary: 10000, HireDate: "2019-09-01", Status: StatusLeave, UpdatedBy: "system"},
		{EmployeeID: "E006", Name: "赵磊", Gender: "男", Department: "研发部", Position: "架构师", Phone: "13800000006", Salary: 26000, HireDate: "2018-05-18", Status: StatusActive, UpdatedBy: "system"},
		{EmployeeID: "E007", Name: "孙敏", Gender: "女", Department: "市场部", Position: "市场专员", Phone: "13800000007", Salary: 9000, HireDate: "2023-06-25", Status: StatusActive, UpdatedBy: "system"},
		{EmployeeID: "E008", Name: "周涛", Gender: "男", Department: "财务部", Position: "财务主管", Phone: "13800000008", Salary: 16000, HireDate: "2017-12-03", Status: StatusResigned, UpdatedBy: "system"},
	}
	for i := range seeds {
		exist, err := getEmployee(ctx, seeds[i].EmployeeID)
		if err != nil {
			return err
		}
		if exist != nil {
			continue
		}
		if err := c.putEmployee(ctx, &seeds[i]); err != nil {
			return err
		}
	}
	return nil
}

// ===================== 增 =====================

// CreateEmployee 新增员工
func (c *HRContract) CreateEmployee(ctx contractapi.TransactionContextInterface, id, name, gender, department, position, phone string, salary float64, hireDate, status, operator string) error {
	if id == "" || name == "" || department == "" {
		return fmt.Errorf("员工编号、姓名、部门不能为空")
	}
	exist, err := getEmployee(ctx, id)
	if err != nil {
		return err
	}
	if exist != nil {
		return fmt.Errorf("员工 %s 已存在", id)
	}
	if status == "" {
		status = StatusActive
	}
	e := Employee{
		DocType: EmployeeDocType, EmployeeID: id, Name: name, Gender: gender,
		Department: department, Position: position, Phone: phone, Salary: salary,
		HireDate: hireDate, Status: status, UpdatedAt: nowText(), UpdatedBy: operator,
	}
	return c.putEmployee(ctx, &e)
}

// putEmployee 写入员工主数据与部门索引（内部使用）
func (c *HRContract) putEmployee(ctx contractapi.TransactionContextInterface, e *Employee) error {
	e.DocType = EmployeeDocType
	if e.UpdatedAt == "" {
		e.UpdatedAt = nowText()
	}
	key, err := employeeKey(ctx, e.EmployeeID)
	if err != nil {
		return err
	}
	empBytes, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(key, empBytes); err != nil {
		return err
	}
	// 部门索引：值存员工编号
	dk, err := departmentKey(ctx, e.Department, e.EmployeeID)
	if err != nil {
		return err
	}
	return ctx.GetStub().PutState(dk, []byte(e.EmployeeID))
}

// ===================== 改 =====================

// UpdateEmployee 更新员工完整信息（部门变更时同步迁移部门索引）
func (c *HRContract) UpdateEmployee(ctx contractapi.TransactionContextInterface, id, name, gender, department, position, phone string, salary float64, hireDate, status, operator string) error {
	old, err := getEmployee(ctx, id)
	if err != nil {
		return err
	}
	if old == nil {
		return fmt.Errorf("员工 %s 不存在，无法更新", id)
	}
	if department == "" {
		department = old.Department
	}
	// 部门发生变化，删除旧部门索引
	if old.Department != department {
		oldDk, err := departmentKey(ctx, old.Department, id)
		if err != nil {
			return err
		}
		if err := ctx.GetStub().DelState(oldDk); err != nil {
			return err
		}
	}
	updated := Employee{
		DocType: EmployeeDocType, EmployeeID: id, Name: name, Gender: gender,
		Department: department, Position: position, Phone: phone, Salary: salary,
		HireDate: hireDate, Status: status, UpdatedAt: nowText(), UpdatedBy: operator,
	}
	return c.putEmployee(ctx, &updated)
}

// UpdateEmployeeStatus 仅更新员工在职状态
func (c *HRContract) UpdateEmployeeStatus(ctx contractapi.TransactionContextInterface, id, status, operator string) error {
	e, err := getEmployee(ctx, id)
	if err != nil {
		return err
	}
	if e == nil {
		return fmt.Errorf("员工 %s 不存在", id)
	}
	e.Status = status
	e.UpdatedAt = nowText()
	e.UpdatedBy = operator
	return c.putEmployee(ctx, e)
}

// UpdateEmployeeSalary 仅调整员工薪资
func (c *HRContract) UpdateEmployeeSalary(ctx contractapi.TransactionContextInterface, id string, salary float64, operator string) error {
	e, err := getEmployee(ctx, id)
	if err != nil {
		return err
	}
	if e == nil {
		return fmt.Errorf("员工 %s 不存在", id)
	}
	e.Salary = salary
	e.UpdatedAt = nowText()
	e.UpdatedBy = operator
	return c.putEmployee(ctx, e)
}

// ===================== 删 =====================

// DeleteEmployee 删除员工（同时清理主数据与部门索引）
func (c *HRContract) DeleteEmployee(ctx contractapi.TransactionContextInterface, id string) error {
	e, err := getEmployee(ctx, id)
	if err != nil {
		return err
	}
	if e == nil {
		return fmt.Errorf("员工 %s 不存在", id)
	}
	key, err := employeeKey(ctx, id)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().DelState(key); err != nil {
		return err
	}
	dk, err := departmentKey(ctx, e.Department, id)
	if err != nil {
		return err
	}
	return ctx.GetStub().DelState(dk)
}

// ===================== 查 =====================

// QueryEmployee 按员工编号查询
func (c *HRContract) QueryEmployee(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	e, err := getEmployee(ctx, id)
	if err != nil {
		return "", err
	}
	if e == nil {
		return "", fmt.Errorf("员工 %s 不存在", id)
	}
	return toJSON(e)
}

// QueryAllEmployees 查询全部员工（进阶功能：员工列表）
func (c *HRContract) QueryAllEmployees(ctx contractapi.TransactionContextInterface) (string, error) {
	employees, err := getAllEmployees(ctx)
	if err != nil {
		return "", err
	}
	if employees == nil {
		employees = []Employee{}
	}
	return toJSON(employees)
}

// QueryByDepartment 按部门查询员工（进阶功能）
func (c *HRContract) QueryByDepartment(ctx contractapi.TransactionContextInterface, department string) (string, error) {
	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(DepartmentIndex, []string{department})
	if err != nil {
		return "", err
	}
	defer iter.Close()

	var result []Employee
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return "", err
		}
		id := string(kv.Value)
		e, err := getEmployee(ctx, id)
		if err != nil {
			return "", err
		}
		if e != nil {
			result = append(result, *e)
		}
	}
	if result == nil {
		result = []Employee{}
	}
	return toJSON(result)
}

// QueryDepartments 查询所有部门（去重）
func (c *HRContract) QueryDepartments(ctx contractapi.TransactionContextInterface) (string, error) {
	employees, err := getAllEmployees(ctx)
	if err != nil {
		return "", err
	}
	seen := map[string]bool{}
	depts := []string{}
	for _, e := range employees {
		if !seen[e.Department] {
			seen[e.Department] = true
			depts = append(depts, e.Department)
		}
	}
	return toJSON(depts)
}

// ===================== 历史溯源 =====================

// GetEmployeeHistory 查询员工信息的历史变更记录（进阶功能：历史溯源）
func (c *HRContract) GetEmployeeHistory(ctx contractapi.TransactionContextInterface, id string) (string, error) {
	key, err := employeeKey(ctx, id)
	if err != nil {
		return "", err
	}
	iter, err := ctx.GetStub().GetHistoryForKey(key)
	if err != nil {
		return "", err
	}
	defer iter.Close()

	records := []HistoryRecord{}
	for iter.HasNext() {
		mod, err := iter.Next()
		if err != nil {
			return "", err
		}
		rec := HistoryRecord{
			TxID:      mod.TxId,
			Timestamp: mod.Timestamp.Seconds,
			Time:      time.Unix(mod.Timestamp.Seconds, 0).Format("2006-01-02 15:04:05"),
			IsDelete:  mod.IsDelete,
		}
		if !mod.IsDelete && len(mod.Value) > 0 {
			var e Employee
			if err := json.Unmarshal(mod.Value, &e); err != nil {
				return "", err
			}
			rec.Employee = &e
		}
		records = append(records, rec)
	}
	return toJSON(records)
}

// ===================== 统计概览（进阶功能） =====================

// GetStats 统计概览：总人数、各状态人数、部门分布、薪资统计
func (c *HRContract) GetStats(ctx contractapi.TransactionContextInterface) (string, error) {
	employees, err := getAllEmployees(ctx)
	if err != nil {
		return "", err
	}
	stats := StatsOverview{DepartmentStats: []DepartmentStat{}}
	deptCount := map[string]int{}
	var activeSalary float64
	for _, e := range employees {
		stats.Total++
		deptCount[e.Department]++
		switch e.Status {
		case StatusActive:
			stats.ActiveCount++
			activeSalary += e.Salary
		case StatusLeave:
			stats.LeaveCount++
		case StatusResigned:
			stats.ResignedCount++
		}
	}
	stats.TotalSalary = activeSalary
	if stats.ActiveCount > 0 {
		stats.AverageSalary = activeSalary / float64(stats.ActiveCount)
	}
	for dept, cnt := range deptCount {
		stats.DepartmentStats = append(stats.DepartmentStats, DepartmentStat{Department: dept, Count: cnt})
	}
	return toJSON(stats)
}
