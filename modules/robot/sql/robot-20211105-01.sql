
-- +migrate Up

ALTER TABLE `robot` ADD COLUMN username VARCHAR(40) not null DEFAULT '' comment '机器人的username';

ALTER TABLE `robot` ADD COLUMN app_id VARCHAR(40) not null DEFAULT '' comment '机器人所属app id';