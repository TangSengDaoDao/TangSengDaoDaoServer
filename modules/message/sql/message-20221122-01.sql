-- +migrate Up

ALTER TABLE `reminders` ADD COLUMN `publisher`  varchar(40)    not null default ''  COMMENT '提醒项发布者uid';
