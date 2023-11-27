package group

import (
	"errors"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"go.uber.org/zap"
)

// IService 群相关
type IService interface {
	// 获取群总数
	GetAllGroupCount() (int64, error)
	// 查询某天的新建群数量
	GetCreatedCountWithDate(date string) (int64, error)
	// 添加一个群
	AddGroup(model *AddGroupReq) error
	// 某个时间段的建群数据
	GetGroupWithDateSpace(startDate, endDate string) (map[string]int64, error)
	// 查询某个群信息
	GetGroupWithGroupNo(groupNo string) (*InfoResp, error)
	// GetGroups 获取群集合
	GetGroups(groupNos []string) ([]*InfoResp, error)
	// 获取某一批群与指定用户的详情（包括用户对群的设置等等）
	GetGroupDetails(groupNos []string, uid string) ([]*GroupResp, error)
	// 获取群详情
	GetGroupDetail(groupNo string, uid string) (*GroupResp, error)

	// -------------------- 群设置 --------------------
	// GetSettings 获取群的设置
	GetSettings(groupNos []string, uid string) ([]*SettingResp, error)
	// GetSettingsWithUids 获取一批用户对某个群的设置
	GetSettingsWithUIDs(groupNo string, uids []string) ([]*SettingResp, error)

	// -------------------- 群成员 --------------------
	// 获取指定群的群成员列表
	GetMembers(groupNo string) ([]*MemberResp, error)
	// 获取黑明单成员uid集合
	GetBlacklistMemberUIDs(groupNo string) ([]string, error)
	// 查询管理员成员uid列表（包括创建者）
	GetMemberUIDsOfManager(groupNo string) ([]string, error)
	// 是否是创建者或管理者
	IsCreatorOrManager(groupNo string, uid string) (bool, error)
	// 获取成员总数量和在线数量
	// 第一个返回参数为成员总数量
	// 第二个返回参数为在线数量
	GetMemberTotalAndOnlineCount(groupNo string) (int, int, error)
	// 是否存在群成员
	ExistMember(groupNo string, uid string) (bool, error)
	// 成员是否在某群里存在 返回对应在群里的群编号
	ExistMembers(groupNos []string, uid string) ([]string, error)
	// GetGroupsWithMemberUID 获取某个用户的所有群
	GetGroupsWithMemberUID(uid string) ([]*InfoResp, error)
	// 获取指定群的群成员的最大数据版本
	GetGroupMemberMaxVersion(groupNo string) (int64, error)
	// 获取用户所有超级群信息
	GetUserSupers(uid string) ([]*InfoResp, error)
	// 新增群成员
	AddMember(model *AddMemberReq) error
}

// Service Service
type Service struct {
	ctx       *config.Context
	db        *DB
	managerDB *managerDB
	log.Log
	settingDB *settingDB
}

// NewService NewService
func NewService(ctx *config.Context) IService {
	return &Service{
		ctx:       ctx,
		db:        NewDB(ctx),
		managerDB: newManagerDB(ctx.DB()),
		Log:       log.NewTLog("groupService"),
		settingDB: newSettingDB(ctx),
	}
}

// GetAllGroupCount 获取群总数
func (s *Service) GetAllGroupCount() (int64, error) {
	return s.db.queryGroupCount()
}

// GetCreatedCountWithDate 获取某天的新建群数量
func (s *Service) GetCreatedCountWithDate(date string) (int64, error) {
	if date == "" {
		return 0, errors.New("时间不能为空")
	}
	return s.db.queryCreatedCountWithDate(date)
}

// AddGroup 添加一个群
func (s *Service) AddGroup(model *AddGroupReq) error {
	err := s.db.Insert(&Model{
		GroupNo: model.GroupNo,
		Name:    model.Name,
	})
	return err
}

func (s *Service) GetGroupsWithMemberUID(uid string) ([]*InfoResp, error) {
	groups, err := s.db.queryGroupsWithMemberUID(uid)
	if err != nil {
		return nil, err
	}
	infos := make([]*InfoResp, 0, len(groups))
	if len(groups) > 0 {
		for _, gp := range groups {
			infos = append(infos, toInfoResp(gp))
		}
	}
	return infos, nil
}

// GetGroupWithDateSpace 某个时间段的建群数据
func (s *Service) GetGroupWithDateSpace(startDate, endDate string) (map[string]int64, error) {
	if startDate == "" || endDate == "" {
		return nil, errors.New("时间不能为空")
	}
	list, err := s.managerDB.queryRegisterCountWithDateSpace(startDate, endDate)
	if err != nil {
		s.Error("查询群列表错误", zap.Error(err))
		return nil, err
	}
	result := make(map[string]int64)
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

// GetGroupWithGroupNo 查询一个群信息
func (s *Service) GetGroupWithGroupNo(groupNo string) (*InfoResp, error) {
	if groupNo == "" {
		return nil, errors.New("群编号不能为空")
	}
	group, err := s.db.QueryWithGroupNo(groupNo)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, errors.New("不存在此群")
	}
	return toInfoResp(group), nil
}

func (s *Service) GetGroupDetails(groupNos []string, uid string) ([]*GroupResp, error) {
	groupDetails, err := s.db.QueryDetailWithGroupNos(groupNos, uid)
	if err != nil {
		return nil, err
	}
	groupResps := make([]*GroupResp, 0)
	if len(groupDetails) > 0 {
		for _, groupDetail := range groupDetails {
			groupResp := &GroupResp{}
			groupResps = append(groupResps, groupResp.from(groupDetail))
		}
	}
	return groupResps, nil
}

func (s *Service) GetGroupDetail(groupNo string, uid string) (*GroupResp, error) {
	groupDetailModel, err := s.db.QueryDetailWithGroupNo(groupNo, uid)
	if err != nil {
		s.Error("查询群信息失败！", zap.Error(err))
		return nil, errors.New("查询群信息失败！")
	}
	if groupDetailModel == nil {
		return nil, nil
	}
	memberCount, onlineCount, err := s.GetMemberTotalAndOnlineCount(groupNo)
	if err != nil {
		s.Error("查询成员数量和在线数量失败！")
		return nil, err
	}
	memberOfMe, err := s.db.QueryMemberWithUID(uid, groupNo)
	if err != nil {
		s.Error("查询成员失败！", zap.Error(err))
		return nil, err
	}
	quit := 0
	if memberOfMe == nil {
		quit = 1
	}
	groupResp := &GroupResp{}
	groupResp = groupResp.from(groupDetailModel)
	groupResp.MemberCount = memberCount
	groupResp.OnlineCount = onlineCount
	groupResp.Quit = quit
	if memberOfMe != nil {
		groupResp.Role = memberOfMe.Role
		groupResp.ForbiddenExpirTime = memberOfMe.ForbiddenExpirTime
	}
	return groupResp, nil
}

func (s *Service) GetGroups(groupNos []string) ([]*InfoResp, error) {
	groups, err := s.db.QueryWithGroupNos(groupNos)
	if err != nil {
		return nil, err
	}

	if len(groups) == 0 {
		return nil, nil
	}
	infoResps := make([]*InfoResp, 0, len(groups))
	for _, group := range groups {
		infoResps = append(infoResps, toInfoResp(group))
	}
	return infoResps, nil
}

func (s *Service) GetUserSupers(uid string) ([]*InfoResp, error) {
	groups, err := s.db.queryUserSupers(uid)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, nil
	}
	infoResps := make([]*InfoResp, 0, len(groups))
	for _, group := range groups {
		infoResps = append(infoResps, toInfoResp(group))
	}
	return infoResps, nil
}

func (s *Service) AddMember(model *AddMemberReq) error {
	err := s.db.InsertMember(&MemberModel{
		GroupNo: model.GroupNo,
		UID:     model.MemberUID,
	})
	return err
}
func (s *Service) GetGroupMemberMaxVersion(groupNo string) (int64, error) {
	version, err := s.db.queryGroupMemberMaxVersion(groupNo)
	return version, err
}

func (s *Service) GetMembers(groupNo string) ([]*MemberResp, error) {
	memberDetails, err := s.db.queryMembersWithGroupNo(groupNo)
	if err != nil {
		return nil, err
	}
	memberResps := make([]*MemberResp, 0, len(memberDetails))
	if len(memberDetails) > 0 {
		for _, memberDetail := range memberDetails {
			memberResps = append(memberResps, newMemberResp(memberDetail))
		}
	}
	return memberResps, nil
}

func (s *Service) GetBlacklistMemberUIDs(groupNo string) ([]string, error) {
	uids, err := s.db.queryBlacklistMemberUIDsWithGroupNo(groupNo)
	if err != nil {
		return nil, err
	}
	return uids, nil
}

func (s *Service) GetMemberUIDsOfManager(groupNo string) ([]string, error) {
	return s.db.QueryGroupManagerOrCreatorUIDS(groupNo)
}

func (s *Service) IsCreatorOrManager(groupNo string, uid string) (bool, error) {
	return s.db.QueryIsGroupManagerOrCreator(groupNo, uid)
}

func (s *Service) GetMemberTotalAndOnlineCount(groupNo string) (int, int, error) {
	var onlineCount, memberCount int64
	var err error
	memberCount, err = s.db.QueryMemberCount(groupNo)
	if err != nil {
		return 0, 0, err
	}
	onlineCount, err = s.db.queryMemberOnlineCount(groupNo)
	if err != nil {
		return 0, 0, err
	}
	return int(memberCount), int(onlineCount), nil
}

func (s *Service) ExistMember(groupNo string, uid string) (bool, error) {
	return s.db.ExistMember(uid, groupNo)
}

func (s *Service) ExistMembers(groupNos []string, uid string) ([]string, error) {
	return s.db.existMembers(groupNos, uid)
}

func (s *Service) GetSettings(groupNos []string, uid string) ([]*SettingResp, error) {
	settings, err := s.settingDB.QuerySettings(groupNos, uid)
	if err != nil {
		return nil, err
	}
	resps := make([]*SettingResp, 0, len(settings))
	if len(settings) > 0 {
		for _, setting := range settings {
			resps = append(resps, toSettingResp(setting))
		}
	}
	return resps, nil
}

// GetSettingsWithUIDs 查询一批用户对某个群的设置
func (s *Service) GetSettingsWithUIDs(groupNo string, uids []string) ([]*SettingResp, error) {
	settings, err := s.settingDB.QuerySettingsWithUIDs(groupNo, uids)
	if err != nil {
		return nil, err
	}
	resps := make([]*SettingResp, 0, len(settings))
	if len(settings) > 0 {
		for _, setting := range settings {
			resps = append(resps, toSettingResp(setting))
		}
	}
	return resps, nil
}

// AddGroupReq 添加群
type AddGroupReq struct {
	GroupNo string
	Name    string
}

// AddMemberReq 添加群成员
type AddMemberReq struct {
	GroupNo   string
	MemberUID string
}

// InfoResp 群信息
type InfoResp struct {
	GroupNo             string    `json:"group_no"`               // 群编号
	GroupType           GroupType `json:"group_type"`             // 群类型
	Name                string    `json:"name"`                   // 群名称
	Notice              string    `json:"notice"`                 // 群公告
	Creator             string    `json:"creator"`                // 创建者uid
	Status              int       `json:"status"`                 // 群状态
	Forbidden           int       `json:"forbidden"`              // 是否全员禁言
	Invite              int       `json:"invite"`                 // 是否开启邀请确认 0.否 1.是
	ForbiddenAddFriend  int       `json:"forbidden_add_friend"`   //群内禁止加好友
	AllowViewHistoryMsg int       `json:"allow_view_history_msg"` // 是否允许新成员查看历史记录
	CreatedAt           string    `json:"created_at"`
	UpdatedAt           string    `json:"updated_at"`
	Version             int64     `json:"version"` // 群数据版本
}

func toInfoResp(m *Model) *InfoResp {
	return &InfoResp{
		GroupNo:             m.GroupNo,
		GroupType:           GroupType(m.GroupType),
		Name:                m.Name,
		Notice:              m.Notice,
		Creator:             m.Creator,
		Status:              m.Status,
		Forbidden:           m.Forbidden,
		Invite:              m.Invite,
		ForbiddenAddFriend:  m.ForbiddenAddFriend,
		AllowViewHistoryMsg: m.AllowViewHistoryMsg,
		CreatedAt:           m.CreatedAt.String(),
		UpdatedAt:           m.UpdatedAt.String(),
		Version:             m.Version,
	}
}

type MemberResp struct {
	GroupNo string // 群编号
	UID     string // 成员uid
	Name    string // 群成员名称
	Remark  string // 成员备注
	Role    int    // 成员角色
	Version int64
	Vercode string //验证码
}

func newMemberResp(m *MemberDetailModel) *MemberResp {
	return &MemberResp{
		GroupNo: m.GroupNo,
		UID:     m.UID,
		Name:    m.Name,
		Remark:  m.Remark,
		Role:    m.Role,
		Version: m.Version,
		Vercode: m.Vercode,
	}
}

// SettingResp 群设置
type SettingResp struct {
	UID             string
	GroupNo         string // 群编号
	Mute            int    // 免打扰
	Top             int    // 置顶
	ShowNick        int    // 显示昵称
	Save            int    // 是否保存
	ChatPwdOn       int    //是否开启聊天密码
	Screenshot      int    //截屏通知
	RevokeRemind    int    //撤回通知
	JoinGroupRemind int    //进群提醒
	Receipt         int    //消息是否回执
	Remark          string // 群备注
	Version         int64  // 版本
}

func toSettingResp(m *Setting) *SettingResp {
	return &SettingResp{
		GroupNo:         m.GroupNo,
		Mute:            m.Mute,
		Top:             m.Top,
		ShowNick:        m.ShowNick,
		Save:            m.Save,
		ChatPwdOn:       m.ChatPwdOn,
		Screenshot:      m.Screenshot,
		RevokeRemind:    m.RevokeRemind,
		JoinGroupRemind: m.JoinGroupRemind,
		Receipt:         m.Receipt,
		Remark:          m.Remark,
		Version:         m.Version,
		UID:             m.UID,
	}
}

type GroupResp struct {
	GroupNo             string    `json:"group_no"`               // 群编号
	GroupType           GroupType `json:"group_type"`             // 群类型
	Category            string    `json:"category"`               // 群分类
	Name                string    `json:"name"`                   // 群名称
	Remark              string    `json:"remark"`                 // 群备注
	Notice              string    `json:"notice"`                 // 群公告
	Mute                int       `json:"mute"`                   // 免打扰
	Top                 int       `json:"top"`                    // 置顶
	ShowNick            int       `json:"show_nick"`              // 显示昵称
	Save                int       `json:"save"`                   // 是否保存
	Forbidden           int       `json:"forbidden"`              // 是否全员禁言
	Invite              int       `json:"invite"`                 // 群聊邀请确认
	ChatPwdOn           int       `json:"chat_pwd_on"`            //是否开启聊天密码
	Screenshot          int       `json:"screenshot"`             //截屏通知
	RevokeRemind        int       `json:"revoke_remind"`          //撤回提醒
	JoinGroupRemind     int       `json:"join_group_remind"`      //进群提醒
	ForbiddenAddFriend  int       `json:"forbidden_add_friend"`   //群内禁止加好友
	Status              int       `json:"status"`                 //群状态
	Receipt             int       `json:"receipt"`                //消息是否回执
	Flame               int       `json:"flame"`                  // 阅后即焚
	FlameSecond         int       `json:"flame_second"`           // 阅后即焚秒数
	AllowViewHistoryMsg int       `json:"allow_view_history_msg"` // 是否允许新成员查看历史消息
	MemberCount         int       `json:"member_count"`           // 成员数量
	OnlineCount         int       `json:"online_count"`           // 在线数量
	Quit                int       `json:"quit"`                   // 我是否已退出群聊
	Role                int       `json:"role"`                   // 我在群聊里的角色
	ForbiddenExpirTime  int64     `json:"forbidden_expir_time"`   // 我在此群的禁言过期时间
	CreatedAt           string    `json:"created_at"`
	UpdatedAt           string    `json:"updated_at"`
	Version             int64     `json:"version"` // 群数据版本
}

func (g *GroupResp) from(model *DetailModel) *GroupResp {
	return &GroupResp{
		GroupNo:             model.GroupNo,
		GroupType:           GroupType(model.GroupType),
		Category:            model.Category,
		Name:                model.Name,
		Notice:              model.Notice,
		Mute:                model.Mute,
		Top:                 model.Top,
		ShowNick:            model.ShowNick,
		Save:                model.Save,
		Remark:              model.Remark,
		Version:             model.Version,
		Forbidden:           model.Forbidden,
		Invite:              model.Invite,
		ChatPwdOn:           model.ChatPwdOn,
		Screenshot:          model.Screenshot,
		RevokeRemind:        model.RevokeRemind,
		JoinGroupRemind:     model.JoinGroupRemind,
		ForbiddenAddFriend:  model.ForbiddenAddFriend,
		Receipt:             model.Receipt,
		Flame:               model.Flame,
		FlameSecond:         model.FlameSecond,
		Status:              model.Status,
		AllowViewHistoryMsg: model.AllowViewHistoryMsg,
		CreatedAt:           model.CreatedAt.String(),
		UpdatedAt:           model.UpdatedAt.String(),
	}
}
