-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN revoke_second smallint not null DEFAULT 0 COMMENT '消息可撤回时长';
ALTER TABLE `app_config` ADD COLUMN welcome_message varchar(2000) not null DEFAULT '' COMMENT '登录欢迎语';
