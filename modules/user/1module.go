package user

import (
	"embed"
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/model"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

//go:embed sql
var sqlFS embed.FS

//go:embed swagger/api.yaml
var swaggerContent string

//go:embed swagger/friend.yaml
var friendSwaggerContent string

func init() {

	// ====================== 注册用户模块 ======================
	register.AddModule(func(ctx interface{}) register.Module {
		x := ctx.(*config.Context)
		api := New(x)
		return register.Module{
			Name: "user",
			SetupAPI: func() register.APIRouter {
				return api
			},
			Swagger: swaggerContent,
			SQLDir:  register.NewSQLFS(sqlFS),
			IMDatasource: register.IMDatasource{
				SystemUIDs: func() ([]string, error) {
					users, err := api.userService.GetUsersWithCategory(CategoryService)
					if err != nil {
						return nil, err
					}
					uids := make([]string, 0, len(users))
					if len(users) > 0 {
						for _, user := range users {
							uids = append(uids, user.UID)
						}
					}
					return uids, nil
				},
			},
			BussDataSource: register.BussDataSource{
				ChannelGet: func(channelID string, channelType uint8, loginUID string) (*model.ChannelResp, error) {
					if channelType != common.ChannelTypePerson.Uint8() {
						return nil, register.ErrDatasourceNotProcess
					}
					userDetailResp, err := api.userService.GetUserDetail(channelID, loginUID)
					if err != nil {
						return nil, err
					}
					if userDetailResp == nil {
						api.Error("用户不存在！", zap.String("channel_id", channelID))
						return nil, errors.New("用户不存在！")
					}
					return newChannelRespWithUserDetailResp(userDetailResp), nil
				},
			},
		}
	})

	// ====================== 注册好友模块 ======================
	register.AddModule(func(ctx interface{}) register.Module {
		api := NewFriend(ctx.(*config.Context))
		return register.Module{
			Name: "friend",
			SetupAPI: func() register.APIRouter {
				return api
			},
			Swagger: friendSwaggerContent,
			IMDatasource: register.IMDatasource{
				HasData: func(channelID string, channelType uint8) register.IMDatasourceType {
					if channelType == common.ChannelTypePerson.Uint8() {
						return register.IMDatasourceTypeWhitelist
					}
					return register.IMDatasourceTypeNone
				},
				Whitelist: func(channelID string, channelType uint8) ([]string, error) {
					friends, err := api.userService.GetFriends(channelID)
					if err != nil {
						return nil, err
					}
					firendUIDs := make([]string, 0, len(friends))
					if len(friends) > 0 {
						for _, friend := range friends {
							if friend.IsAlone == 0 {
								firendUIDs = append(firendUIDs, friend.UID)
							}
						}
					}
					return firendUIDs, nil
				},
			},
		}
	})

	// ====================== 注册用户管理模块 ======================
	register.AddModule(func(ctx interface{}) register.Module {

		return register.Module{
			Name: "user_manager",
			SetupAPI: func() register.APIRouter {
				return NewManager(ctx.(*config.Context))
			},
		}
	})

}

func newChannelRespWithUserDetailResp(user *UserDetailResp) *model.ChannelResp {

	resp := &model.ChannelResp{}
	resp.Channel.ChannelID = user.UID
	resp.Channel.ChannelType = uint8(common.ChannelTypePerson)
	resp.Name = user.Name
	resp.Username = user.Username
	resp.Logo = fmt.Sprintf("users/%s/avatar", user.UID)
	resp.Mute = user.Mute
	resp.Stick = user.Top
	resp.Receipt = user.Receipt
	resp.Robot = user.Robot
	resp.Online = user.Online
	resp.LastOffline = int64(user.LastOffline)
	resp.DeviceFlag = user.DeviceFlag
	resp.Category = user.Category
	resp.Follow = user.Follow
	resp.Remark = user.Remark
	resp.Status = user.Status
	resp.BeBlacklist = user.BeBlacklist
	resp.BeDeleted = user.BeDeleted
	resp.Flame = user.Flame
	resp.FlameSecond = user.FlameSecond
	extraMap := make(map[string]interface{})
	extraMap["sex"] = user.Sex
	extraMap["chat_pwd_on"] = user.ChatPwdOn
	extraMap["short_no"] = user.ShortNo
	extraMap["source_desc"] = user.SourceDesc
	extraMap["vercode"] = user.Vercode
	extraMap["screenshot"] = user.Screenshot
	extraMap["revoke_remind"] = user.RevokeRemind
	resp.Extra = extraMap

	return resp
}
