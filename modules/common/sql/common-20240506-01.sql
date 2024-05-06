-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN register_user_must_complete_info_on smallint not null DEFAULT 0 COMMENT '注册用户是否必须完善信息';
