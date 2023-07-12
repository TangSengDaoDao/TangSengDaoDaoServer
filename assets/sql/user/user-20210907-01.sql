

-- +migrate Up

-- 用户身份表 （signal protocol使用）
create table `signal_identities`
(
  id         bigint        not null primary key AUTO_INCREMENT,
  uid         varchar(40) not null DEFAULT '', --  用户uid
  registration_id bigint  not null DEFAULT 0, -- 身份ID
  identity_key text     not null, -- 用户身份公钥
  signed_prekey_id integer not null DEFAULT 0, -- 签名key的id
  signed_pubkey text       not null,  -- 签名key的公钥
  signed_signature   text          not null, -- 由身份密钥签名的signed_pubkey
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);

CREATE UNIQUE INDEX  identities_index_id ON signal_identities(uid);


-- 一次性公钥
create table `signal_onetime_prekeys`
(
  id         bigint        not null primary key AUTO_INCREMENT,
  uid         varchar(40) not null DEFAULT '', -- 用户uid
  key_id     integer not null DEFAULT 0,
  pubkey   text           not null,   -- 公钥
  created_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP, -- 创建时间
  updated_at timeStamp     not null DEFAULT CURRENT_TIMESTAMP  -- 更新时间
);
CREATE UNIQUE INDEX  key_id_uid_index_id ON signal_onetime_prekeys(uid,key_id);