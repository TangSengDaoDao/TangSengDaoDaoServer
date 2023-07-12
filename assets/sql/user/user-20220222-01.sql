-- +migrate Up

ALTER TABLE `user` ADD COLUMN is_destroy smallint not null default 0 COMMENT '是否已销毁';
ALTER TABLE `user` MODIFY COLUMN zone VARCHAR(20);
ALTER TABLE `user` MODIFY COLUMN phone VARCHAR(100);
