-- +migrate Up

--  设备标识
create table `device_flag`
(
  id         bigint         not null primary key AUTO_INCREMENT,
  device_flag  smallint    not null default 0 COMMENT '设备标记 0. app 1.Web 2.PC',
  `weight`       integer       not null default 0 COMMENT '设备权重 值越大越优先',
  remark    VARCHAR(100)    not null default '' COMMENT '备注',
  created_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX `udx_device_flag` on `device_flag` (`device_flag`);


insert into device_flag(device_flag,`weight`,remark) values(2,'80000','PC');
insert into device_flag(device_flag,`weight`,remark) values(1,'70000','Web');
insert into device_flag(device_flag,`weight`,remark) values(0,'90000','手机');
