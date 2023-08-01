
-- +migrate Up

ALTER TABLE `robot` ADD COLUMN inline_on smallint not null DEFAULT 0 comment '是否开启行内搜索';
ALTER TABLE `robot` ADD COLUMN placeholder VARCHAR(40) not null DEFAULT '' comment '输入框占位符，开启行内搜索有效';