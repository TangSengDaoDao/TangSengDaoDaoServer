-- +migrate Up

-- 短编号
create table `shortno`
(
  id         bigint         not null primary key AUTO_INCREMENT,
  shortno        VARCHAR(40)    not null default '' COMMENT '唯一短编号',
  used       smallint       not null default 0 COMMENT '是否被用',
  hold       smallint       not null default 0 COMMENT '保留，保留的号码将不会再被分配',
  locked       smallint       not null default 0 COMMENT '是否被锁定，锁定了的短编号将不再被分配,直到解锁',
  business    VARCHAR(40)    not null default '' COMMENT '被使用的业务，比如 user',
  created_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp      not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX `udx_shortno` on `shortno` (`shortno`);