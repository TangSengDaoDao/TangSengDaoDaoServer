package webhook

import (
	"github.com/gocraft/dbr/v2"
)

// DB DB
type DB struct {
	session *dbr.Session
}

// NewDB NewDB
func NewDB(session *dbr.Session) *DB {
	return &DB{
		session: session,
	}
}

// GetThirdName 获取三个名字 （常用名字，好友备注，群内名字） （TODO: 此方法不应该直接写sql 应该调用各模块的server来获取数据）
func (db *DB) GetThirdName(fromUID string, toUID string, groupNo string) (string, string, string, error) {
	if fromUID == "" {
		return "", "", "", nil
	}
	var name string        // 常用名
	var remark string      // 好友备注
	var nameInGroup string // 群内备注

	if toUID == "" && groupNo == "" {
		_, err := db.session.Select("name").From("`user`").Where("uid=?", fromUID).Load(&name)
		if err != nil {
			return "", "", "", nil
		}
	} else if toUID != "" && groupNo == "" {
		var nameStruct struct {
			Name   string
			Remark string
		}
		builder := db.session.SelectBySql("select `user`.name,IFNULL(friend.remark,'') remark from `user` left join friend on `user`.uid=friend.to_uid and friend.uid=? where `user`.uid=? ", toUID, fromUID)
		_, err := builder.Load(&nameStruct)
		if err != nil {
			return "", "", "", err
		}
		name = nameStruct.Name
		remark = nameStruct.Remark
	} else if toUID == "" && groupNo != "" {
		var nameStruct struct {
			Name        string
			NameInGroup string
		}
		_, err := db.session.SelectBySql("select `user`.name,IFNULL(group_member.remark,'') name_in_group from `user` left join group_member on group_member.group_no=?  and `user`.uid=group_member.uid and group_member.is_deleted=0 where `user`.uid=? ", groupNo, fromUID).Load(&nameStruct)
		if err != nil {
			return "", "", "", err
		}
		name = nameStruct.Name
		nameInGroup = nameStruct.NameInGroup
	} else if toUID != "" && groupNo != "" {
		var nameStruct struct {
			Name        string
			Remark      string
			NameInGroup string
		}
		_, err := db.session.SelectBySql("select `user`.name,IFNULL(group_member.remark,'') name_in_group,IFNULL(friend.remark ,'') remark from `user` left join group_member on  group_member.group_no=?  and `user`.uid=group_member.uid and group_member.is_deleted=0 left join friend on `user`.uid=friend.to_uid and `user`.uid=? and friend.uid=? where `user`.uid=?", groupNo, fromUID, toUID, fromUID).Load(&nameStruct)
		if err != nil {
			return "", "", "", err
		}
		name = nameStruct.Name
		nameInGroup = nameStruct.NameInGroup
		remark = nameStruct.Remark
	}
	return name, remark, nameInGroup, nil
}

// GetGroupName 获取群名
func (db *DB) GetGroupName(groupNo string) (string, error) {
	var name string
	_, err := db.session.Select("name").From("`group`").Where("group_no=?", groupNo).Load(&name)
	return name, err
}
