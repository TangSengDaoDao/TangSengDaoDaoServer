-- +migrate Up

create table `app_config`(
    id   integer not null primary key AUTO_INCREMENT,
    rsa_private_key varchar(4000)  not null default '',     -- 系统私钥 (使用来加密cmd类消息内容 防止前端模拟发送)
    rsa_public_key varchar(4000)  not null default '',     -- 系统公钥
    `version` integer   not null default 0,     -- 数据版本
    super_token varchar(40)  not null default '', -- 超级token 用于操作一些系统api的安全校验
    super_token_on smallint  not null default 0, -- 是否禁用super_token  0.禁用 1.开启 如果禁用 则一些需要super_token的API将不能使用 默认为禁用
    created_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP,   -- 创建时间
    updated_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP    -- 更新时间
);