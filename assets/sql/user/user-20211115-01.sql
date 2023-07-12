-- +migrate Up

CREATE unique INDEX to_uid_uid on `user_setting` (uid, to_uid);

CREATE unique INDEX to_uid_uid on `friend` (uid, to_uid);