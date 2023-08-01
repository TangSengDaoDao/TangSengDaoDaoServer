package group

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"go.uber.org/zap"
)

// GetGroupMemberByVercode 通过vercode获取群成员
func (g *Group) GetGroupMemberByVercode(vercode string) (*source.GroupMember, error) {
	if vercode == "" {
		return nil, errors.New("vercode不能为空")
	}
	model, err := g.db.queryMemberWithVercode(vercode)
	if err != nil {
		g.Error("通过vercode查询群成员错误", zap.Error(err))
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return &source.GroupMember{UID: model.UID, GroupNo: model.GroupNo, Vercode: model.Vercode}, nil
}

func (g *Group) GetGroupMemberByVercodes(vercodes []string) ([]*source.GroupMember, error) {
	if len(vercodes) == 0 {
		return nil, errors.New("vercodes不能为空")
	}
	models, err := g.db.queryMemberWithVercodes(vercodes)
	if err != nil {
		g.Error("通过vercodes查询群成员错误", zap.Error(err))
		return nil, err
	}
	if models == nil {
		return nil, nil
	}
	members := make([]*source.GroupMember, 0, len(models))
	for _, model := range models {
		members = append(members, &source.GroupMember{
			UID:     model.UID,
			GroupNo: model.GroupNo,
			Vercode: model.Vercode,
			Name:    model.GroupName,
		})
	}
	return members, nil
}

func (g *Group) GetGroupMemberByUID(uid string, groupNo string) (*source.GroupMember, error) {
	if uid == "" {
		return nil, errors.New("uid不能为空")
	}
	if groupNo == "" {
		return nil, errors.New("群编号不能为空")
	}
	model, err := g.db.QueryMemberWithUID(uid, groupNo)
	if err != nil {
		g.Error("通过用户ID查询群成员错误", zap.Error(err))
		return nil, err
	}
	if model == nil {
		return nil, nil
	}
	return &source.GroupMember{UID: model.UID, GroupNo: model.GroupNo, Vercode: model.Vercode, Role: model.Role}, nil

}

// GetGroupByGroupNo 查询群详情
func (g *Group) GetGroupByGroupNo(groupNo string) (*source.GroupModel, error) {
	if groupNo == "" {
		return nil, errors.New("群ID不能为空")
	}
	model, err := g.db.QueryWithGroupNo(groupNo)
	if err != nil {
		g.Error("查询群编号错误", zap.Error(err))
		return nil, err
	}
	if model == nil {
		return nil, errors.New("群不存在")
	}
	return &source.GroupModel{
		Name:               model.Name,
		GroupNo:            model.GroupNo,
		ForbiddenAddFriend: model.ForbiddenAddFriend,
	}, nil
}
