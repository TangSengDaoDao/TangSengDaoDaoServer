-- +migrate Up

ALTER TABLE `group` ADD COLUMN group_type smallint not null DEFAULT 0 COMMENT '群类型 0.普通群 1.超大群';
