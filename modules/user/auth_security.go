package user

import (
	"encoding/json"
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
)

func (u *User) refreshTokenName(token, name string) error {
	key := u.ctx.GetConfig().Cache.TokenCachePrefix + token
	previous, err := u.ctx.Cache().Get(key)
	if err != nil {
		return err
	}
	info, err := wkhttp.ParseTokenCacheInfo(previous)
	if err != nil {
		return err
	}
	info.Name = name
	updated, err := json.Marshal(info)
	if err != nil {
		return err
	}
	ok, err := u.ctx.GetRedisConn().CompareAndSetPreservingTTL(key, previous, string(updated), u.ctx.GetConfig().Cache.TokenExpire)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("登录凭证已失效，请重新登录")
	}
	return nil
}

func isManagerRole(role string) bool {
	return role == string(wkhttp.Admin) || role == string(wkhttp.SuperAdmin)
}

func checkPublicPasswordRecovery(user *Model) error {
	if user == nil || user.Status != 1 || user.IsDestroy != 0 {
		return errors.New("该账号不可使用找回密码")
	}
	if isManagerRole(user.Role) {
		return errors.New("管理账号不支持通过公共接口找回密码，请联系管理员")
	}
	return nil
}

// Initialize a version for existing privileged accounts without accepting a stale
// password/role snapshot if the account changes concurrently with login.
func (d *DB) encodeLoginToken(uid, name, role, password string) (string, error) {
	if !isManagerRole(role) {
		// Historical/business roles do not grant management privileges. Keep the
		// database role unchanged, but issue a regular application credential.
		return wkhttp.EncodeTokenCacheInfo(uid, name, ""), nil
	}
	_, err := d.session.Update("user").Set("auth_version", util.GenerUUID()).Where(
		"uid=? AND password=? AND role=? AND status=1 AND is_destroy=0 AND auth_version=''", uid, password, role,
	).Exec()
	if err != nil {
		return "", err
	}
	var account struct{ AuthVersion string }
	err = d.session.Select("auth_version").From("user").Where(
		"uid=? AND password=? AND role=? AND status=1 AND is_destroy=0", uid, password, role,
	).LoadOne(&account)
	if err != nil {
		return "", err
	}
	if account.AuthVersion == "" {
		return "", errors.New("登录凭证初始化失败")
	}
	return wkhttp.EncodeUserTokenCacheInfo(uid, name, role, account.AuthVersion), nil
}
