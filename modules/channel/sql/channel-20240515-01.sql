-- +migrate Up

ALTER TABLE `channel_setting` ADD COLUMN offset_message_seq integer not null DEFAULT 0 COMMENT 'channel消息删除偏移seq';
