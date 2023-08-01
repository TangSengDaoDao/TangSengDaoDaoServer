

-- +migrate Up

-- 消息表
create table `user_online`
(
  id         bigint        not null primary key AUTO_INCREMENT,
  uid        VARCHAR(40)   not null default '', -- 用户uid
  device_flag      smallint    not null default 0, -- 设备flag 0.APP 1. WEB
  last_online integer       not null DEFAULT 0, -- 最后一次在线时间
  last_offline integer     not null DEFAULT 0, -- 最后一次离线时间
  online     tinyint(1)     not null default 0, -- 用户是否在线
  `version`    bigint       not null default 0, -- 数据版本
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX `uid_device` on `user_online` (`uid`,device_flag);

CREATE  INDEX `online_idx` on `user_online` (`online`);
CREATE  INDEX `uid_idx` on `user_online` (`uid`);