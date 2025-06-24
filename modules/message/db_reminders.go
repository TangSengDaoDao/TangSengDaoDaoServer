package message

import (
	"errors"
	"fmt"
	"sort"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
	"go.uber.org/zap"
)

type remindersDB struct {
	ctx     *config.Context
	session *dbr.Session
}

func newRemindersDB(ctx *config.Context) *remindersDB {
	return &remindersDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (r *remindersDB) inserts(models []*remindersModel) error {
	tx, err := r.session.Begin()
	if err != nil {
		return errors.New("开启事物错误")
	}
	defer func() {
		if err := recover(); err != nil {
			tx.RollbackUnlessCommitted()
			panic(err)
		}
	}()
	for _, m := range models {
		_, err := tx.InsertInto("reminders").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *remindersDB) deleteWithChannel(channelID string, channelType uint8, messageID int64, version int64) error {
	_, err := r.session.Update("reminders").Set("is_deleted", 1).Set("version", version).Where("channel_id=? and channel_type=? and message_id=?", channelID, channelType, messageID).Exec()
	return err
}

func (r *remindersDB) deleteWithChannelAndUIDTx(channelID string, channelType uint8, uid string, messageID int64, version int64, tx *dbr.Tx) error {
	_, err := tx.Update("reminders").Set("is_deleted", 1).Set("version", version).Where("channel_id=? and channel_type=? and uid=? and message_id=?", channelID, channelType, uid, messageID).Exec()
	return err
}
func (r *remindersDB) queryWithUIDAndChannel(uid string, channelID string, channelType uint8, messageSeq uint32) ([]*remindersDetailModel, error) {
	var list []*remindersDetailModel
	builder := r.session.Select("reminders.*,IF(reminder_done.id is null and reminders.is_deleted=0,0,1) done").From("reminders").LeftJoin("reminder_done", fmt.Sprintf("reminders.id=reminder_done.reminder_id and reminder_done.uid='%s'", uid))
	_, err := builder.Where("(reminders.uid=?  or  ( reminders.uid='' and reminders.channel_id=? and reminders.channel_type=?))  and reminders.message_seq<=? and reminder_done.id is null", uid, channelID, channelType, messageSeq).Load(&list)
	return list, err
}

/*
*
同步提醒项
@param uid 当前登录用户的uid
@param version 以uid为key的增量版本号
@param limit 数据限制
@param channelIDs 频道集合 查询以频道为目标的提醒项
*
*/
func (r *remindersDB) sync(uid string, version int64, limit uint64, channelIDs []string) ([]*remindersDetailModel, error) {
	var models []*remindersDetailModel
	var err error
	if version == 0 {
		builder := r.session.Select("reminders.*,IF(reminder_done.id is null and reminders.is_deleted=0,0,1) done").From("reminders").LeftJoin("reminder_done", fmt.Sprintf("reminders.id=reminder_done.reminder_id and reminder_done.uid='%s'", uid))

		if len(channelIDs) == 0 {
			_, err = builder.Where("(reminders.uid=?  or   reminders.uid='')  and reminders.version>? and reminder_done.id is null", uid, version).OrderAsc("version").Limit(limit).Load(&models)
		} else {
			_, err = builder.Where("(reminders.uid=?  or  ( reminders.uid='' and reminders.channel_id in ?))  and reminders.version>? and reminder_done.id is null", uid, channelIDs, version).OrderAsc("version").Limit(limit).Load(&models)
		}
	} else {
		build := r.session.Select("reminders.*,IF(reminder_done.id is null and reminders.is_deleted=0,0,1) done").From("reminders").LeftJoin("reminder_done", fmt.Sprintf("reminders.id=reminder_done.reminder_id and reminder_done.uid='%s'", uid))
		if len(channelIDs) == 0 {
			_, err = build.Where("(reminders.uid=?  or  reminders.uid='')  and reminders.version>?", uid, version).OrderAsc("version").Limit(limit).Load(&models)
		} else {
			_, err = build.Where("(reminders.uid=?  or  ( reminders.uid='' and reminders.channel_id in ?))  and reminders.version>?", uid, channelIDs, version).OrderAsc("version").Limit(limit).Load(&models)
		}

	}
	return models, err
}

func (r *remindersDB) insertDonesTx(ids []int64, uid string, tx *dbr.Tx) error {
	if len(ids) == 0 {
		return nil
	}

	// 对 reminder_id 进行排序，确保事务按相同顺序获取锁，避免死锁
	sortedIds := make([]int64, len(ids))
	copy(sortedIds, ids)
	sort.Slice(sortedIds, func(i, j int) bool {
		return sortedIds[i] < sortedIds[j]
	})

	// 使用批量插入来减少锁持有时间
	if len(sortedIds) > 1 {
		return r.batchInsertDonesTx(sortedIds, uid, tx)
	}

	// 单个插入
	_, err := tx.InsertBySql("insert ignore into reminder_done(reminder_id,uid) values(?,?)", sortedIds[0], uid).Exec()
	if err != nil {
		r.ctx.Error("insertDonesTx failed", zap.Error(err), zap.Int64("reminder_id", sortedIds[0]), zap.String("uid", uid))
		return err
	}
	return nil
}

// 批量插入方法，减少锁持有时间
func (r *remindersDB) batchInsertDonesTx(sortedIds []int64, uid string, tx *dbr.Tx) error {
	// 构建批量插入SQL
	valueStrings := make([]string, 0, len(sortedIds))
	valueArgs := make([]any, 0, len(sortedIds)*2)

	for _, id := range sortedIds {
		valueStrings = append(valueStrings, "(?,?)")
		valueArgs = append(valueArgs, id, uid)
	}

	sql := fmt.Sprintf("insert ignore into reminder_done(reminder_id,uid) values %s", valueStrings[0])
	for i := 1; i < len(valueStrings); i++ {
		sql += "," + valueStrings[i]
	}

	_, err := tx.InsertBySql(sql, valueArgs...).Exec()
	if err != nil {
		r.ctx.Error("batchInsertDonesTx failed", zap.Error(err), zap.String("uid", uid), zap.Int("count", len(sortedIds)))
		return err
	}
	return nil
}

func (r *remindersDB) updateVersionTx(version int64, id int64, tx *dbr.Tx) error {
	_, err := tx.Update("reminders").Set("version", version).Where("id=?", id).Exec()
	return err
}

type remindersDetailModel struct {
	Done int
	remindersModel
}

type remindersModel struct {
	ChannelID    string
	ChannelType  uint8
	ClientMsgNo  string
	MessageSeq   uint32
	MessageID    string
	ReminderType int
	Publisher    string
	UID          string
	Text         string
	Data         string
	IsLocate     int
	Version      int64
	IsDeleted    int
	db.BaseModel
}
