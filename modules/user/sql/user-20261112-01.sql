-- +migrate Up
ALTER TABLE `user` ADD COLUMN auth_version VARCHAR(40) NOT NULL DEFAULT '' COMMENT 'Session version rotated on password changes';
