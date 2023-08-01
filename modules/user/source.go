package user

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/source"
	"go.uber.org/zap"
)

// GetUserByVercode 通过vercode获取用户信息
func (u *User) GetUserByVercode(vercode string) (*source.UserModel, error) {
	if vercode == "" {
		return nil, errors.New("验证码不能为空")
	}
	model, err := u.db.QueryByVercode(vercode)
	if err != nil {
		u.Error("通过验证码获取用户信息错误", zap.Error(err))
		return nil, errors.New("通过验证码获取用户信息错误")
	}
	if model == nil {
		return nil, nil
	}
	return &source.UserModel{
		UID:       model.UID,
		Name:      model.Name,
		Vercode:   model.Vercode,
		QRVercode: model.QRVercode}, nil
}

// GetUserByQRVercode 通过二维码验证码获取用户信息
func (u *User) GetUserByQRVercode(qrvercode string) (*source.UserModel, error) {
	if qrvercode == "" {
		return nil, errors.New("验证码不能为空")
	}

	model, err := u.db.queryByQRVerCode(qrvercode)
	if err != nil {
		u.Error("通过二维码验证码获取用户信息错误", zap.Error(err))
		return nil, errors.New("通过二维码验证码获取用户信息错误")
	}
	if model == nil {
		return nil, nil
	}
	return &source.UserModel{
		UID:       model.UID,
		Name:      model.Name,
		Vercode:   model.Vercode,
		QRVercode: model.QRVercode}, nil
}

// GetFriendByVercode 通过vercode获取好友信息
func (u *User) GetFriendByVercode(vercode string) (*source.FriendModel, error) {
	if vercode == "" {
		return nil, errors.New("验证码不能为空")
	}
	model, err := u.friendDB.queryWithVercode(vercode)
	if err != nil {
		u.Error("通过vercode查询好友信息错误", zap.Error(err))
		return nil, errors.New("通过vercode查询好友信息错误")
	}
	if model == nil {
		return nil, errors.New("验证码错误")
	}
	return &source.FriendModel{
		UID:     model.UID,
		ToUID:   model.ToUID,
		Vercode: model.Vercode,
	}, nil
}

func (u *User) GetFriendByVercodes(vercodes []string) ([]*source.FriendModel, error) {
	if len(vercodes) == 0 {
		return nil, errors.New("验证码不能为空")
	}
	models, err := u.friendDB.queryWithVercodes(vercodes)
	if err != nil {
		u.Error("通过vercode查询好友信息错误", zap.Error(err))
		return nil, errors.New("通过vercode查询好友信息错误")
	}
	if models == nil {
		return nil, nil
	}
	friends := make([]*source.FriendModel, 0)
	for _, model := range models {
		friends = append(friends, &source.FriendModel{
			UID:     model.UID,
			ToUID:   model.ToUID,
			Vercode: model.Vercode,
			Name:    model.Name,
		})
	}
	return friends, nil
}

// GetUserByUID 通过UID获取用户信息
func (u *User) GetUserByUID(uid string) (*source.UserModel, error) {
	if uid == "" {
		return nil, errors.New("uid不能为空")
	}
	model, err := u.db.QueryByUID(uid)
	if err != nil {
		u.Error("通过Uid查询用户信息错误", zap.Error(err))
		return nil, errors.New("通过Uid查询用户信息错误")
	}
	if model == nil {
		return nil, errors.New("用户不存在")
	}
	return &source.UserModel{
		UID:     model.UID,
		Name:    model.Name,
		Vercode: model.Vercode,
	}, err
}

// 通过通讯录验证码获取用户信息
func (u *User) GetUserByMailListVercode(vercode string) (*source.UserModel, error) {
	if vercode == "" {
		return nil, errors.New("验证码不能为空")
	}
	model, err := u.maillistDB.queryWitchVercode(vercode)
	if err != nil {
		u.Error("通过通讯验证码查询通讯录联系人错误", zap.Error(err))
		return nil, err
	}
	if model == nil {
		return nil, errors.New("验证码错误")
	}
	user, err := u.db.QueryByPhone(model.Zone, model.Phone)
	if err != nil {
		u.Error("通过手机号查询用户错误", zap.Error(err))
		return nil, err
	}
	if user == nil {
		return nil, errors.New("该手机号未注册")
	}
	return &source.UserModel{
		Name:            user.Name,
		UID:             user.UID,
		QRVercode:       user.QRVercode,
		Vercode:         user.Vercode,
		MailListVercode: model.Vercode,
	}, nil
}
