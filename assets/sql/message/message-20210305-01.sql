-- +migrate Up

-- 管理员发送消息记录
create table `send_history`(
    id                   integer                not null primary key AUTO_INCREMENT,
    receiver             VARCHAR(40)            not null default '',                     -- 接受者uid
    receiver_name        varchar(100)             not null default '',                    -- 接受者
    receiver_channel_type    smallint not null default 0,                               -- 接受者频道类型
    sender              varchar(40)               not null default '',                  -- 发送者uid
    sender_name         varchar(100)            not null default '',                    -- 发送者名字
    handler_uid         varchar(40)             not null default '',                    -- 操作者uid
    handler_name        VARCHAR(100)            not null default '',                     -- 操作者名字
    content              TEXT,                                                            -- 发送内容   
    created_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP,      -- 创建时间
    updated_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP       -- 更新时间
);


