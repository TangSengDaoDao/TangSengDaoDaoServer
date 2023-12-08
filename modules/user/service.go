package user

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

var ErrorUserNotExist = errors.New("用户不存在！")

// IService 用户服务接口
type IService interface {
	//获取用户
	GetUser(uid string) (*Resp, error)
	// 获取用户详情（包括与loginUID的关系等等）
	GetUserDetail(uid string, loginUID string) (*UserDetailResp, error)
	// 批量获取用户详情
	GetUserDetails(uids []string, loginUID string) ([]*UserDetailResp, error)
	// 通过用户名获取用户
	GetUserWithUsername(username string) (*Resp, error)
	// 通过用户名获取用户uid集合
	GetUserUIDWithUsernames(usernames []string) ([]string, error)
	// 批量获取用户信息
	GetUsers(uids []string) ([]*Resp, error)
	// 通过APPID获取用户
	GetUsersWithAppID(appID string) ([]*Resp, error)
	// 获取用户集合
	GetUsersWithCategory(category Category) ([]*Resp, error)
	//查询某个人好友
	GetFriendsWithToUIDs(uid string, toUIDs []string) ([]*FriendResp, error)
	//查询某个用户的所有好友
	GetFriends(uid string) ([]*FriendResp, error)
	//添加一个好友
	AddFriend(uid string, friend *FriendReq) error
	//添加一个用户
	AddUser(user *AddUserReq) error
	// 通过qrvercode获取用户信息
	GetUserWithQRVercode(qrVercode string) (*Resp, error)
	// 获取总用户数量
	GetAllUserCount() (int64, error)
	// 查询某天注册用户数
	GetRegisterWithDate(date string) (int64, error)
	// 获取某个时间区间的注册数量
	GetRegisterCountWithDateSpace(startDate, endDate string) (map[string]int64, error)
	// IsFriend 查询两个用户是否为好友关系
	IsFriend(uid string, toUID string) (bool, error)
	// 获取在线用户
	GetUserOnlineStatus([]string) ([]*OnLineUserResp, error)
	// 更新用户信息
	UpdateUser(req UserUpdateReq) error
	// 获取所有用户
	GetAllUsers() ([]*Resp, error)
	// 更新登录密码
	UpdateLoginPassword(req UpdateLoginPasswordReq) error

	// GetUserSettings 获取用户的配置
	GetUserSettings(uids []string, loginUID string) ([]*SettingResp, error)

	// GetOnetimePrekeyCount 获取用户一次性signal key的数量(决定是否可以开启加密通讯)
	GetOnetimePrekeyCount(uid string) (int, error)

	// 获取设备在线状态
	GetDeviceOnline(uid string, deviceFlag config.DeviceFlag) (*config.OnlinestatusResp, error)
	// 查询在线用户总数量
	GetOnlineCount() (int64, error)
	// 存在黑明单
	ExistBlacklist(uid string, toUID string) (bool, error)
	// 更新用户消息过期时长
	UpdateUserMsgExpireSecond(uid string, msgExpireSecond int64) error
}

// Service Service
type Service struct {
	ctx *config.Context
	db  *DB
	log.Log
	friendDB         *friendDB
	onlineDB         *onlineDB
	settingDB        *SettingDB
	onetimePrekeysDB *onetimePrekeysDB
	onlineService    *OnlineService
}

// NewService NewService
func NewService(ctx *config.Context) IService {
	return &Service{
		ctx:              ctx,
		db:               NewDB(ctx),
		friendDB:         newFriendDB(ctx),
		settingDB:        NewSettingDB(ctx.DB()),
		onetimePrekeysDB: newOnetimePrekeysDB(ctx),
		onlineDB:         newOnlineDB(ctx),
		Log:              log.NewTLog("userService"),
		onlineService:    NewOnlineService(ctx),
	}
}

// 获取所有用户
func (s *Service) GetAllUsers() ([]*Resp, error) {
	models, err := s.db.queryAll()
	if err != nil {
		s.Error("查询所有用户错误", zap.Error(err))
		return nil, err
	}
	list := make([]*Resp, 0)
	for _, user := range models {
		list = append(list, &Resp{
			UID:   user.UID,
			Name:  user.Name,
			Zone:  user.Zone,
			Phone: user.Phone,
		})
	}
	return list, nil
}
func (s *Service) GetUserDetail(uid string, loginUID string) (*UserDetailResp, error) {
	model, err := s.db.QueryDetailByUID(uid, loginUID)
	if err != nil {
		s.Error("查询用户信息失败！", zap.Error(err), zap.String("uid", uid))
		return nil, err
	}
	if model == nil {
		return nil, errors.New("用户信息不存在！")
	}
	onlineM, err := s.onlineDB.queryLastOnlineDeviceWithUID(uid)
	if err != nil {
		s.Error("查询用户在线状态失败", zap.Error(err))
		return nil, err
	}
	var online int
	var lastOffline int
	var deviceFlag config.DeviceFlag
	if onlineM != nil {
		online = onlineM.Online
		lastOffline = onlineM.LastOffline
		deviceFlag = config.DeviceFlag(onlineM.DeviceFlag)
	}
	//查询用户设置
	blacklist := 1
	userSettings, err := s.settingDB.QueryTwoUserSettingModel(uid, loginUID)
	if err != nil {
		s.Error("查询用户设置错误", zap.Error(err))
		return nil, err
	}
	var userSetting *SettingModel
	var toUserSetting *SettingModel
	if len(userSettings) > 0 {
		for _, userSett := range userSettings {
			if userSett.UID == loginUID {
				userSetting = userSett
			} else if userSett.UID == uid {
				toUserSetting = userSett
			}
		}
	}

	if userSetting != nil && userSetting.Blacklist == 1 {
		blacklist = 2
	}
	// 默认打开撤回通知/截屏通知
	if userSetting == nil {
		model.RevokeRemind = 1
		model.Screenshot = 1
		model.Receipt = 1
	}

	friends, err := s.friendDB.queryTwoWithUID(loginUID, uid)
	// isFriend, err := u.friendDB.IsFriend(loginUID, uid)
	if err != nil {
		s.Error("查询是否为好友关系失败", zap.Error(err))
		return nil, err
	}
	var friend *FriendModel
	var toFriend *FriendModel
	if len(friends) > 0 {
		for _, f := range friends {
			if f.UID == loginUID {
				friend = f
			} else if f.UID == uid {
				toFriend = f
			}
		}
	}

	var follow int
	var sourceFrom string
	var remark string
	var beDeleted int
	var beBlacklist int
	var vercode string
	if friend != nil && friend.IsDeleted == 0 {
		follow = 1
		//查询加好友来源
		sourceFrom = source.GetSoruce(friend.SourceVercode)
		if friend.Initiator == 0 && sourceFrom != "" {
			sourceFrom = fmt.Sprintf("对方%s", sourceFrom)
		}

		if toFriend != nil {
			beDeleted = toFriend.IsDeleted

		} else {
			beDeleted = 1
		}
		vercode = friend.Vercode
	}
	if userSetting != nil {
		remark = userSetting.Remark
	}

	if toUserSetting != nil {
		beBlacklist = toUserSetting.Blacklist
	}
	return NewUserDetailResp(model, remark, loginUID, sourceFrom, online, lastOffline, deviceFlag, follow, blacklist, beDeleted, beBlacklist, userSetting, vercode), nil
}

func (s *Service) GetUserDetails(uids []string, loginUID string) ([]*UserDetailResp, error) {

	userDetails, err := s.db.QueryDetailByUIDs(uids, loginUID)
	if err != nil {
		s.Error("查询用户详情失败！")
		return nil, err
	}
	if userDetails == nil {
		return nil, nil
	}
	onlineStatusResults, err := s.onlineDB.queryUserLastNewOnlines(uids)
	if err != nil {
		s.Error("查询用户在线状态失败", zap.Error(err))
		return nil, err
	}
	onlineStatusResultMap := map[string]*onlineStatusWeightModel{}
	if len(onlineStatusResults) > 0 {
		for _, onlineStatusResult := range onlineStatusResults {
			onlineStatusResultMap[onlineStatusResult.UID] = onlineStatusResult
		}
	}
	// 查询loginUID用户对uids的设置
	settings, err := s.settingDB.QueryUserSettings(uids, loginUID)
	if err != nil {
		return nil, err
	}
	settingMap := map[string]*SettingModel{}
	if len(settings) > 0 {
		for _, setting := range settings {
			settingMap[setting.ToUID] = setting
		}
	}

	// 查询uids对loginUID的设置
	toSettings, err := s.settingDB.QueryWithUidsAndToUID(uids, loginUID)
	if err != nil {
		return nil, err
	}
	toSettingMap := map[string]*SettingModel{}
	if len(toSettings) > 0 {
		for _, toSetting := range toSettings {
			toSettingMap[toSetting.UID] = toSetting
		}
	}
	// 查询loginUID与uids的好友
	friends, err := s.friendDB.queryWithToUIDsAndUID(uids, loginUID)
	if err != nil {
		return nil, err
	}
	friendMap := map[string]*FriendModel{}
	if len(friends) > 0 {
		for _, friend := range friends {
			friendMap[friend.ToUID] = friend
		}
	}
	// 好友来源
	friendVSourceVercodes := make([]string, 0, len(friends))
	for _, friend := range friends {
		friendVSourceVercodes = append(friendVSourceVercodes, friend.SourceVercode)
	}
	friendVercodeSourceMap := map[string]string{}
	if len(friendVSourceVercodes) > 0 {
		friendVercodeSourceMap, err = source.GetSources(friendVSourceVercodes)
		if err != nil {
			return nil, err
		}
		if friendVercodeSourceMap == nil {
			friendVercodeSourceMap = map[string]string{}
		}
	}

	// 查询uids内与loginUID的好友 （查询uids与loginUID的好友）
	toFriends, err := s.friendDB.queryWithToUIDAndUIDs(loginUID, uids)
	if err != nil {
		return nil, err
	}
	toFriendMap := map[string]*FriendModel{}
	if len(friends) > 0 {
		for _, toFriend := range toFriends {
			toFriendMap[toFriend.UID] = toFriend
		}
	}

	userDetailResps := make([]*UserDetailResp, 0)

	for _, userDetail := range userDetails {
		uid := userDetail.UID
		online := 0
		lastOffline := 0
		var deviceFlag config.DeviceFlag
		onlineStatus := onlineStatusResultMap[uid]
		if onlineStatus != nil {
			online = onlineStatus.Online
			lastOffline = onlineStatus.LastOffline
			deviceFlag = config.DeviceFlag(onlineStatus.DeviceFlag)
		}
		follow := 0
		nameRemark := ""
		sourceFrom := ""
		vercode := ""
		friend := friendMap[uid]
		if friend != nil && friend.IsDeleted == 0 {
			follow = 1
			sourceFrom = friendVercodeSourceMap[friend.SourceVercode]
			vercode = friend.Vercode
		}

		status := 1
		setting := settingMap[uid] // loginUID用户对对方的设置
		if setting != nil {
			if setting.Blacklist == 1 {
				status = 2 // 拉黑
			}
			nameRemark = setting.Remark

		}

		beBlacklist := 0
		toSetting := toSettingMap[uid] // 对方对loginUID用户的设置
		if toSetting != nil {
			if toSetting.Blacklist == 1 {
				beBlacklist = 1
			}
		}

		beDeleted := 0
		toFriend := toFriendMap[uid]
		if toFriend != nil {
			beDeleted = toFriend.IsDeleted
		} else {
			beDeleted = 1
		}
		userDetailResps = append(userDetailResps, NewUserDetailResp(userDetail, nameRemark, loginUID, sourceFrom, online, lastOffline, deviceFlag, follow, status, beDeleted, beBlacklist, setting, vercode))
	}

	return userDetailResps, nil
}

// GetUserOnlineStatus 查询在线用户
func (s *Service) GetUserOnlineStatus(uids []string) ([]*OnLineUserResp, error) {
	result, err := s.onlineService.GetUserLastOnlineStatus(uids)
	if err != nil {
		s.Error("查询在线用户信息错误", zap.Error(err))
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}
	list := make([]*OnLineUserResp, 0)
	for _, user := range result {
		list = append(list, &OnLineUserResp{
			UID:         user.UID,
			LastOffline: user.LastOffline,
			Online:      user.Online,
			DeviceFlag:  user.DeviceFlag,
		})
	}
	return list, nil
}

// AddUser AddUser
func (s *Service) AddUser(user *AddUserReq) error {
	uid := user.UID
	if strings.TrimSpace(uid) == "" {
		uid = util.GenerUUID()
	}
	username := user.Username
	if strings.TrimSpace(username) == "" {
		username = fmt.Sprintf("%s%s", user.Zone, user.Phone)
	}
	userM := &Model{
		Name:     user.Name,
		UID:      uid,
		Zone:     user.Zone,
		Phone:    user.Phone,
		Username: username,
		Email:    user.Email,
		Status:   1,
	}
	if user.Password != "" {
		userM.Password = util.MD5(util.MD5(user.Password))
	}

	err := s.db.Insert(userM)
	if err != nil {
		s.Error("添加用户失败", zap.Error(err))
		return err
	}
	return nil
}

// AddFriend 添加一个好友
func (s *Service) AddFriend(uid string, friend *FriendReq) error {
	err := s.friendDB.Insert(&FriendModel{
		UID:   friend.UID,
		ToUID: friend.ToUID,
	})
	if err != nil {
		s.Error("添加好友失败", zap.Error(err))
		return err
	}
	return nil
}

// GetFriendsWithToUIDs 查询一批好友
func (s *Service) GetFriendsWithToUIDs(uid string, toUIDs []string) ([]*FriendResp, error) {
	friends, err := s.friendDB.QueryFriendsWithUIDs(uid, toUIDs)
	if err != nil {
		s.Error("批量查询用户失败", zap.Error(err))
		return nil, err
	}
	list := make([]*FriendResp, 0)
	for _, friend := range friends {
		list = append(list, &FriendResp{
			Name: friend.ToName,
			UID:  friend.ToUID,
		})
	}
	return list, nil
}

// GetFriends 查询某个用户的所有好友
func (s *Service) GetFriends(uid string) ([]*FriendResp, error) {
	friends, err := s.friendDB.QueryFriends(uid)
	if err != nil {
		s.Error("批量查询用户失败", zap.Error(err))
		return nil, err
	}
	list := make([]*FriendResp, 0)
	for _, friend := range friends {
		list = append(list, &FriendResp{
			Name:    friend.ToName,
			UID:     friend.ToUID,
			IsAlone: friend.IsAlone,
		})
	}
	return list, nil
}

// GetUser 获取用户
func (s *Service) GetUser(uid string) (*Resp, error) {
	userM, err := s.db.QueryByUID(uid)
	if err != nil {
		return nil, err
	}
	if userM == nil {
		return nil, errors.New("用户不存在！")
	}
	if userM.Status != StatusEnable.Int() {
		return nil, errors.New("用户不可用！")
	}

	return newResp(userM), nil
}

// GetUserWithUsername 获取用户
func (s *Service) GetUserWithUsername(username string) (*Resp, error) {
	userM, err := s.db.QueryByUsername(username)
	if err != nil {
		return nil, err
	}
	if userM == nil {
		return nil, nil
	}
	if userM.Status != StatusEnable.Int() {
		return nil, errors.New("用户不可用！")
	}

	return newResp(userM), nil
}

// GetUserUIDWithUsernames 获取用户uid集合
func (s *Service) GetUserUIDWithUsernames(usernames []string) ([]string, error) {
	if len(usernames) == 0 {
		return nil, nil
	}
	return s.db.QueryUIDsByUsernames(usernames)
}

// GetUsers 批量获取用户
func (s *Service) GetUsers(uids []string) ([]*Resp, error) {
	if len(uids) <= 0 {
		return nil, nil
	}
	userModels, err := s.db.queryByUIDs(uids)
	if err != nil {
		return nil, err
	}
	resps := make([]*Resp, 0, len(userModels))
	if len(userModels) > 0 {
		for _, userModel := range userModels {
			resps = append(resps, newResp(userModel))
		}
	}
	return resps, nil
}

// GetUsersWithAppID 通过appID获取用户集合
func (s *Service) GetUsersWithAppID(appID string) ([]*Resp, error) {
	userModels, err := s.db.QueryWithAppID(appID)
	if err != nil {
		return nil, err
	}
	resps := make([]*Resp, 0, len(userModels))
	if len(userModels) > 0 {
		for _, userModel := range userModels {
			resps = append(resps, newResp(userModel))
		}
	}
	return resps, nil
}

// GetUsersWithCategory 获取用户列表
func (s *Service) GetUsersWithCategory(category Category) ([]*Resp, error) {
	userModels, err := s.db.QueryByCategory(string(category))
	if err != nil {
		return nil, err
	}
	resps := make([]*Resp, 0, len(userModels))
	for _, userM := range userModels {
		resps = append(resps, newResp(userM))
	}
	return resps, nil
}

// GetUserWithQRVercode 通过qrvercode获取用户信息
func (s *Service) GetUserWithQRVercode(qrVercode string) (*Resp, error) {
	userModel, err := s.db.queryByQRVerCode(qrVercode)
	if err != nil {
		return nil, err
	}
	if userModel != nil {
		return newResp(userModel), nil
	}
	return nil, nil
}

// GetAllUserCount 获取总用户数量
func (s *Service) GetAllUserCount() (int64, error) {
	count, err := s.db.queryUserCount()
	return count, err
}

// GetRegisterWithDate 查询某天的注册量
func (s *Service) GetRegisterWithDate(date string) (int64, error) {
	count, err := s.db.queryRegisterCountWithDate(date)
	return count, err
}

// GetRegisterCountWithDateSpace 获取某个时间区间的注册数量
func (s *Service) GetRegisterCountWithDateSpace(startDate, endDate string) (map[string]int64, error) {
	list, err := s.db.queryRegisterCountWithDateSpace(startDate, endDate)
	if err != nil {
		s.Error("查询注册用户数据错误", zap.Error(err))
		return nil, err
	}
	result := make(map[string]int64, 0)
	if len(list) > 0 {
		for _, model := range list {
			key := util.Toyyyy_MM_dd(time.Time(model.CreatedAt))
			if _, ok := result[key]; ok {
				//存在某个
				result[key]++
			} else {
				result[key] = 1
			}
		}
	}
	return result, nil
}

// IsFriend 查询两个用户是否为好友关系
func (s *Service) IsFriend(uid string, toUID string) (bool, error) {
	if uid == "" || toUID == "" {
		return false, errors.New("用户ID不能为空")
	}
	model, err := s.friendDB.queryWithUID(uid, toUID)
	if err != nil {
		s.Error("查询好友关系错误", zap.Error(err))
		return false, errors.New("查询好友关系错误")
	}
	isFriend := true
	if model == nil || model.UID == "" || model.IsDeleted == 1 {
		isFriend = false
	}
	return isFriend, nil
}

func (s *Service) UpdateUser(req UserUpdateReq) error {
	updateMap := map[string]interface{}{}
	if req.Name != nil {
		updateMap["name"] = req.Name
	}
	err := s.db.updateUser(updateMap, req.UID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) UpdateLoginPassword(req UpdateLoginPasswordReq) error {
	if req.UID == "" {
		return errors.New("uid不能为空！")
	}
	if req.Password == "" {
		return errors.New("原密码不能为空！")
	}
	userM, err := s.db.QueryByUID(req.UID)
	if err != nil {
		return err
	}
	if userM == nil {
		return errors.New("用户不存在！")
	}
	if util.MD5(util.MD5(req.Password)) != userM.Password {
		return errors.New("原密码不正确！")
	}

	err = s.db.updatePassword(util.MD5(util.MD5(req.NewPassword)), req.UID)
	if err != nil {
		return errors.New("更新密码失败！")
	}

	return nil
}

func (s *Service) GetUserSettings(uids []string, loginUID string) ([]*SettingResp, error) {
	if len(uids) == 0 || loginUID == "" {
		return nil, nil
	}
	settingModels, err := s.settingDB.QueryUserSettings(uids, loginUID)
	if err != nil {
		return nil, err
	}
	settingResps := make([]*SettingResp, 0)
	if len(settingModels) > 0 {
		for _, settingM := range settingModels {
			settingResps = append(settingResps, toSettingResp(settingM))
		}
	}
	return settingResps, nil
}

func (s *Service) GetOnetimePrekeyCount(uid string) (int, error) {
	cn, err := s.onetimePrekeysDB.queryCount(uid)
	return cn, err
}

func (s *Service) GetDeviceOnline(uid string, deviceFlag config.DeviceFlag) (*config.OnlinestatusResp, error) {
	onlineM, err := s.onlineDB.queryOnlineDevice(uid, deviceFlag)
	if err != nil {
		return nil, err
	}
	if onlineM == nil {
		return nil, nil
	}

	return &config.OnlinestatusResp{
		UID:         onlineM.UID,
		DeviceFlag:  onlineM.DeviceFlag,
		LastOffline: onlineM.LastOffline,
		Online:      onlineM.Online,
	}, nil
}

// 查询在线总数量
func (s *Service) GetOnlineCount() (int64, error) {
	count, err := s.onlineService.GetOnlineCount()
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *Service) ExistBlacklist(uid string, toUID string) (bool, error) {
	return s.friendDB.existBlacklist(uid, toUID)
}

func (s *Service) UpdateUserMsgExpireSecond(uid string, msgExpireSecond int64) error {
	return s.db.updateUserMsgExpireSecond(uid, msgExpireSecond)
}

// Resp 用户返回
type Resp struct {
	UID             string
	Name            string
	Zone            string
	Phone           string
	Email           string
	IsUploadAvatar  int
	NewMsgNotice    int
	MsgShowDetail   int //显示消息通知详情0.否1.是
	MsgExpireSecond int64
}

func newResp(m *Model) *Resp {
	return &Resp{
		UID:             m.UID,
		Name:            m.Name,
		Zone:            m.Zone,
		Phone:           m.Phone,
		Email:           m.Email,
		IsUploadAvatar:  m.IsUploadAvatar,
		NewMsgNotice:    m.NewMsgNotice,
		MsgShowDetail:   m.MsgShowDetail,
		MsgExpireSecond: m.MsgExpireSecond,
	}
}

// FriendResp 用户好友
type FriendResp struct {
	Name    string
	UID     string
	IsAlone int // 是否为单项好友
}

// FriendReq FriendReq
type FriendReq struct {
	UID     string
	ToUID   string
	Flag    int
	Version int64
}

// AddUserReq  AddUserReq
type AddUserReq struct {
	Name     string
	UID      string // 如果无值，则随机生成
	Username string
	Zone     string
	Phone    string
	Email    string
	Password string
}

type UserUpdateReq struct {
	UID  string
	Name *string
}

type UpdateLoginPasswordReq struct {
	UID         string // 用户uid
	Password    string // 用户旧密码
	NewPassword string // 用户新密码
}

type SettingResp struct {
	UID          string // 用户UID
	Mute         int    // 免打扰
	Top          int    // 置顶
	ChatPwdOn    int    // 是否开启聊天密码
	Screenshot   int    //截屏通知
	RevokeRemind int    //撤回提醒
	Blacklist    int    //黑名单
	Receipt      int    //消息是否回执
	Version      int64  // 版本
}

type OnLineUserResp struct {
	UID         string
	LastOffline int
	Online      int
	DeviceFlag  uint8
}

func toSettingResp(m *SettingModel) *SettingResp {

	return &SettingResp{
		UID:          m.ToUID,
		Mute:         m.Mute,
		Top:          m.Top,
		ChatPwdOn:    m.ChatPwdOn,
		Screenshot:   m.Screenshot,
		RevokeRemind: m.RevokeRemind,
		Blacklist:    m.Blacklist,
		Receipt:      m.Receipt,
		Version:      m.Version,
	}
}

type UserDetailResp struct {
	UID            string            `json:"uid"`
	Name           string            `json:"name"`
	Username       string            `json:"username"`
	Email          string            `json:"email,omitempty"`  // email（仅自己能看）
	Zone           string            `json:"zone,omitempty"`   // 手机区号（仅自己能看）
	Phone          string            `json:"phone,omitempty"`  // 手机号（仅自己能看）
	Mute           int               `json:"mute"`             // 免打扰
	Top            int               `json:"top"`              // 置顶
	Sex            int               `json:"sex"`              //性别1:男
	Category       string            `json:"category"`         //用户分类 '客服'
	ShortNo        string            `json:"short_no"`         // 用户唯一短编号
	ChatPwdOn      int               `json:"chat_pwd_on"`      //是否开启聊天密码
	Screenshot     int               `json:"screenshot"`       //截屏通知
	RevokeRemind   int               `json:"revoke_remind"`    //撤回提醒
	Receipt        int               `json:"receipt"`          //消息是否回执
	Online         int               `json:"online"`           //是否在线
	LastOffline    int               `json:"last_offline"`     //最后一次离线时间
	DeviceFlag     config.DeviceFlag `json:"device_flag"`      // 在线设备标记
	Follow         int               `json:"follow"`           //是否是好友
	BeDeleted      int               `json:"be_deleted"`       // 被删除
	BeBlacklist    int               `json:"be_blacklist"`     // 被拉黑
	Code           string            `json:"code"`             //加好友所需vercode TODO: code不再使用 请使用Vercode
	Vercode        string            `json:"vercode"`          //
	SourceDesc     string            `json:"source_desc"`      // 好友来源
	Remark         string            `json:"remark"`           //好友备注
	IsUploadAvatar int               `json:"is_upload_avatar"` // 是否上传头像
	Status         int               `json:"status"`           //用户状态 1 正常 2:黑名单
	Robot          int               `json:"robot"`            // 机器人0.否1.是
	IsDestroy      int               `json:"is_destroy"`       // 是否注销0.否1.是
	Flame          int               `json:"flame"`            // 是否开启阅后即焚
	FlameSecond    int               `json:"flame_second"`     // 阅后即焚秒数
}

func NewUserDetailResp(m *Detail, remark, loginUID string, sourceFrom string, onLine int, lastOffline int, deviceFlag config.DeviceFlag, follow int, status int, beDeleted int, beBlacklist int, setting *SettingModel, vercode string) *UserDetailResp {
	self := loginUID == m.UID

	email := ""
	phone := ""
	zone := ""
	username := ""
	if self {
		email = m.Email
		phone = m.Phone
		zone = m.Zone
	}
	if m.Robot == 1 {
		username = m.Username
	}
	var flame int
	var flameSecond int
	if setting != nil {
		flame = setting.Flame
		flameSecond = setting.FlameSecond

	}

	return &UserDetailResp{
		UID:            m.UID,
		Name:           m.Name,
		Email:          email,
		Zone:           zone,
		Phone:          phone,
		Mute:           m.Mute,
		Top:            m.Top,
		Sex:            m.Sex,
		ChatPwdOn:      m.ChatPwdOn,
		Category:       m.Category,
		ShortNo:        m.ShortNo,
		Screenshot:     m.Screenshot,
		RevokeRemind:   m.RevokeRemind,
		Receipt:        m.Receipt,
		Online:         onLine,
		LastOffline:    lastOffline,
		DeviceFlag:     deviceFlag,
		Follow:         follow,
		SourceDesc:     sourceFrom,
		Remark:         remark,
		IsUploadAvatar: m.IsUploadAvatar,
		Status:         status,
		Robot:          m.Robot,
		Username:       username,
		BeDeleted:      beDeleted,
		BeBlacklist:    beBlacklist,
		IsDestroy:      m.IsDestroy,
		Flame:          flame,
		FlameSecond:    flameSecond,
		Vercode:        vercode,
	}
}
