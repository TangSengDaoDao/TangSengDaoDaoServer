-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN search_by_phone smallint not null DEFAULT 0 COMMENT '是否可通过手机号搜索';
