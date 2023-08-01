-- +migrate Up

CREATE TABLE `reminders`(
    id              bigint          not null primary key AUTO_INCREMENT,  
    channel_id      VARCHAR(100)    not null default '' COMMENT '频道ID',
    channel_type    smallint        not null default 0 COMMENT '频道类型', 
    reminder_type   integer         not null default 0 COMMENT '提醒类型 1.有人@我 2.草稿', 
    uid             varchar(40)     not null default '' COMMENT '提醒的用户uid，如果此字段为空则表示 提醒项为整个频道内的成员',     
    `text`          varchar(255)    not null default '' COMMENT '提醒内容',
    `data`          varchar(1000)   not null default '' COMMENT '自定义数据',
    is_locate       smallint        not null default 0 COMMENT ' 是否需要定位',
    message_seq     bigint          not null default 0 COMMENT '消息序列号', 
    message_id      VARCHAR(20)     not null default '' COMMENT '消息唯一ID（全局唯一）',
    `version`       bigint          not null default 0 COMMENT ' 数据版本',
    created_at      timeStamp       not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp       not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE  INDEX channel_uid_uidx on `reminders` (uid,channel_id,channel_type);


CREATE TABLE `reminder_done`(
    id              bigint          not null primary key AUTO_INCREMENT,  
    reminder_id     bigint    not null default 0 COMMENT '提醒事项的id',
    uid             varchar(40)     not null default '' COMMENT '完成的用户uid',     
    created_at      timeStamp       not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp       not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX reminder_id_uidx on `reminder_done` (reminder_id,uid);