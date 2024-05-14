-- +migrate Up

create table `pinned_message`(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(非严格递增)
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  is_deleted  smallint     not null default 0,  -- 是否已删除
  `version`   bigint    not null default 0, -- 同步版本号
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX pinned_message_message_idx on `pinned_message` (message_id);
CREATE INDEX pinned_message_channelx on `pinned_message` (channel_id, channel_type);
ALTER TABLE `message_extra` ADD COLUMN is_pinned smallint not null default 0 COMMENT '消息是否置顶';