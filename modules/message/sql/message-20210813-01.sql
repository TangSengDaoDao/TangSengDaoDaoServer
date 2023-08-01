-- +migrate Up

--  用户独立对消息的扩充
CREATE TABLE `message_user_extra`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
    message_seq  bigint not null default 0,  -- 消息序列号(严格递增)
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    voice_readed smallint         not null default 0,  -- 语音是否已读
    message_is_deleted  smallint     not null default 0,  -- 消息是否已删除
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_message_idx on `message_user_extra` (uid,message_id);


CREATE TABLE `message_user_extra1`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
    message_seq  bigint not null default 0,  -- 消息序列号(严格递增)
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    voice_readed smallint         not null default 0,  -- 语音是否已读
    message_is_deleted  smallint     not null default 0,  -- 消息是否已删除
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_message_idx on `message_user_extra1` (uid,message_id);

CREATE TABLE `message_user_extra2`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
    message_seq  bigint not null default 0,  -- 消息序列号(严格递增)
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    voice_readed smallint         not null default 0,  -- 语音是否已读
    message_is_deleted  smallint     not null default 0,  -- 消息是否已删除
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_message_idx on `message_user_extra2` (uid,message_id);


-- 频道偏移表 （每个用户针对于频道的偏移位置）
CREATE TABLE `channel_offset`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    message_seq  bigint not null default 0, -- 偏移的消息序号
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_channel_idx on `channel_offset` (uid,channel_id,channel_type);

CREATE TABLE `channel_offset1`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    message_seq  bigint not null default 0, -- 偏移的消息序号
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_channel_idx on `channel_offset1` (uid,channel_id,channel_type);

CREATE TABLE `channel_offset2`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    message_seq  bigint not null default 0, -- 偏移的消息序号
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid_channel_idx on `channel_offset2` (uid,channel_id,channel_type);