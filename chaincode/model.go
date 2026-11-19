package main

// Employee 人力资源员工信息数据模型
type Employee struct {
	DocType    string  `json:"docType"`    // 固定为 employee，用于复合键区分
	EmployeeID string  `json:"employeeId"` // 员工编号（业务主键，唯一）
	Name       string  `json:"name"`       // 姓名
	Gender     string  `json:"gender"`     // 性别
	Department string  `json:"department"` // 所属部门
	Position   string  `json:"position"`   // 职位
	Phone      string  `json:"phone"`      // 联系电话
	Salary     float64 `json:"salary"`     // 薪资
	HireDate   string  `json:"hireDate"`   // 入职日期 2006-01-02
	Status     string  `json:"status"`     // 在职状态：在职 / 离职 / 休假
	UpdatedAt  string  `json:"updatedAt"`  // 最近更新时间
	UpdatedBy  string  `json:"updatedBy"`  // 最近更新人
}

// 常量
const (
	EmployeeDocType = "employee"
	EmployeeIndex   = "employee"           // 员工主复合键： employee~id
	DepartmentIndex = "departmentEmployee" // 部门索引复合键： departmentEmployee~dept~id

	StatusActive  = "在职"
	StatusLeave   = "休假"
	StatusResigned = "离职"
)

// HistoryRecord 员工信息历史溯源记录
type HistoryRecord struct {
	TxID      string    `json:"txId"`
	Timestamp int64     `json:"timestamp"` // unix 秒
	Time      string    `json:"time"`      // 可读时间
	IsDelete  bool      `json:"isDelete"`
	Employee  *Employee `json:"employee"` // 删除时为 nil
}

// DepartmentStat 部门人数统计
type DepartmentStat struct {
	Department string `json:"department"`
	Count      int    `json:"count"`
}

// StatsOverview 统计概览
type StatsOverview struct {
	Total             int               `json:"total"`             // 总人数（含所有状态）
	ActiveCount       int               `json:"activeCount"`       // 在职
	LeaveCount        int               `json:"leaveCount"`        // 休假
	ResignedCount     int               `json:"resignedCount"`     // 离职
	TotalSalary       float64           `json:"totalSalary"`       // 在职员工薪资合计
	AverageSalary     float64           `json:"averageSalary"`     // 在职员工平均薪资
	DepartmentStats   []DepartmentStat  `json:"departmentStats"`   // 各部门人数
}
