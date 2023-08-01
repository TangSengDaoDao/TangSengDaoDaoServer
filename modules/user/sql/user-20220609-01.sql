-- +migrate Up

ALTER TABLE `user_setting` ADD COLUMN remark VARCHAR(100) NOT NULL DEFAULT '' COMMENT '用户备注';

-- 迁移备注数据
insert into user_setting(uid,to_uid,remark) select uid,to_uid,remark from friend where remark<>''
on duplicate key update remark=values(remark);