package source

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
)

// IGetGroupMemberProvider 获取群成员提供者
type IGetGroupMemberProvider interface {
	//获取群成员信息
	GetGroupMemberByVercode(vercode string) (*GroupMember, error)
	GetGroupMemberByVercodes(vercodes []string) ([]*GroupMember, error)
	//获取群信息
	GetGroupByGroupNo(groupNo string) (*GroupModel, error)
	// 通过uid获取群成员信息
	GetGroupMemberByUID(uid string, groupNo string) (*GroupMember, error)
}

// IGetUserProvider 获取用户提供者
type IGetUserProvider interface {
	//通过vercode获取用户信息
	GetUserByVercode(vercode string) (*UserModel, error)
	//通过手机通讯录验证码获取用户信息
	GetUserByMailListVercode(vercode string) (*UserModel, error)
	//通过qrvercode获取用户信息
	GetUserByQRVercode(qrvercode string) (*UserModel, error)
	//通过UID获取用户信息
	GetUserByUID(uid string) (*UserModel, error)
	//通过vercode获取好友信息
	GetFriendByVercode(vercode string) (*FriendModel, error)
	GetFriendByVercodes(vercodes []string) ([]*FriendModel, error)
}

var getGroupMemberProvide IGetGroupMemberProvider

// SetGroupMemberProvider 设置获取群成员提供者
func SetGroupMemberProvider(groupMemberProvide IGetGroupMemberProvider) {
	getGroupMemberProvide = groupMemberProvide
}

var getUserProvider IGetUserProvider

// SetUserProvider 设置获取用户提供者
func SetUserProvider(userProvider IGetUserProvider) {
	getUserProvider = userProvider
}

// GetSoruce 获取好友来源
func GetSoruce(code string) string {
	if code == "" {
		return ""
	}
	strs := strings.Split(code, "@")
	codeTypeStr, _ := strconv.Atoi(strs[1])
	codeType := common.VercodeType(codeTypeStr)
	if codeType != common.Friend && codeType != common.QRCode && codeType != common.User && codeType != common.GroupMember && codeType != common.MailList {
		return ""
	}
	if codeType == common.Friend {
		friend, err := getUserProvider.GetFriendByVercode(code)
		if err != nil || friend == nil || friend.Vercode != code {
			return "通过名片添加"
		}
		user, err := getUserProvider.GetUserByUID(friend.UID)
		if err != nil || user == nil {
			return "通过名片添加"
		}
		return fmt.Sprintf("通过%s推荐的名片添加", user.Name)
	} else if codeType == common.User {
		return "通过搜索添加"
	} else if codeType == common.QRCode {
		return "通过扫一扫添加"
	} else if codeType == common.GroupMember {
		groupMember, err := getGroupMemberProvide.GetGroupMemberByVercode(code)
		if err != nil || groupMember == nil {
			return "通过群聊添加"
		}
		group, err := getGroupMemberProvide.GetGroupByGroupNo(groupMember.GroupNo)
		if err != nil || group == nil {
			return "通过群聊添加"
		}
		return fmt.Sprintf("通过群聊'%s'添加", group.Name)
	} else if codeType == common.MailList {
		return "通过手机通讯录添加"
	}
	return ""
}

// GetSources 批量获取好友来源 返回 以code为key 来源内容为value的map
func GetSources(codes []string) (map[string]string, error) {
	if len(codes) == 0 {
		return nil, nil
	}
	codeTypeMap := map[common.VercodeType][]string{}
	for _, code := range codes {
		strs := strings.Split(code, "@")
		if len(strs) < 2 {
			continue
		}
		codeTypeStr, _ := strconv.Atoi(strs[1])
		codeType := common.VercodeType(codeTypeStr)
		if codeType != common.Friend && codeType != common.QRCode && codeType != common.User && codeType != common.GroupMember && codeType != common.MailList {
			continue
		}
		codes := codeTypeMap[codeType]
		if codes == nil {
			codes = make([]string, 0)
		}
		codes = append(codes, code)
		codeTypeMap[codeType] = codes
	}
	codeSourceMap := map[string]string{}

	for codeType, codes := range codeTypeMap {
		if codeType == common.Friend {
			friends, err := getUserProvider.GetFriendByVercodes(codes)
			if err != nil {
				return nil, err
			}
			for _, code := range codes {
				exist := false
				for _, friend := range friends {
					if friend.Vercode == code {
						codeSourceMap[code] = fmt.Sprintf("通过%s推荐的名片添加", friend.Name)
						exist = true
						break
					}
				}
				if !exist {
					codeSourceMap[code] = "通过名片添加"
				}
			}
		} else if codeType == common.User {
			for _, code := range codes {
				codeSourceMap[code] = "通过搜索添加"
			}
		} else if codeType == common.QRCode {
			for _, code := range codes {
				codeSourceMap[code] = "通过扫一扫添加"
			}
		} else if codeType == common.GroupMember {
			groupMembers, err := getGroupMemberProvide.GetGroupMemberByVercodes(codes)
			if err != nil {
				return nil, err
			}
			if len(groupMembers) > 0 {
				for _, code := range codes {
					exist := false
					for _, groupMember := range groupMembers {
						if code == groupMember.Vercode {
							codeSourceMap[code] = fmt.Sprintf("通过群聊'%s'添加", groupMember.Name)
							exist = true
						}
					}
					if !exist {
						codeSourceMap[code] = "通过群聊添加"
					}
				}
			}
		} else if codeType == common.MailList {
			for _, code := range codes {
				codeSourceMap[code] = "通过手机通讯录添加"
			}
		}
	}
	return codeSourceMap, nil
}

// CheckRequestAddFriendCode 检测加好友申请code是否有效
func CheckRequestAddFriendCode(code string, requestUID string) error {
	err := CheckSource(code)
	if err != nil {
		return err
	}
	strs := strings.Split(code, "@")
	codeTypeStr, _ := strconv.Atoi(strs[1])
	codeType := common.VercodeType(codeTypeStr)
	//验证群是否开启禁止加好友
	if codeType == common.GroupMember {
		groupMember, err := getGroupMemberProvide.GetGroupMemberByVercode(code)
		if err != nil {
			return err
		}
		group, err := getGroupMemberProvide.GetGroupByGroupNo(groupMember.GroupNo)
		if err != nil {
			return err
		}
		if group == nil {
			return errors.New("申请的群不存在")
		}
		// 查询申请人是否为该群的管理员
		groupMember, err = getGroupMemberProvide.GetGroupMemberByUID(requestUID, group.GroupNo)
		if err != nil {
			return err
		}
		isCanApply := false
		if groupMember != nil && groupMember.Role != int(common.GroupMemberRoleNormal) {
			isCanApply = true
		}
		if group.ForbiddenAddFriend == 1 && !isCanApply {
			return errors.New("群主或群管理员开启了群内禁止加好友")
		}
	}
	return nil
}

// CheckSource 验证加好友来源
func CheckSource(code string) error {
	strs := strings.Split(code, "@")
	codeTypeStr, _ := strconv.Atoi(strs[1])
	codeType := common.VercodeType(codeTypeStr)
	if codeType != common.Friend && codeType != common.QRCode && codeType != common.User && codeType != common.GroupMember && codeType != common.MailList {
		return errors.New("来源错误")
	}
	if codeType == common.Friend {
		// 来源好友推荐
		friend, err := getUserProvider.GetFriendByVercode(code)
		if err != nil {
			return err
		}
		if friend == nil || friend.Vercode != code {
			return errors.New("来源不匹配")
		}
	} else if codeType == common.User {
		// 搜索
		user, err := getUserProvider.GetUserByVercode(code)
		if err != nil {
			return errors.New("通过验证码查询用户信息错误")
		}
		if user == nil || user.Vercode != code {
			return errors.New("来源不匹配")
		}
	} else if codeType == common.QRCode {
		// 二维码
		user, err := getUserProvider.GetUserByQRVercode(code)
		if err != nil {
			return errors.New("通过验证码查询用户信息错误")
		}
		if user == nil || user.QRVercode != code {
			return errors.New("来源不匹配")
		}
	} else if codeType == common.GroupMember {
		// 群
		groupMember, err := getGroupMemberProvide.GetGroupMemberByVercode(code)
		if err != nil {
			return err
		}
		if groupMember == nil || groupMember.Vercode != code {
			return errors.New("来源不匹配")
		}
	} else if codeType == common.MailList {
		// 手机通讯录
		user, err := getUserProvider.GetUserByMailListVercode(code)
		if err != nil {
			return err
		}
		if user == nil || user.MailListVercode != code {
			return errors.New("来源不匹配")
		}
	}
	return nil
}

// GroupMember 群成员
type GroupMember struct {
	UID     string
	Name    string
	GroupNo string
	Vercode string
	Role    int
}

// GroupModel 群model
type GroupModel struct {
	Name               string
	GroupNo            string
	ForbiddenAddFriend int //群内禁止加好友
}

// FriendModel 好友model
type FriendModel struct {
	UID     string
	ToUID   string
	Vercode string
	Name    string
}

// UserModel 用户model
type UserModel struct {
	UID             string
	Name            string
	Vercode         string
	QRVercode       string
	MailListVercode string
}
