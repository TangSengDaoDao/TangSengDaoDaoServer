package robot

import (
	"errors"
	"strconv"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

type Manager struct {
	ctx *config.Context
	log.Log
	db *robotDB
}

func NewManager(ctx *config.Context) *Manager {
	return &Manager{
		ctx: ctx,
		Log: log.NewTLog("robotManager"),
		db:  newBotDB(ctx),
	}
}

// 路由配置
func (m *Manager) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1/manager", m.ctx.AuthMiddleware(r))
	{
		auth.GET("/robot/menus", m.list)                                 // 机器人菜单
		auth.DELETE("/robot/:robot_id/:id", m.delete)                    // 删除某个机器人菜单
		auth.PUT("/robot/status/:robot_id/:status", m.updateRobotStatus) // 修改机器人状态
	}
}

// 查询某个机器人菜单
func (m *Manager) list(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	robotID := c.Query("robot_id")
	if robotID == "" {
		c.ResponseError(errors.New("机器人ID不能为空"))
		return
	}
	list, err := m.db.queryMenusWithRobotID(robotID)
	if err != nil {
		c.ResponseError(errors.New("查询机器人菜单错误"))
		return
	}
	resps := make([]*robotMenu, 0)
	if len(list) == 0 {
		c.Response(resps)
		return
	}

	for _, menu := range list {
		resps = append(resps, &robotMenu{
			Id:        menu.Id,
			CMD:       menu.CMD,
			Remark:    menu.Remark,
			Type:      menu.Type,
			RobotID:   menu.RobotID,
			CreatedAt: menu.CreatedAt.String(),
			UpdatedAt: menu.UpdatedAt.String(),
		})
	}
	c.Response(resps)
}

func (m *Manager) delete(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	robot_id := c.Param("robot_id")
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if robot_id == "" {
		c.ResponseError(errors.New("机器人ID不能为空"))
		return
	}
	robot, err := m.db.queryRobotWithRobtID(robot_id)
	if err != nil {
		c.ResponseError(errors.New("查询操作的机器人错误"))
		return
	}
	if robot == nil {
		c.ResponseError(errors.New("操作的机器人不存在"))
		return
	}
	tx, _ := m.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = m.db.deleteMenuWithID(robot_id, id, tx)
	if err != nil {
		tx.Rollback()
		m.Error("删除机器人菜单失败", zap.Error(err))
		c.ResponseError(errors.New("删除机器人菜单失败"))
		return
	}
	robot.Version = m.ctx.GenSeq(common.RobotSeqKey)
	err = m.db.updateRobotTx(robot, tx)
	if err != nil {
		tx.Rollback()
		m.Error("修改机器人版本号错误", zap.Error(err))
		c.ResponseError(errors.New("修改机器人版本号错误"))
		return
	}
	err = tx.Commit()
	if err != nil {
		tx.RollbackUnlessCommitted()
		m.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		return
	}
	c.ResponseOK()
}

// 启用或禁用机器人
func (m *Manager) updateRobotStatus(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	robot_id := c.Param("robot_id")
	status, _ := strconv.ParseInt(c.Param("status"), 10, 64)

	if robot_id == "" {
		c.ResponseError(errors.New("机器人ID不能为空"))
		return
	}
	robot, err := m.db.queryRobotWithRobtID(robot_id)
	if err != nil {
		c.ResponseError(errors.New("查询操作的机器人错误"))
		return
	}
	robot.Status = int(status)
	if robot == nil {
		c.ResponseError(errors.New("操作的机器人不存在"))
		return
	}
	err = m.db.updateRobot(robot)
	if err != nil {
		c.ResponseError(errors.New("修改机器人状态信息错误"))
		return
	}
	c.ResponseOK()
}

type robotMenu struct {
	Id        int64  `json:"id"`
	CMD       string `json:"cmd"`
	Remark    string `json:"remark"`
	Type      string `json:"type"`
	RobotID   string `json:"robot_id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
