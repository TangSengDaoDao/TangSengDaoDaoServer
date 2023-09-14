
-- +migrate Up

ALTER TABLE `user` ADD COLUMN web3_public_key VARCHAR(200) NOT NULL DEFAULT '' COMMENT 'web3公钥';
