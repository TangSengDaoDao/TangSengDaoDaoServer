-- +migrate Up

ALTER TABLE `app_version` ADD COLUMN `signature` varchar(1000) not null DEFAULT '' COMMENT '二进制包的签名';
