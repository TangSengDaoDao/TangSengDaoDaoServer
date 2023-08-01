-- +migrate Up

ALTER TABLE `user_setting` ADD COLUMN `flame`  smallint  not null default 0  COMMENT '阅后即焚是否开启 1.开启 0.未开启';
ALTER TABLE `user_setting` ADD COLUMN `flame_second`  smallint  not null default 0  COMMENT '阅后即焚销毁秒数';