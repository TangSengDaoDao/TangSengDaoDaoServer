-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN can_modify_api_url smallint not null DEFAULT 0 COMMENT '是否能修改服务器地址';
