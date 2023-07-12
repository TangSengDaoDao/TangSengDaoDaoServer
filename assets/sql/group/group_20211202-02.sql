-- +migrate Up

ALTER TABLE `group_member` ADD COLUMN forbidden_expir_time integer NOT NULL DEFAULT 0 COMMENT '群成员禁言时长';
