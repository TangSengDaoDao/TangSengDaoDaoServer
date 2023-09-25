-- +migrate Up

ALTER TABLE `group_setting` ADD COLUMN msg_auto_delete bigint not null DEFAULT -1 COMMENT '-1:未设置 0:关闭 其他为定时删除时间';
