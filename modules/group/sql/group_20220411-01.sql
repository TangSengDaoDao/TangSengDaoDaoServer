-- +migrate Up

ALTER TABLE `group` ADD COLUMN avatar VARCHAR(255) NOT NULL DEFAULT '' COMMENT '群头像';
