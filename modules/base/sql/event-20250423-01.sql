-- +migrate Up

ALTER TABLE `event` MODIFY COLUMN data VARCHAR(10000) not null default '';