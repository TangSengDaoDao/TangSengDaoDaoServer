package user

import (
	"fmt"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// 上传用户通讯录好友
func (u *User) addMaillist(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	var req []*mailListReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseError(errors.New("请求数据格式有误！"))
		return
	}
	result := make([]*mailListResp, 0)
	if len(req) == 0 {
		c.Response(result)
		return
	}
	loginUser, err := u.db.QueryByUID(loginUID)
	if err != nil {
		c.ResponseError(errors.New("查询登录用户信息错误"))
		return
	}
	tx, _ := u.db.session.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	for _, maillist := range req {
		zone := maillist.Zone
		if maillist.Zone == "" && !strings.HasPrefix(maillist.Phone, "00") {
			zone = loginUser.Zone
		}
		err := u.maillistDB.insertTx(&maillistModel{
			UID:     loginUID,
			Name:    maillist.Name,
			Zone:    zone,
			Phone:   maillist.Phone,
			Vercode: fmt.Sprintf("%s@%d", util.GenerUUID(), common.MailList),
		}, tx)
		if err != nil {
			tx.RollbackUnlessCommitted()
			u.Error("添加用户通讯录联系人错误")
			c.ResponseError(errors.New("添加用户通讯录联系人错误"))
			return
		}
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		u.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		return
	}
	c.ResponseOK()
}

// 获取用户通讯录好友
func (u *User) getMailList(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	result := make([]*mailListResp, 0)
	mailLists, err := u.maillistDB.query(loginUID)
	if err != nil {
		u.Error("查询用户通讯录数据错误")
		c.ResponseError(errors.New("查询用户通讯录数据错误"))
		return
	}
	if mailLists == nil {
		c.Response(result)
		return
	}
	phones := make([]string, 0)
	for _, m := range mailLists {
		phones = append(phones, fmt.Sprintf("%s%s", m.Zone, m.Phone))
	}
	users, err := u.db.QueryByPhones(phones)
	if err != nil {
		u.Error("批量查询用户信息错误")
		c.ResponseError(errors.New("批量查询用户信息错误"))
		return
	}
	friends, err := u.friendDB.QueryFriends(loginUID)
	if err != nil {
		u.Error("查询用户好友错误")
		c.ResponseError(errors.New("查询用户好友错误"))
		return
	}
	for _, m := range mailLists {
		var uid = ""
		for _, user := range users {
			if user.Zone == m.Zone && user.Phone == m.Phone {
				uid = user.UID
				break
			}
		}
		if uid == "" {
			continue
		}
		var isFriend = 0
		for _, friend := range friends {
			if uid != "" && friend.ToUID == uid {
				isFriend = 1
				break
			}
		}
		result = append(result, &mailListResp{
			Vercode:  m.Vercode,
			Phone:    m.Phone,
			Name:     m.Name,
			Zone:     m.Zone,
			UID:      uid,
			IsFriend: isFriend,
		})
	}
	c.Response(result)
}

type mailListReq struct {
	Name  string `json:"name"`
	Zone  string `json:"zone"`
	Phone string `json:"phone"`
}

type mailListResp struct {
	Name     string `json:"name"`
	Zone     string `json:"zone"`
	Phone    string `json:"phone"`
	UID      string `json:"uid"`
	Vercode  string `json:"vercode"`
	IsFriend int    `json:"is_friend"`
}
