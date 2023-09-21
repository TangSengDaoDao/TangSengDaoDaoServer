-- +migrate Up



ALTER TABLE `message` ADD COLUMN expire integer not null DEFAULT 0 COMMENT '消息过期时长';
ALTER TABLE `message` ADD COLUMN expire_at BIGINT not null DEFAULT 0 COMMENT '消息过期时间';

ALTER TABLE `message1` ADD COLUMN expire integer not null DEFAULT 0 COMMENT '消息过期时长';
ALTER TABLE `message1` ADD COLUMN expire_at BIGINT not null DEFAULT 0 COMMENT '消息过期时间';

ALTER TABLE `message2` ADD COLUMN expire integer not null DEFAULT 0 COMMENT '消息过期时长';
ALTER TABLE `message2` ADD COLUMN expire_at BIGINT not null DEFAULT 0 COMMENT '消息过期时间';

ALTER TABLE `message3` ADD COLUMN expire integer not null DEFAULT 0 COMMENT '消息过期时长';
ALTER TABLE `message3` ADD COLUMN expire_at BIGINT not null DEFAULT 0 COMMENT '消息过期时间';

ALTER TABLE `message4` ADD COLUMN expire integer not null DEFAULT 0 COMMENT '消息过期时长';
ALTER TABLE `message4` ADD COLUMN expire_at BIGINT not null DEFAULT 0 COMMENT '消息过期时间';