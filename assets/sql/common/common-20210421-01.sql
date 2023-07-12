-- +migrate Up

-- app 版本管理
create table `app_version`(
    id                   integer                not null primary key AUTO_INCREMENT,
    app_version          VARCHAR(40)            not null default '',                  -- app 版本
    os                   varchar(40)            not null default '',                  -- 系统 ios|android
    is_force             smallint               not null default 0,                   -- 是否强制升级
    update_desc          varchar(100)           not null default '',                  -- 更新说明  
    download_url         varchar(255)           not null default '',                  -- 下载地址
    created_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP,   -- 创建时间
    updated_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP    -- 更新时间
);


