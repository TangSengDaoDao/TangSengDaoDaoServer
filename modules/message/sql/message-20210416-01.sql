-- +migrate Up

-- 回应用户
CREATE TABLE `reaction_users`(
    id    bigint          not null primary key AUTO_INCREMENT,  
    message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
    seq           bigint not null default 0,  --  回复递增序号（可以用此序号做递增操作）
    channel_id   VARCHAR(100)      not null default '', -- 频道ID
    channel_type smallint         not null default 0,  -- 频道类型
    uid        varchar(40)    not null default '',  -- 回应的用户uid
    name       varchar(40)    not null default '',  -- 回应的用户名
    emoji      varchar(20)       not null default '',  -- 回应的emoji
    is_deleted  smallint     not null default 0,  -- 是否已删除
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE  INDEX `reaction_user_message_channel` on `reaction_users` (`message_id`,uid,`emoji`);


