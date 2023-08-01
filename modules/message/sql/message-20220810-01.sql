-- +migrate Up

create table `prohibit_words`(
    id                   integer                not null primary key AUTO_INCREMENT,
    is_deleted          smallint                not null default 0, -- 是否删除
    `version`           bigint                  not null default 0,
    content              TEXT,                                                            -- 内容   
    created_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP,      -- 创建时间
    updated_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP       -- 更新时间
);
