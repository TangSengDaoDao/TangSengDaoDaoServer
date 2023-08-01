package message

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type messageReactionDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newMessageReactionDB(ctx *config.Context) *messageReactionDB {
	return &messageReactionDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

// 查询某个频道的回应数据
func (d *messageReactionDB) queryReactionWithChannelAndSeq(channelID string, channelType uint8, seq int64, limit uint64) ([]*reactionModel, error) {
	var list []*reactionModel
	var err error
	if seq <= 0 { // TODO: 如果seq为0 不能去同步整个频道的 应该同步最新指定数量的回应数据（建议limit 100）
		_, err = d.session.Select("*").From("reaction_users").Where("channel_id=? and channel_type=?", channelID, channelType).OrderDesc("seq").Limit(limit).Load(&list)
	} else {
		_, err = d.session.Select("*").From("reaction_users").Where("channel_id=? and channel_type=? and seq>?", channelID, channelType, seq).OrderAsc("seq").Limit(limit).Load(&list)
	}
	return list, err
}

func (d *messageReactionDB) queryWithMessageIDs(messageIDs []string) ([]*reactionModel, error) {
	if len(messageIDs) <= 0 {
		return nil, nil
	}
	var models []*reactionModel
	_, err := d.session.Select("*").From("reaction_users").Where("message_id in ?", messageIDs).Load(&models)
	return models, err
}

// 查询某个用户的回应数据
func (d *messageReactionDB) queryReactionWithUIDAndMessageID(uid string, messageID string) (*reactionModel, error) {
	var model *reactionModel
	_, err := d.session.Select("*").From("reaction_users").Where("uid=? and message_id=?", uid, messageID).Load(&model)
	return model, err
}

// 新增回应
func (d *messageReactionDB) insertReaction(model *reactionModel) error {
	_, err := d.session.InsertInto("reaction_users").Columns(util.AttrToUnderscore(model)...).Record(model).Exec()
	return err
}

// 修改某条消息的回应
func (d *messageReactionDB) updateReactionStatus(model *reactionModel) error {
	_, err := d.session.Update("reaction_users").SetMap(map[string]interface{}{
		"is_deleted": model.IsDeleted,
		"seq":        model.Seq,
		"emoji":      model.Emoji,
	}).Where("message_id=? and uid=?", model.MessageID, model.UID).Exec()
	return err
}
func (d *messageReactionDB) updateReactionText(model *reactionModel) error {
	_, err := d.session.Update("reaction_users").SetMap(map[string]interface{}{
		"is_deleted": model.IsDeleted,
		"seq":        model.Seq,
	}).Where("message_id=? and uid=? and emoji=?", model.MessageID, model.UID, model.Emoji).Exec()
	return err
}

type reactionModel struct {
	MessageID   string // 消息唯一ID
	Seq         int64  // 回复序列号
	ChannelID   string // 频道唯一ID
	ChannelType uint8  // 频道类型
	UID         string // 用户ID
	Name        string // 用户名称
	Emoji       string // 回应表情
	IsDeleted   int    // 是否已删除
	db.BaseModel
}
