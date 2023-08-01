

-- +migrate Up

-- 手机联系人
create table `user_maillist`
(
  id         bigint         not null primary key AUTO_INCREMENT,
  uid        VARCHAR(40)    not null default '', -- 用户uid
  phone      VARCHAR(40)    not null default '', -- 手机号
  zone       VARCHAR(40)    not null default '', -- 区号
  name       VARCHAR(40)    not null default '', -- 名字
  vercode    VARCHAR(100)   not null default '', -- 验证码 加好友来源
  created_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX `uid_maillist_index` on `user_maillist` (`uid`,`zone`,`phone`);