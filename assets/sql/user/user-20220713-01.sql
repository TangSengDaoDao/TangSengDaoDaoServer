-- +migrate Up

ALTER TABLE `user` ADD COLUMN wx_openid VARCHAR(100) NOT NULL DEFAULT '' COMMENT '微信openid';
ALTER TABLE `user` ADD COLUMN wx_unionid VARCHAR(100) NOT NULL DEFAULT '' COMMENT '微信unionid';
