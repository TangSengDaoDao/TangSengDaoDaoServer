package organization

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	dba "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type db struct {
	session *dbr.Session
	ctx     *config.Context
}

func newDB(ctx *config.Context) *db {
	return &db{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *db) insertOrg(m *orgModel) error {
	_, err := d.session.InsertInto("organization").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *db) insertOrgTx(m *orgModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("organization").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *db) insertOrgEmployee(m *orgEmployeeModel) error {
	_, err := d.session.InsertInto("organization_employee").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}
func (d *db) insertOrgEmployeeTx(m *orgEmployeeModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("organization_employee").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *db) insertDept(m *deptModel) error {
	_, err := d.session.InsertInto("department").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}
func (d *db) insertDeptTx(m *deptModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("department").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *db) queryOrg(orgId string) (*orgModel, error) {
	var m *orgModel
	_, err := d.session.Select("*").From("organization").Where("org_id=?", orgId).Load(&m)
	return m, err
}
func (d *db) queryOrgsWithOrgIds(orgIds []string) ([]*orgModel, error) {
	var models []*orgModel
	_, err := d.session.Select("*").From("organization").Where("org_id in ?", orgIds).Load(&models)
	return models, err
}

func (d *db) queryOrgEmployee(uid, orgId string) (*orgEmployeeModel, error) {
	var m *orgEmployeeModel
	_, err := d.session.Select("*").From("organization_employee").Where("org_id=? and employee_uid=?", orgId, uid).Load(&m)
	return m, err
}
func (d *db) queryOrgEmployees(orgId string) ([]*orgEmployeeModel, error) {
	var models []*orgEmployeeModel
	_, err := d.session.Select("*").From("organization_employee").Where("org_id=? and status=1", orgId).Load(&models)
	return models, err
}

func (d *db) queryOrgEmployeesWithUid(uid string) ([]*orgEmployeeModel, error) {
	var models []*orgEmployeeModel
	_, err := d.session.Select("*").From("organization_employee").Where("employee_uid=? and status=1", uid).Load(&models)
	return models, err
}

func (d *db) queryOrgEmployeesWithOrgIdAndUids(orgId string, uids []string) ([]*orgEmployeeModel, error) {
	var models []*orgEmployeeModel
	_, err := d.session.Select("*").From("organization_employee").Where("org_id=? and status=1  and employee_uid in ?", orgId, uids).Load(&models)
	return models, err
}

func (d *db) insertDeptEmployeeTx(m *deptEmployeeModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("department_employee").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}
func (d *db) insertDeptEmployee(m *deptEmployeeModel) error {
	_, err := d.session.InsertInto("department_employee").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *db) deleteOrgEmployeeTx(uid, orgId string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("organization_employee").Where("org_id=? and employee_uid=?", orgId, uid).Exec()
	return err
}
func (d *db) deleteOrgEmployeesTx(orgId string, uids []string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("organization_employee").Where("org_id=? and employee_uid in ?", orgId, uids).Exec()
	return err
}

func (d *db) deleteDeptEmployeeTx(uid, deptId string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("department_employee").Where("dept_id=? and employee_uid=?", deptId, uid).Exec()
	return err
}
func (d *db) deleteDeptEmployeesWithOrgidAndUidTx(orgId, uid string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("department_employee").Where("org_id=? and employee_uid=?", orgId, uid).Exec()
	return err
}

func (d *db) queryDeptsWithIds(deptIds []string) ([]*deptModel, error) {
	var models []*deptModel
	_, err := d.session.Select("*").From("department").Where("dept_id in ?", deptIds).Load(&models)
	return models, err
}

func (d *db) queryDeptWithId(deptId string) (*deptModel, error) {
	var m *deptModel
	_, err := d.session.Select("*").From("department").Where("dept_id=?", deptId).Load(&m)
	return m, err
}

func (d *db) queryDeptsWithParentId(parentId string) ([]*deptModel, error) {
	var models []*deptModel
	_, err := d.session.Select("*").From("department").Where("parent_id=?", parentId).Load(&models)
	return models, err
}

func (d *db) deleteOrgTx(orgId string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("organization").Where("org_id=?", orgId).Exec()
	return err
}
func (d *db) deleteOrgEmployeeWithOrgIdTx(orgId string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("organization_employee").Where("org_id=?", orgId).Exec()
	return err
}

func (d *db) deleteDeptWithOrgIdTx(orgId string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("department").Where("org_id=?", orgId).Exec()
	return err
}

func (d *db) deleteDeptWithId(deptId string) error {
	_, err := d.session.DeleteFrom("department").Where("dept_id=?", deptId).Exec()
	return err
}

func (d *db) deleteDeptEmployeesWithOrgidTx(orgId string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("department_employee").Where("org_id=?", orgId).Exec()
	return err
}

func (d *db) deleteDeptEmployeesWithOrgIdAndUidsTx(orgId string, uids []string, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("department_employee").Where("org_id=? and employee_uid in ? ", orgId, uids).Exec()
	return err
}

func (d *db) queryDeptWithShortNo(shortNo string) (*deptModel, error) {
	var m *deptModel
	_, err := d.session.Select("*").From("department").Where("short_no=?", shortNo).Load(&m)
	return m, err
}
func (d *db) queryDeptEmployeesWithOrgIdAndUids(orgId string, uids []string) ([]*deptEmployeeModel, error) {
	var models []*deptEmployeeModel
	_, err := d.session.Select("*").From("department_employee").Where("org_id=? and employee_uid in ?", orgId, uids).Load(&models)
	return models, err
}

func (d *db) queryDeptEmployeesWithOrgIdAndUid(orgId string, uid string) ([]*deptEmployeeModel, error) {
	var models []*deptEmployeeModel
	_, err := d.session.Select("*").From("department_employee").Where("org_id=? and employee_uid=?", orgId, uid).Load(&models)
	return models, err
}

func (d *db) queryDeptEmployeesWithDeptId(deptId string) ([]*deptEmployeeModel, error) {
	var models []*deptEmployeeModel
	_, err := d.session.Select("*").From("department_employee").Where("dept_id=?", deptId).Load(&models)
	return models, err
}
func (d *db) searchEmployee(orgId string, keyword string) ([]*orgEmployeeModel, error) {
	var list []*orgEmployeeModel
	_, err := d.session.Select("*").From("organization_employee").Where("employee_name like ?", "%"+keyword+"%").Load(&list)
	return list, err
}

func (d *db) updateDept(dept *deptModel) error {
	_, err := d.session.Update("department").SetMap(map[string]interface{}{
		"name":      dept.Name,
		"parent_id": dept.ParentId,
		"short_no":  dept.ShortNo,
	}).Where("dept_id=?", dept.DeptId).Exec()
	return err
}
func (d *db) updateOrg(org *orgModel) error {
	_, err := d.session.Update("organization").SetMap(map[string]interface{}{
		"name":           org.Name,
		"short_no":       org.ShortNo,
		"is_upload_logo": org.IsUploadLogo,
	}).Where("org_id=?", org.OrgId).Exec()
	return err
}

type orgModel struct {
	OrgId        string
	ShortNo      string
	Name         string
	Creator      string
	IsUploadLogo int
	dba.BaseModel
}

type orgEmployeeModel struct {
	OrgId          string
	EmployeeUid    string
	EmployeeName   string
	Role           int
	Status         int // 1.正常 0.离职
	EmploymentTime int // 入职时间
	dba.BaseModel
}

type deptModel struct {
	OrgId          string // 组织ID
	DeptId         string // 部门ID
	Name           string // 部门名
	ParentId       string // 上级部门ID
	ShortNo        string // 部门短号
	IsCreatedGroup int    // 是否创建群
	dba.BaseModel
}
type deptEmployeeModel struct {
	OrgId         string // 组织ID
	DeptId        string // 部门ID
	EmployeeId    string // 工号
	WorkforceType string // 人员类型 ‘正式’|‘编外’|‘顾问’
	JobTitle      string // 职务
	EmployeeUid   string
}
