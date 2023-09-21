-- +migrate Up


-- 消息表
create table `message`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(非严格递增)
  client_msg_no VARCHAR(40)      not null default '', -- 客户端消息唯一编号
  header       varchar(100)  not null default '', -- 消息头
  setting     smallint       not null default 0, -- 设置
  `signal`       smallint        not null default 0, -- 是否signal加密
  from_uid     VARCHAR(40)      not null default '', -- 发送者uid
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  timestamp    BIGINT           not null default 0,  -- 消息时间
  payload      mediumblob             not null , -- 消息内容
  is_deleted  smallint     not null default 0,  -- 是否已删除
  voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX message_id on `message` (message_id);


-- 消息表
create table `message1`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(非严格递增)
  client_msg_no VARCHAR(40)      not null default '', -- 客户端消息唯一编号
  setting     smallint       not null default 0, -- 设置
  `signal`       smallint        not null default 0, -- 是否signal加密
  header       varchar(100)  not null default '', -- 消息头
  from_uid     VARCHAR(40)      not null default '', -- 发送者uid
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  timestamp    BIGINT           not null default 0,  -- 消息时间
  payload      mediumblob             not null , -- 消息内容
  is_deleted  smallint     not null default 0,  -- 是否已删除
  voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX message_id on `message1` (message_id);


-- 消息表
create table `message2`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(非严格递增)
  client_msg_no VARCHAR(40)      not null default '', -- 客户端消息唯一编号
  setting     smallint       not null default 0, -- 设置
  `signal`       smallint        not null default 0, -- 是否signal加密
  header       varchar(100)  not null default '', -- 消息头
  from_uid     VARCHAR(40)      not null default '', -- 发送者uid
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  timestamp    BIGINT           not null default 0,  -- 消息时间
  payload      mediumblob             not null , -- 消息内容
  is_deleted  smallint     not null default 0,  -- 是否已删除
  voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX message_id on `message2` (message_id);


-- 消息表
create table `message3`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(非严格递增)
  client_msg_no VARCHAR(40)      not null default '', -- 客户端消息唯一编号
  setting     smallint       not null default 0, -- 设置
  `signal`       smallint        not null default 0, -- 是否signal加密
  header       varchar(100)  not null default '', -- 消息头
  from_uid     VARCHAR(40)      not null default '', -- 发送者uid
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  timestamp    BIGINT           not null default 0,  -- 消息时间
  payload      mediumblob             not null , -- 消息内容
  is_deleted  smallint     not null default 0,  -- 是否已删除
  voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX message_id on `message3` (message_id);


-- 消息表
create table `message4`
(
  id           bigint          not null primary key AUTO_INCREMENT,
  message_id   VARCHAR(20) not null default '',  -- 消息唯一ID（全局唯一）
  message_seq  bigint not null default 0,  -- 消息序列号(非严格递增)
  client_msg_no VARCHAR(40)      not null default '', -- 客户端消息唯一编号
  setting     smallint       not null default 0, -- 设置
  `signal`       smallint        not null default 0, -- 是否signal加密
  header       varchar(100)  not null default '', -- 消息头
  from_uid     VARCHAR(40)      not null default '', -- 发送者uid
  channel_id   VARCHAR(100)      not null default '', -- 频道ID
  channel_type smallint         not null default 0,  -- 频道类型
  timestamp    BIGINT           not null default 0,  -- 消息时间
  payload      mediumblob             not null , -- 消息内容
  is_deleted  smallint     not null default 0,  -- 是否已删除
  voice_status smallint not null default 0, -- 语音状态 0.未读 1.已读
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX message_id on `message4` (message_id);
