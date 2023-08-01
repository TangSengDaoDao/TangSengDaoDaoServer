-- +migrate Up

ALTER TABLE `user` ADD COLUMN app_id VARCHAR(40) NOT NULL DEFAULT '' COMMENT 'app id';