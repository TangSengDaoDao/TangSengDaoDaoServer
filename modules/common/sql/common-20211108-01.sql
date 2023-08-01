-- +migrate Up

create table `seq`(
    id   integer not null primary key AUTO_INCREMENT,
    `key`  varchar(100) not null default '', -- seq的key
    `min_seq` bigint   not null default 1000000,     -- 开始序号
    step integer not null default 1000, -- 序号步长 每次启动后 当前序号 应该等于min_seq+ step
    created_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP,   -- 创建时间
    updated_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP    -- 更新时间
);
CREATE UNIQUE INDEX `seq_uidx` on `seq` (`key`);