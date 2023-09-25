-- +migrate Up

-- 修改channel_setting表
ALTER TABLE `channel_setting` ADD COLUMN msg_auto_delete integer not null DEFAULT 0 COMMENT '消息定时删除时间';

-- 修改channel_id字段的长度
ALTER TABLE `channel_setting` MODIFY COLUMN channel_id VARCHAR(80);

