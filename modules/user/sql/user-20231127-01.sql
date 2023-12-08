

-- +migrate Up

-- 好友申请记录
create table `friend_apply_record`
(
  id         bigint         not null primary key AUTO_INCREMENT,
  uid        VARCHAR(40)    not null default '',                -- 用户uid
  to_uid     VARCHAR(40)    not null default '',                -- 申请者uid
  remark     VARCHAR(200)   not null default '',                -- 申请备注
  status     smallint       not null DEFAULT 1,                 -- 状态 0.未处理 1.通过 2.拒绝
  token      VARCHAR(200)   not null default '',                -- 通过好友所需验证
  created_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE INDEX `friend_apply_record_uidx` on `friend_apply_record` (`uid`);
CREATE UNIQUE INDEX `friend_apply_record_uid_touidx` on `friend_apply_record` (`uid`,`to_uid`);

-- 用户业务红点
CREATE table `user_red_dot`(
  id         bigint         not null primary key AUTO_INCREMENT,
  uid        VARCHAR(40)    not null default '',                -- 用户uid
  count      smallint       not null default 0,                 -- 未读数量
  category   VARCHAR(40)    not null default '',                -- 红点分类
  is_dot     smallint       not null default 0,                 -- 是否显示红点 1.是 0.否
  created_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX `user_red_dot_uid_categoryx` on `user_red_dot` (`uid`,`category`);
