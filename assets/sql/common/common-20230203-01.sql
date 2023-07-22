-- +migrate Up

-- app 模块管理
create table `app_module`(
    id                   integer                not null primary key AUTO_INCREMENT,
    sid                  varchar(40)            not null default '',                  -- 模块ID    
    name                 VARCHAR(40)            not null default '',                  -- 模块名称
    `desc`               varchar(100)           not null default '',                  -- 模块介绍
    status               smallint               not null default 0,                   -- 模块状态 1.可用 0.不可用
    created_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP,   -- 创建时间
    updated_at           timeStamp              not null DEFAULT CURRENT_TIMESTAMP    -- 更新时间
);
CREATE  INDEX app_module_sid_idx on `app_module` (sid);

INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('base','基础模块','app基础模块，包含基本的文本消息，图片消息，语音消息，名片消息，联系人，用户资料，个人资料，通用设置等等',2);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('login','登录模块','app基础模块，包含用户登录，注册，授权pc/web登录，修改登录密码，还可以在此开发第三方登录等',2);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('scan','扫一扫模块','app基础模块，扫描二维码添加好友，跳转网页等',2);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('security','安全与隐私模块','手机号搜索保护，设备锁，黑名单，聊天密码设置，锁屏密码设置，禁止app截屏录屏等等',1);