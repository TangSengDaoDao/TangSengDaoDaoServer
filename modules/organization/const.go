package organization

// 员工角色
const (
	// OrgEmployeeNormal 普通员工
	OrgEmployeeNormal = 0
	// OrgEmployeeSuperAdmin 超级管理员
	OrgEmployeeSuperAdmin = 1
	// OrgEmployeeAdmin 子管理员
	OrgEmployeeAdmin = 2
)

var WorkforceTypes = []string{
	"Regular", "Intern", "Outsourcing", "Contractor", "Consultant",
}
