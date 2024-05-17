-- +migrate Up

ALTER TABLE `app_config` ADD COLUMN channel_pinned_message_max_count smallint not null DEFAULT 10 COMMENT '频道最多置顶消息数量';
