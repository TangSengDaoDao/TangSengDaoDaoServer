-- +migrate Up

ALTER TABLE `group` ADD COLUMN is_upload_avatar smallint not null DEFAULT 0 COMMENT '群头像是否已经被用户上传';
