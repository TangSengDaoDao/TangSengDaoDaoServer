package robot

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type robotDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newBotDB(ctx *config.Context) *robotDB {
	return &robotDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}
func (d *robotDB) queryRobotWithRobtID(robotID string) (*robot, error) {
	var m *robot
	_, err := d.session.Select("*").From("robot").Where("robot_id=?", robotID).Load(&m)
	return m, err
}
func (d *robotDB) queryVaildRobotWithRobtID(robotID string) (*robot, error) {
	var m *robot
	_, err := d.session.Select("*").From("robot").Where("robot_id=? and status=1", robotID).Load(&m)
	return m, err
}

func (d *robotDB) exist(robotID string) (bool, error) {
	var cn int
	err := d.session.Select("count(*)").From("robot").Where("robot_id=? and status=1", robotID).LoadOne(&cn)
	return cn > 0, err
}

func (d *robotDB) insert(m *robot) error {
	_, err := d.session.InsertInto("robot").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *robotDB) insertTx(m *robot, tx *dbr.Tx) error {
	_, err := tx.InsertInto("robot").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}
func (d *robotDB) insertMenuTx(m *menu, tx *dbr.Tx) error {
	_, err := tx.InsertInto("robot_menu").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *robotDB) queryWithIDs(robotIDs []string) ([]*robot, error) {
	var list []*robot
	_, err := d.session.Select("*").From("robot").Where("robot_id in ?", robotIDs).Load(&list)
	return list, err
}
func (d *robotDB) queryWithUsernames(usernames []string) ([]*robot, error) {
	var list []*robot
	_, err := d.session.Select("*").From("robot").Where("username in ?", usernames).Load(&list)
	return list, err
}
func (d *robotDB) queryWithUsername(username string) (*robot, error) {
	var rb *robot
	_, err := d.session.Select("*").From("robot").Where("username = ?", username).Load(&rb)
	return rb, err
}

func (d *robotDB) queryVaildRobotIDs(robotIDs []string) ([]string, error) {
	var vaildRobotIDs []string
	_, err := d.session.Select("robot_id").From("robot").Where("robot_id in ?", robotIDs).Load(&vaildRobotIDs)
	return vaildRobotIDs, err
}

// 同步机器人菜单
func (d *robotDB) queryMenusWithRobotIDs(uids []string) ([]*menu, error) {
	var menus []*menu
	_, err := d.session.Select("*").From("robot_menu").Where("robot_id in ?", uids).OrderDir("created_at", false).Load(&menus)
	return menus, err
}

// 修改机器人信息
func (d *robotDB) updateRobotTx(m *robot, tx *dbr.Tx) error {
	_, err := tx.Update("robot").SetMap(map[string]interface{}{
		"version": m.Version,
	}).Where("robot_id=?", m.RobotID).Exec()
	return err
}
func (d *robotDB) updateRobot(m *robot) error {
	_, err := d.session.Update("robot").SetMap(map[string]interface{}{
		"version": m.Version,
		"status":  m.Status,
	}).Where("robot_id=?", m.RobotID).Exec()
	return err
}
func (d *robotDB) queryMenusWithRobotID(robotID string) ([]*menu, error) {
	var menus []*menu
	_, err := d.session.Select("*").From("robot_menu").Where("robot_id=?", robotID).OrderDir("created_at", false).Load(&menus)
	return menus, err
}
func (d *robotDB) deleteMenuWithID(robotID string, id int64, tx *dbr.Tx) error {
	_, err := tx.DeleteFrom("robot_menu").Where("robot_id=? and id=?", robotID, id).Exec()
	return err
}

type menu struct {
	RobotID string // 机器人ID
	CMD     string // 命令
	Remark  string // 命令说明
	Type    string // 命令类型
	db.BaseModel
}
type robot struct {
	AppID       string
	RobotID     string // 机器人唯一ID
	Username    string // 机器人用户名
	InlineOn    int    // 是否开启行内搜索
	Placeholder string // 输入框占位符，开启行内搜索有效
	Token       string
	Version     int64
	Status      int
	db.BaseModel
}
