-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN new_user_join_system_group smallint not null DEFAULT 1 COMMENT '注册用户是否默认加入系统群';
