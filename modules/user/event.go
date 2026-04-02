package user

import (
	"go.uber.org/zap"
)

// 添加系统账号到IM
func (u *User) AddSystemUids() {
	uids := []string{u.ctx.GetConfig().Account.SystemUID, u.ctx.GetConfig().Account.FileHelperUID}
	_, err := u.ctx.AddSystemUids(uids)
	if err != nil {
		u.Error("添加系统账号到IM错误", zap.Error(err))
	}
}
