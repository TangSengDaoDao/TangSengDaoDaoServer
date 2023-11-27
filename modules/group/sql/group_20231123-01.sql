-- +migrate Up

ALTER TABLE `group` ADD COLUMN category VARCHAR(40) not null DEFAULT 0 COMMENT '群分类';
