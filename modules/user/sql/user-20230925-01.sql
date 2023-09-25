
-- +migrate Up

ALTER TABLE `user_setting` ADD COLUMN msg_auto_delete bigint NOT NULL DEFAULT 0 COMMENT '消息定时删除时长(单位秒)';
