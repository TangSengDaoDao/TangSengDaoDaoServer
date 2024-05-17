-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN register_invite_on smallint not null DEFAULT 0 COMMENT '是否开启注册邀请';
ALTER TABLE `app_config` ADD COLUMN send_welcome_message_on smallint not null DEFAULT 1 COMMENT '是否开启登录欢迎语';
ALTER TABLE `app_config` ADD COLUMN invite_system_account_join_group_on smallint not null DEFAULT 0 COMMENT '是否开启系统账号进入群聊';
