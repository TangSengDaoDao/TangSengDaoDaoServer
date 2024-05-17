-- +migrate Up

ALTER TABLE `group` ADD COLUMN allow_member_pinned_message smallint not null DEFAULT 0 COMMENT '允许成员置顶聊天消息 0.不允许 1.允许';
