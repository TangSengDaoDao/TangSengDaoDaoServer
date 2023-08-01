-- +migrate Up


-- 群组
create table `group`
(
  id         integer     not null primary key AUTO_INCREMENT,
  group_no   VARCHAR(40) not null default '',                             -- 群唯一编号
  name       VARCHAR(40) not null default '',                             -- 群名字
  creator    VARCHAR(40) not null default '',                             -- 创建者uid
  status     smallint    not null DEFAULT 0,                              -- 群状态 '0.不正常 1.正常',
  forbidden   smallint    not null DEFAULT 0 comment '群禁言',
  invite     smallint    not null DEFAULT 0 comment '群邀请开关',
  forbidden_add_friend   smallint      not null DEFAULT 0,                              -- 群内禁止加好友
  allow_view_history_msg smallint     not null DEFAULT 1, -- 是否允许新成员查看历史消息
  `version`    bigint      not null DEFAULT 0,                               -- 数据版本 
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX group_groupNo on `group` (group_no);
CREATE INDEX group_creator on `group` (creator);

-- -- +migrate StatementBegin
-- CREATE TRIGGER group_updated_at
--   BEFORE UPDATE
--   ON `group` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd

-- 群成员
create table `group_member`
(
  id         integer     not null primary key AUTO_INCREMENT,
  group_no   VARCHAR(40) not null default '',                             -- 群唯一编号
  uid        VARCHAR(40) not null default '',                             -- 成员uid
  remark     VARCHAR(40) not null default '',                             -- 成员备注
  role       smallint    NOT NULL DEFAULT 0,                              -- 成员角色 '0.普通成员 1.群主  2.管理员',
  `version`    bigint     not null DEFAULT 0,                               -- 数据版本 
  is_deleted smallint    not null default 0,                              -- 是否已删除
  status     smallint   not null default 1,                                   -- 成员状态
  vercode    VARCHAR(100) not null default '',                 -- 验证码 加好友来源  
  robot      smallint     not null default 0,                  -- 机器人标识0.否1.是
  invite_uid VARCHAR(40) not null default '',                  -- 邀请人uid
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE unique INDEX group_no_uid on `group_member` (group_no, uid);
CREATE INDEX group_member_groupNo on `group_member` (group_no);
CREATE INDEX group_member_uid on `group_member` (uid);

-- -- +migrate StatementBegin
--   CREATE TRIGGER group_member_updated_at
--   BEFORE UPDATE
--   ON `group_member` for each row 
--   BEGIN
--    set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd


-- 群设置
create table `group_setting`
(
  id                        integer       not null primary key AUTO_INCREMENT,
  uid                       VARCHAR(40)   not null default '',                             -- 用户uid
  group_no                  VARCHAR(40)   not null default '',                             -- 群唯一编号
  remark                    VARCHAR(100) not null default '',                              -- 群备注
  mute                      smallint      not null DEFAULT 0,                              --  是否免打扰
  top                       smallint      not null DEFAULT 0,                              -- 是否置顶
  show_nick                 smallint      not null DEFAULT 0,                              -- 是否显示昵称
  save                      smallint      not null DEFAULT 0,                              -- 是否保存
  chat_pwd_on               smallint      not null DEFAULT 0,                              -- 是否开启聊天密码
  revoke_remind             smallint      not null DEFAULT 1,                              -- 撤回提醒
  join_group_remind         smallint      not null DEFAULT 0,                              -- 进群提醒
  screenshot                smallint      not null DEFAULT 0,                              -- 截屏通知
  receipt                   smallint      not null default 0,                              -- 消息是否回执
  `version`                   BIGINT        not null DEFAULT 0,                              -- 版本
  created_at                timeStamp     not null DEFAULT CURRENT_TIMESTAMP,              -- 创建时间
  updated_at                timeStamp     not null DEFAULT CURRENT_TIMESTAMP               -- 更新时间
);
CREATE unique INDEX groupsetting_group_no_uid on `group_setting` (group_no, uid);
CREATE INDEX group_setting_groupNo on `group_setting` (group_no);
CREATE INDEX group_setting_uid on `group_setting` (uid);

-- -- +migrate StatementBegin
--   CREATE TRIGGER group_setting_updated_at
--   BEFORE UPDATE
--   ON `group_setting` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd

alter table `group` add column notice varchar(400) not null default ''; -- 群公告


-- 群邀请
CREATE TABLE `group_invite` (
  id         integer     not null primary key AUTO_INCREMENT,
  invite_no  VARCHAR(40) not null default '' comment '邀请唯一编号',                           
  group_no   VARCHAR(40) not null default '' comment '群唯一编号',                
  inviter    VARCHAR(40) not null default '' comment '邀请者uid',              
  remark     VARCHAR(100) not null default '' comment '备注',
  status     smallint  not null default 0 comment '状态： 0.待确认 1.已确认', 
  allower   VARCHAR(40)  not null default '' comment '允许此次操作的用户uid',
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP comment '创建时间', 
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  comment '更新时间'
);
-- -- +migrate StatementBegin
-- CREATE TRIGGER group_invite_updated_at
--   BEFORE UPDATE
--   ON `group_invite` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd

-- 群邀请
CREATE TABLE `invite_item` (
  id         integer     not null primary key AUTO_INCREMENT,
  invite_no  VARCHAR(40) not null default '' comment '邀请唯一编号',                            
  group_no   VARCHAR(40) not null default '' comment '群唯一编号',                            
  inviter    VARCHAR(40) not null default '' comment '邀请者uid',  
  uid        VARCHAR(40) not null default '' comment '被邀请者uid',                            
  status     smallint  not null default 0 comment '状态： 0.待确认 1.已确认', 
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP comment '创建时间', 
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  comment '更新时间'
);
-- -- +migrate StatementBegin
-- CREATE TRIGGER invite_item_updated_at
--   BEFORE UPDATE
--   ON `invite_item` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd