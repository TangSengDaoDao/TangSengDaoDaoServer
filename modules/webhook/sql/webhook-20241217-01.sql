-- +migrate Up

ALTER TABLE `message` MODIFY COLUMN client_msg_no VARCHAR(100) not null default '';
ALTER TABLE `message1` MODIFY COLUMN client_msg_no VARCHAR(100) not null default '';
ALTER TABLE `message2` MODIFY COLUMN client_msg_no VARCHAR(100) not null default '';
ALTER TABLE `message3` MODIFY COLUMN client_msg_no VARCHAR(100) not null default '';
ALTER TABLE `message4` MODIFY COLUMN client_msg_no VARCHAR(100) not null default '';