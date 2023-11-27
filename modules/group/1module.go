package group

import (
	"embed"
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/model"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
)

//go:embed sql
var sqlFS embed.FS

//go:embed swagger/api.yaml
var swaggerContent string

func init() {
	register.AddModule(func(ctx interface{}) register.Module {

		fmt.Println("register......")
		api := New(ctx.(*config.Context))
		return register.Module{
			Name: "group",
			SetupAPI: func() register.APIRouter {
				return api
			},
			SQLDir:  register.NewSQLFS(sqlFS),
			Swagger: swaggerContent,
			IMDatasource: register.IMDatasource{
				HasData: func(channelID string, channelType uint8) register.IMDatasourceType {
					if channelType == common.ChannelTypeGroup.Uint8() {
						return register.IMDatasourceTypeChannelInfo | register.IMDatasourceTypeSubscribers | register.IMDatasourceTypeBlacklist | register.IMDatasourceTypeWhitelist
					}
					return register.IMDatasourceTypeNone
				},
				ChannelInfo: func(channelID string, channelType uint8) (map[string]interface{}, error) {
					groupInfo, err := api.groupService.GetGroupWithGroupNo(channelID)
					if err != nil {
						return nil, err
					}
					channelInfoMap := map[string]interface{}{}
					if groupInfo != nil {
						if groupInfo.Status == GroupStatusDisabled {
							channelInfoMap["ban"] = 1
						}
						if groupInfo.GroupType == GroupTypeSuper {
							channelInfoMap["large"] = 1
						}

					}
					return channelInfoMap, nil
				},
				Subscribers: func(channelID string, channelType uint8) ([]string, error) {

					mebmers, err := api.groupService.GetMembers(channelID)
					if err != nil {
						return nil, err
					}
					subscribers := make([]string, 0)
					if len(mebmers) > 0 {
						for _, member := range mebmers {
							subscribers = append(subscribers, member.UID)
						}
					}
					return subscribers, nil
				},
				Blacklist: func(channelID string, channelType uint8) ([]string, error) {
					return api.groupService.GetBlacklistMemberUIDs(channelID)
				},
				Whitelist: func(channelID string, channelType uint8) ([]string, error) {
					groupInfo, err := api.groupService.GetGroupWithGroupNo(channelID)
					if err != nil {
						return nil, err
					}
					if groupInfo == nil {
						return nil, nil
					}
					if groupInfo.Forbidden == 1 {
						return api.groupService.GetMemberUIDsOfManager(channelID)
					}
					return make([]string, 0), nil
				},
			},
			BussDataSource: register.BussDataSource{
				ChannelGet: func(channelID string, channelType uint8, loginUID string) (*model.ChannelResp, error) {
					if channelType != common.ChannelTypeGroup.Uint8() {
						return nil, register.ErrDatasourceNotProcess
					}
					groupResp, err := api.groupService.GetGroupDetail(channelID, loginUID)
					if err != nil {
						return nil, err
					}
					return newChannelRespWithGroupResp(groupResp), nil
				},
			},
		}
	})

	register.AddModule(func(ctx interface{}) register.Module {
		return register.Module{
			SetupAPI: func() register.APIRouter {
				return NewManager(ctx.(*config.Context))
			},
		}
	})
}

func newChannelRespWithGroupResp(groupResp *GroupResp) *model.ChannelResp {
	resp := &model.ChannelResp{}
	resp.Channel.ChannelID = groupResp.GroupNo
	resp.Channel.ChannelType = uint8(common.ChannelTypeGroup)
	resp.Name = groupResp.Name
	resp.Remark = groupResp.Remark
	resp.Logo = fmt.Sprintf("groups/%s/avatar", groupResp.GroupNo)
	resp.Notice = groupResp.Notice
	resp.Mute = groupResp.Mute
	resp.Stick = groupResp.Top
	resp.Receipt = groupResp.Receipt
	resp.ShowNick = groupResp.ShowNick
	resp.Forbidden = groupResp.Forbidden
	resp.Invite = groupResp.Invite
	resp.Status = groupResp.Status
	resp.Save = groupResp.Save
	resp.Remark = groupResp.Remark
	resp.Flame = groupResp.Flame
	resp.FlameSecond = groupResp.FlameSecond
	resp.Category = groupResp.Category
	extraMap := make(map[string]interface{})
	extraMap["forbidden_add_friend"] = groupResp.ForbiddenAddFriend
	extraMap["screenshot"] = groupResp.Screenshot
	extraMap["revoke_remind"] = groupResp.RevokeRemind
	extraMap["join_group_remind"] = groupResp.JoinGroupRemind
	extraMap["chat_pwd_on"] = groupResp.ChatPwdOn
	extraMap["allow_view_history_msg"] = groupResp.AllowViewHistoryMsg
	extraMap["group_type"] = groupResp.GroupType

	if groupResp.MemberCount != 0 {
		extraMap["member_count"] = groupResp.MemberCount
	}
	if groupResp.OnlineCount != 0 {
		extraMap["online_count"] = groupResp.OnlineCount
	}
	if groupResp.Quit != 0 {
		extraMap["quit"] = groupResp.Quit
	}
	if groupResp.Role != 0 {
		extraMap["role"] = groupResp.Role
	}
	if groupResp.ForbiddenExpirTime != 0 {
		extraMap["forbidden_expir_time"] = groupResp.ForbiddenExpirTime
	}

	resp.Extra = extraMap

	return resp
}
