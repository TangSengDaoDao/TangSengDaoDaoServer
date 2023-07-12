-- +migrate Up

ALTER TABLE `conversation_extra` ADD COLUMN draft varchar(1000) not null default ''  COMMENT '草稿';
ALTER TABLE `conversation_extra` ADD COLUMN `version` bigint   not null default 0  COMMENT '数据版本';