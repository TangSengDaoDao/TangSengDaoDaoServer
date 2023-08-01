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
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('advanced','旗舰模块','高级功能，包含消息已读未读，消息点赞，截屏消息，在线状态，查看会话历史消息(非全局历史消息查询)等等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('groupManager','群管理模块','支持群禁言，群管理添加，群头像设置，开启群成员邀请机制等等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('sticker','表情商店','消息可支持矢量表情和GIF动图，用户可制作表情，添加移除表情，排序表情包等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('rich','富文本','聊天中支持富文本消息，包含加粗，图片，强提醒，斜线，变色，下划线，字体颜色等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('video','小视频模块','聊天中支持视频消息，可录视频，发送小视频消息，播放小视频',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('map','地图模块','聊天中支持地理位置消息，发朋友圈也支持显示位置信息等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('file','文件模块','添加文件模块可在聊天中发送文件消息，消息搜索支持文件类型筛选，也可查看收到的文件并打开',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('rtc','音视频模块','聊天中支持个人音视频通话，群支持会议模式等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('label','标签模块','将好友进行标签分组管理，可在发朋友圈时选择标签用户过滤可见或不可见等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('security','安全与隐私模块','手机号搜索保护，设备锁，黑名单，聊天密码设置，锁屏密码设置，禁止app截屏录屏等等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('customerService','客服','支持客服分配坐席演示',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('moment','朋友圈模块','发布朋友圈，查看朋友圈，评论朋友圈，收到评论动态时显示朋友圈消息红点等等',1);
INSERT INTO `app_module` (sid,name,`desc`,status) VALUES ('favorite','收藏模块','可对聊天中的文本消息，图片消息消息进行收藏，查看收藏列表信息等',1);