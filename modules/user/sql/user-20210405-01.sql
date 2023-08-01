-- +migrate Up

ALTER TABLE `user` ADD COLUMN email VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'email地址';