-- +migrate Up

-- 消息扩展表
create table `message_extra`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(严格递增)
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  from_uid   VARCHAR(40)      not null default '', -- 发送者uid
  `revoke`      smallint     not null default 0,  -- 是否撤回
  revoker       VARCHAR(40)   not null default '',  -- 是否撤回
  clone_no     VARCHAR(40)   not null default '', -- 未读编号
  -- voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  `version`       bigint          not null default 0, -- 数据版本
  readed_count  integer     not null default 0,  -- 已读数量
  is_deleted  smallint     not null default 0,  -- 是否已删除
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
); 
CREATE  INDEX from_uid_idx on `message_extra` (from_uid);
CREATE  INDEX channel_idx on `message_extra` (channel_id,channel_type);
CREATE UNIQUE INDEX message_id on `message_extra` (message_id);

-- 成员已读列表
CREATE TABLE `member_readed`(
  id           bigint          not null primary key AUTO_INCREMENT,
  clone_no     VARCHAR(40)   not null default '', -- 克隆成员唯一编号
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  uid     VARCHAR(40)      not null default '', -- 已读用户uid
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE  INDEX channel_idx on `member_readed` (channel_id,channel_type);
CREATE  INDEX uid_idx on `member_readed` (uid);
CREATE UNIQUE INDEX message_uid_idx on `member_readed` (message_id,uid);

-- 成员克隆列表(TODO: 此表已作废)
CREATE TABLE `member_clone`(
    id           bigint          not null primary key AUTO_INCREMENT,
    clone_no     VARCHAR(40)   not null default '', -- 克隆成员唯一编号
    channel_id   VARCHAR(40)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    uid     VARCHAR(40)      not null default '', -- 已读用户uid
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE  INDEX clone_no_idx on `member_clone` (clone_no);
CREATE  INDEX channel_idx on `member_clone` (channel_id,channel_type);

-- 频道成员变化记录
CREATE TABLE `member_change`(
    id        bigint          not null primary key AUTO_INCREMENT,
    clone_no     VARCHAR(40)   not null default '', -- 未读编号
    channel_id   VARCHAR(40)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    max_version bigint   not null default 0, -- 当前最大版本
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);


-- 最近会话扩展表
CREATE TABLE `conversation_extra`(
  id        bigint          not null primary key AUTO_INCREMENT,
  uid     VARCHAR(40)      not null default '' comment '所属用户',
  channel_id   VARCHAR(100)  not null default '' comment '频道ID',
  channel_type smallint      not null default 0 comment '频道类型',
  browse_to    bigint not null default 0 comment '预览到的位置，与会话保持位置不同的是 预览到的位置是用户读到的最大的messageSeq。跟未读消息数量有关系',
  keep_message_seq bigint not null default 0 comment '会话保持的位置',
  keep_offset_y integer not null default 0 comment '会话保持的位置的偏移量',
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP comment '创建时间',
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  comment '更新时间'
);
CREATE UNIQUE INDEX uid_channel_idx on `conversation_extra` (uid,channel_id,channel_type);
CREATE  INDEX uid_idx on `conversation_extra` (uid);