-- +migrate Up

ALTER TABLE `reminders` ADD COLUMN `client_msg_no`  varchar(40)    not null default ''  COMMENT '消息client msg no';
ALTER TABLE `reminders` ADD COLUMN `is_deleted`  smallint  not null default 0  COMMENT '是否被删除';