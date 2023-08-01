
-- +migrate Up

-- 用户表
create table `user`
(
  id         integer      not null primary key AUTO_INCREMENT,
  uid        VARCHAR(40)  not null default '',                             -- 用户唯一ID
  name       VARCHAR(100) not null default '',                             -- 用户的名字
  short_no   VARCHAR(40)  not null default '',                             -- 短编码
  short_status smallint   not null default 0,                              -- 短编码 0.未修改 1.已修改
  sex        smallint     not null default 0,                              -- 性别 0.女 1.男
  robot      smallint     not null default 0,                              -- 机器人 0.否1.是
  category   VARCHAR(40)  not null default '',                             -- 用户分类  service:客服
  role       VARCHAR(40)  not null default '',                             -- 用户角色  admin:管理员 superAdmin
  username   VARCHAR(40)  not null default '',                             -- 用户名
  password   VARCHAR(40)  not null default '',                             -- 密码
  zone       VARCHAR(40)  not null default '',                             -- 手机区号
  phone      VARCHAR(20)  not null default '',                             -- 手机号
  chat_pwd   VARCHAR(40)  not null default '',                             -- 聊天密码
  lock_screen_pwd varchar(40) not null default '',                         -- 锁屏密码
  lock_after_minute integer  not null default 0,                           -- 在几分钟后锁屏 0 表示立即
  vercode    VARCHAR(100) not null default '',                             -- 验证码 加好友来源
  is_upload_avatar        smallint not null default 0,                     -- 是否上传过头像 1:上传0:未上传
  qr_vercode VARCHAR(100) not null default '',                             -- 二维码验证码 加好友来源
  device_lock           smallint     not null DEFAULT 0,                   -- 是否开启设备锁
  search_by_phone       smallint     not null default 1,                   -- 是否可用通过手机号搜索到本人0.否1.是
  search_by_short       smallint     not null default 1,                   -- 是否可以通过短编号搜索0.否1.是
  new_msg_notice        smallint     not null default 1,                   -- 新消息通知0.否1.是
  msg_show_detail       smallint     not null default 1,                   -- 新消息通知详情0.否1.是
  voice_on              smallint     not null default 1,                   -- 是否开启声音0.否1.是
  shock_on              smallint     not null default 1,                   -- 是否开启震动0.否1.是
  mute_of_app           smallint     not null default 0,                   -- app是否禁音（当pc登录的时候app可以设置禁音，当pc登录后有效）
  offline_protection    smallint     not null default 0,                   -- 离线保护，断网屏保
  `version`    bigint     not null DEFAULT 0,                                -- 数据版本 
  status smallint       not null DEFAULT 1,                                -- 用户状态 0.禁用 1.可用
  bench_no        VARCHAR(40)     not null default '',                     -- 性能测试批次号，性能测试幂等用
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX uid on `user` (uid);
CREATE UNIQUE INDEX short_no_udx on `user` (short_no);

-- -- +migrate StatementBegin
-- CREATE TRIGGER user_updated_at
--   BEFORE UPDATE
--   ON `user` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd

-- 创建系统账号
INSERT INTO `user` (uid,name,short_no,phone,zone,search_by_phone,search_by_short,new_msg_notice,voice_on,shock_on,msg_show_detail,status,is_upload_avatar,category,robot) VALUES ('u_10000','系统账号',10000,'13000000000','0086',0,0,0,0,0,0,1,1,'system',1);
INSERT INTO `user` (uid,name,short_no,phone,zone,search_by_phone,search_by_short,new_msg_notice,voice_on,shock_on,msg_show_detail,status,is_upload_avatar,category) VALUES ('fileHelper','文件传输助手',20000,'13000000001','0086',0,0,0,0,0,0,1,1,'system');
-- 创建后台管理平台超级管理员账号 admin/admiN123456
-- INSERT INTO `user` (uid,name,short_no,username,password,role,phone,zone,search_by_phone,search_by_short,new_msg_notice,voice_on,shock_on,msg_show_detail,status,is_upload_avatar,category) VALUES ('admin','超级管理员',30000,'admin','14c3a0db22308e34ca7dacb1806c0bdf','superAdmin','13000000002','0086',0,0,0,0,0,0,1,0,'system');

-- 用户设置
create table `user_setting`
(
  id               integer       not null primary key AUTO_INCREMENT,
  uid              VARCHAR(40)   not null default '',                              -- 用户UID
  to_uid           VARCHAR(40)   not null default '',                              -- 对方uid
  mute             smallint      not null DEFAULT 0,                               --  是否免打扰
  top              smallint      not null DEFAULT 0,                               -- 是否置顶
  blacklist        smallint      not null DEFAULT 0,                               -- 是否黑名单 0:正常1:黑名单
  chat_pwd_on      smallint      not null DEFAULT 0,                               -- 是否开启聊天密码
  screenshot       smallint      not null DEFAULT 1,                               -- 截屏通知
  revoke_remind    smallint      not null DEFAULT 1,                               -- 撤回通知
  receipt          smallint      not null default 1,                               -- 消息是否回执
  version          BIGINT        not null DEFAULT 0,                               -- 版本
  created_at       timeStamp     not null DEFAULT CURRENT_TIMESTAMP,               -- 创建时间
  updated_at       timeStamp     not null DEFAULT CURRENT_TIMESTAMP                -- 更新时间
);

CREATE  INDEX uid_idx on `user_setting` (uid);
-- -- +migrate StatementBegin
-- CREATE TRIGGER user_setting_updated_at
--   BEFORE UPDATE
--   ON `user_setting` for each row 
--   BEGIN
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd


-- 用户设备
create table `device`
(
  id                integer             not null primary key AUTO_INCREMENT,
  uid               VARCHAR(40)         not null default '',                      -- 设备所属用户uid                     
  device_id         VARCHAR(40)         not null default '',                      -- 设备唯一ID          
  device_name       VARCHAR(100)        not null default '',                      -- 设备名称                  
  device_model      VARCHAR(100)        not null default '',                      -- 设备型号              
  last_login        integer             not null DEFAULT 0,                       -- 最后一次登录时间(时间戳 10位)
  created_at        timeStamp           not null DEFAULT CURRENT_TIMESTAMP,       -- 创建时间
  updated_at        timeStamp           not null DEFAULT CURRENT_TIMESTAMP        -- 更新时间
);
CREATE unique INDEX device_uid_device_id on `device` (uid, device_id);
CREATE INDEX device_uid on `device` (uid);
CREATE INDEX device_device_id on `device` (device_id);

-- 好友表
create table `friend`
(
  id                integer               not null primary key AUTO_INCREMENT,
  uid               VARCHAR(40)           not null default '' comment '用户UID',       
  to_uid            VARCHAR(40)           not null default '' comment '好友uid',                        
  remark            varchar(100)          not null default '' comment '对好友的备注 TODO: 此字段不再使用，已经迁移到user_setting表', 
  flag              smallint              not null default 0 comment '好友标示', 
  `version`           bigint                not null default 0 comment '版本号',
  vercode           VARCHAR(100)          not null default '' comment '验证码 加好友来源',   
  source_vercode    varchar(100)          not null default '' comment '好友来源',      
  is_deleted        smallint              not null default 0 comment '是否已删除', 
  is_alone          smallint              not null default 0 comment  '单项好友',
  initiator         smallint              not null default 0 comment '加好友发起方',
  created_at        timeStamp             not null DEFAULT CURRENT_TIMESTAMP comment '创建时间',
  updated_at        timeStamp             not null DEFAULT CURRENT_TIMESTAMP comment '更新时间'
);

-- -- +migrate StatementBegin
-- CREATE TRIGGER friend_updated_at
--   BEFORE UPDATE
--   ON `friend` for each row 
--   BEGIN
--    set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd


-- +migrate Up
-- 登录日志
CREATE TABLE IF NOT EXISTS login_log(
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  uid VARCHAR(40) DEFAULT '' NOT NULL  COMMENT '用户OpenId',
  login_ip    VARCHAR(40) DEFAULT '' NOT NULL COMMENT '最后一次登录ip',
  created_at  timeStamp     not null DEFAULT CURRENT_TIMESTAMP comment '创建时间',
  updated_at  timeStamp     not null DEFAULT CURRENT_TIMESTAMP comment '更新时间'
) CHARACTER SET utf8mb4;

