
-- +migrate Up
ALTER TABLE `workplace_banner` ADD COLUMN `sort_num` integer not null default 0  COMMENT '排序号';
