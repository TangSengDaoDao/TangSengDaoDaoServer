
-- +migrate Up

-- 工作台分类
create table `workplace_category`(
    id              bigint         not null primary key AUTO_INCREMENT,
    category_no     VARCHAR(40)    not null DEFAULT '',                -- 分类编号
    name            VARCHAR(100)    not null DEFAULT '',                -- 分类名称
    sort_num        integer        not null DEFAULT 0,                 -- 排序编号
    created_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

-- 工作台应用
create table `workplace_app`(
    id              bigint         not null primary key AUTO_INCREMENT,
    app_id          VARCHAR(40)    not null DEFAULT '',                -- 应用ID
    icon            VARCHAR(100)    not null DEFAULT '',                -- 应用icon
    name            VARCHAR(100)    not null DEFAULT '',                -- 应用名称
    `description`   VARCHAR(1000)   not null DEFAULT '',                -- 应用介绍
    app_category    VARCHAR(40)    not null DEFAULT '',                -- 应用分类 [‘机器人’ ‘客服’]
    status          smallint       not null DEFAULT 1,                 -- 是否可用 0.禁用 1.可用
    jump_type       smallint       not null DEFAULT 0,                 -- 打开方式 0.网页 1.原生
    app_route       VARCHAR(200)   not null DEFAULT '',                -- app打开地址
    web_route       VARCHAR(200)   not null DEFAULT '',                -- web打开地址
    is_paid_app     smallint       not null DEFAULT 0,                 -- 是否为付费应用 0.否 1.是
    created_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE INDEX workplace_app_appid on `workplace_app` (app_id);

-- 用户常用app
create  table `workplace_user_app`(
    id              bigint         not null primary key AUTO_INCREMENT,
    app_id          VARCHAR(40)    not null DEFAULT '',                -- 应用ID
    sort_num        integer        not null DEFAULT 0,                 -- 排序编号
    uid             VARCHAR(40)    not null DEFAULT '',                -- 所属用户uid
    created_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE INDEX workplace_user_app_uid on `workplace_user_app` (uid);

-- 工作台横幅
create  table `workplace_banner`(
    id              bigint         not null primary key AUTO_INCREMENT,
    banner_no       VARCHAR(40)    not null DEFAULT '',                -- 封面编号
    cover           VARCHAR(100)    not null DEFAULT '',                -- 封面地址
    title           VARCHAR(100)    not null DEFAULT '',                -- 横幅标题
    `description`   VARCHAR(1000)   not null DEFAULT '',                -- 横幅介绍
    jump_type       smallint       not null DEFAULT 0,                 -- 打开方式 0.网页 1.原生
    `route`         VARCHAR(200)   not null DEFAULT '',                -- 打开地址
    created_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

-- app使用记录
create  table `workplace_app_user_record`(
    id              bigint         not null primary key AUTO_INCREMENT,
    app_id          VARCHAR(40)    not null DEFAULT '',                -- 应用ID
    uid             VARCHAR(40)    not null DEFAULT '',                -- 所属用户uid
    count           integer        not null DEFAULT 0,                 -- 使用次数
    created_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
    updated_at      timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE INDEX workplace_app_user_record_uid on `workplace_app_user_record` (uid);
CREATE unique INDEX workplace_app_user_record_uid_appid on `workplace_app_user_record` (uid,app_id);
