-- +migrate Up


create table `app`
(
    app_id VARCHAR(40) NOT NULL DEFAULT ''  COMMENT 'app id',
    app_key VARCHAR(40) NOT NULL DEFAULT ''  COMMENT 'app key',
    status  integer   NOT NULL DEFAULT 0  COMMENT '状态 0.禁用 1.可用',
    created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP,
    updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP 
);
CREATE UNIQUE INDEX app_id on `app` (app_id);

insert into `app`(app_id,app_key,status) VALUES('wukongchat',substring(MD5(RAND()),1,20),1);