-- +migrate Up

-- ##########  举报类别 ########## 

create table IF NOT EXISTS  `report_category`
(
    id integer PRIMARY KEY AUTO_INCREMENT,
    category_no VARCHAR(40)  not null DEFAULT '' comment '类别编号',
    category_name VARCHAR(40)  not null DEFAULT '' comment '类别名称',
    parent_category_no VARCHAR(40)  not null DEFAULT '' comment '父类别编号',
    created_at timeStamp    not null DEFAULT CURRENT_TIMESTAMP,
    updated_at timeStamp    not null DEFAULT CURRENT_TIMESTAMP 
);
CREATE UNIQUE INDEX  report_category_no_idx on `report_category` (category_no);

INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10000','发布不适当内容对我造成骚扰','');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20000','存在欺诈骗钱行为','');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('30000','此账号可能被盗用','');

INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10001','色情','10000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10002','违法违禁品','10000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10003','赌博','10000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10004','政治谣言','10000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10005','暴恐血腥','10000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('10006','其他违规内容','10000');

INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20001','收款不发货骗钱','20000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20002','金融贷款诈骗骗钱','20000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20003','网络兼职诈骗骗钱','20000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20004','仿冒他人诈骗骗钱','20000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20005','免费送诈骗骗钱','20000');
INSERT INTO report_category(category_no,category_name,parent_category_no) VALUES('20006','其他欺诈骗钱行为','20000');


-- -- +migrate StatementBegin
-- CREATE TRIGGER  report_category_updated_at
--   BEFORE UPDATE
--   ON `report_category` for each row 
--   BEGIN 
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd

create table IF NOT EXISTS  `report`
(
    id integer PRIMARY KEY AUTO_INCREMENT,
    uid         VARCHAR(40)  not null DEFAULT '' comment '举报用户',
    category_no VARCHAR(40)  not null DEFAULT '' comment '类别编号',
    channel_id  VARCHAR(40)  not null DEFAULT '' comment '频道ID',
    channel_type smallint  not null DEFAULT 0 comment '频道类型',
    imgs        VARCHAR(1000) not null DEFAULT '' comment '图片集合',
    remark      VARCHAR(800) not null DEFAULT '' comment '备注',
    created_at timeStamp    not null DEFAULT CURRENT_TIMESTAMP,
    updated_at timeStamp    not null DEFAULT CURRENT_TIMESTAMP 
);

-- -- +migrate StatementBegin
-- CREATE TRIGGER report_updated_at
--   BEFORE UPDATE
--   ON `report` for each row 
--   BEGIN 
--     set NEW.updated_at = NOW();
--   END;
-- -- +migrate StatementEnd