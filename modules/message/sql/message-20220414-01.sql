-- +migrate Up

ALTER TABLE `message_extra` ADD COLUMN content_edit TEXT COMMENT '编辑后的正文';
ALTER TABLE `message_extra` ADD COLUMN content_edit_hash varchar(255) not null default '' COMMENT '编辑正文的hash值，用于重复判断';
ALTER TABLE `message_extra` ADD COLUMN edited_at integer not null default 0 COMMENT '编辑时间 时间戳（秒）';