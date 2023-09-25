
-- +migrate Up

ALTER TABLE `user` ADD COLUMN msg_expire_second bigint NOT NULL DEFAULT 0 COMMENT '消息过期时长(单位秒)';
