-- +migrate Up

-- 设备消息偏移量
CREATE TABLE `device_offset`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    device_uuid   VARCHAR(40)  not null default '',  -- 设备唯一ID
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    message_seq  bigint not null default 0, -- 偏移的消息序号
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE INDEX uid_device_offset_idx on `device_offset` (uid,device_uuid);

CREATE UNIQUE INDEX uid_device_offset_unidx on `device_offset` (uid,device_uuid,channel_id,channel_type);


-- 用户消息最新偏移量
CREATE TABLE `user_last_offset`(
    id           bigint          not null primary key AUTO_INCREMENT,
    uid          VARCHAR(40) not null default '',  -- 编辑用户唯一ID
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    message_seq  bigint not null default 0, -- 偏移的消息序号
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE INDEX uid_user_last_offset_idx on `user_last_offset` (uid);

CREATE UNIQUE INDEX uid_user_last_offset_unidx on `user_last_offset` (uid,channel_id,channel_type);