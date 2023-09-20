-- +migrate Up

ALTER TABLE `channel_setting` ADD COLUMN msg_auto_delete_at integer not null DEFAULT 0 COMMENT '消息定时删除时间';