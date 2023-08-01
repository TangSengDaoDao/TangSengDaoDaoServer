-- +migrate Up

create table `chat_bg`(
    id      integer         not null primary key AUTO_INCREMENT,
    cover   varchar(100)    not null default '',                            -- 封面
    url     varchar(100)    not null default '',                            -- 图片地址
    is_svg  smallint        not null default 1,                             -- 是否为svg图片
    created_at timeStamp    not null DEFAULT CURRENT_TIMESTAMP,             -- 创建时间
    updated_at timeStamp    not null DEFAULT CURRENT_TIMESTAMP              -- 更新时间
);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/1_s.jpg', 'file/preview/common/chatbg/default/1_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/2_s.jpg', 'file/preview/common/chatbg/default/2_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/3_s.jpg', 'file/preview/common/chatbg/default/3_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/4_s.jpg', 'file/preview/common/chatbg/default/4_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/5_s.jpg', 'file/preview/common/chatbg/default/5_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/6_s.jpg', 'file/preview/common/chatbg/default/6_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/7_s.jpg', 'file/preview/common/chatbg/default/7_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/8_s.png', 'file/preview/common/chatbg/default/8_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/9_s.jpg', 'file/preview/common/chatbg/default/9_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/10_s.jpg', 'file/preview/common/chatbg/default/10_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/11_s.jpg', 'file/preview/common/chatbg/default/11_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/12_s.jpg', 'file/preview/common/chatbg/default/12_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/13_s.jpg', 'file/preview/common/chatbg/default/13_b.svg',1);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/14_s.jpg', 'file/preview/common/chatbg/default/14_b.jpg',0);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/15_s.jpg', 'file/preview/common/chatbg/default/15_b.jpg',0);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/16_s.jpg', 'file/preview/common/chatbg/default/16_b.jpg',0);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/17_s.jpg', 'file/preview/common/chatbg/default/17_b.jpg',0);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/18_s.jpg', 'file/preview/common/chatbg/default/18_b.jpg',0);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/19_s.jpg', 'file/preview/common/chatbg/default/19_b.jpg',0);
insert into chat_bg(cover,url,is_svg) value('file/preview/common/chatbg/default/20_s.jpg', 'file/preview/common/chatbg/default/20_b.jpg',0);
