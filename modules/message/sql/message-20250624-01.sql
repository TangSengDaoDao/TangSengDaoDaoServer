-- +migrate Up

-- 优化 reminder_done 表结构以减少死锁
-- 1. 调整索引顺序，将更常用的 uid 放在前面
-- 2. 添加额外的索引来优化查询性能

-- 删除原有的唯一索引
DROP INDEX reminder_id_uidx ON reminder_done;

-- 创建新的唯一索引，调整字段顺序
CREATE UNIQUE INDEX reminder_done_uid_reminder_id_uidx ON reminder_done (uid, reminder_id);

-- 添加单独的 reminder_id 索引来优化某些查询
CREATE INDEX reminder_done_reminder_id_idx ON reminder_done (reminder_id);

-- 添加创建时间索引来优化时间范围查询
CREATE INDEX reminder_done_created_at_idx ON reminder_done (created_at);


