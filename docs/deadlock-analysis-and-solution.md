# 数据库死锁问题分析与解决方案

## 问题描述

在 `reminder_done` 表上发生了死锁，涉及两个并发事务同时尝试插入相同 `reminder_id` 但不同 `uid` 的记录。

### 死锁详情

- **表**: `reminder_done`
- **操作**: INSERT
- **冲突记录**: 
  - 事务1: `(31309,'d0e6b7c071ce4fee86605b0d61f014b3')`
  - 事务2: `(31309,'185ff2d130f9436f8d94cec93586d66b')`

## 根本原因分析

1. **并发插入**: 多个用户同时对同一个提醒项执行"完成"操作
2. **锁竞争**: 唯一索引 `(reminder_id,uid)` 导致的锁竞争
3. **错误处理不当**: 原代码中插入失败只记录警告，没有正确处理错误
4. **事务顺序**: 不同事务可能以不同顺序获取锁，导致死锁

## 解决方案

### 1. 代码层面优化

#### 1.1 使用 INSERT IGNORE
```sql
-- 原来的SQL
INSERT INTO reminder_done(reminder_id,uid) VALUES(?,?)

-- 优化后的SQL  
INSERT IGNORE INTO reminder_done(reminder_id,uid) VALUES(?,?)
```

#### 1.2 锁顺序优化
对 `reminder_id` 进行排序，确保所有事务按相同顺序获取锁：

```go
// 对 reminder_id 进行排序，确保事务按相同顺序获取锁，避免死锁
sortedIds := make([]int64, len(ids))
copy(sortedIds, ids)
sort.Slice(sortedIds, func(i, j int) bool {
    return sortedIds[i] < sortedIds[j]
})
```

#### 1.3 批量插入优化
使用批量插入减少锁持有时间：

```go
// 批量插入SQL
INSERT IGNORE INTO reminder_done(reminder_id,uid) VALUES (?,?),(?,?),(?,?)...
```

#### 1.4 错误处理改进
正确处理插入错误，而不是仅记录警告。

### 2. 数据库层面优化

#### 2.1 索引优化
- 删除原有索引: `(reminder_id,uid)`
- 创建新索引: `(uid,reminder_id)` - 将更常用的字段放在前面
- 添加辅助索引: `(reminder_id)`, `(created_at)`

#### 2.2 事务隔离级别
考虑调整事务隔离级别（如果业务允许）：
- 当前: REPEATABLE READ
- 可选: READ COMMITTED（减少锁持有时间）

### 3. 应用层面优化

#### 3.1 重试机制
对于死锁错误，实现指数退避重试：

```go
func retryOnDeadlock(fn func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        if isDeadlockError(err) && i < maxRetries-1 {
            time.Sleep(time.Duration(1<<i) * 100 * time.Millisecond)
            continue
        }
        return err
    }
    return nil
}
```

#### 3.2 业务逻辑优化
- 减少事务持有时间
- 避免在事务中执行耗时操作
- 考虑使用乐观锁替代悲观锁

## 实施步骤

1. **立即修复**: 应用代码层面的优化（已完成）
2. **数据库迁移**: 执行索引优化脚本
3. **监控**: 增加死锁监控和告警
4. **测试**: 在测试环境验证修复效果
5. **部署**: 逐步部署到生产环境

## 监控建议

1. **死锁监控**: 监控 `SHOW ENGINE INNODB STATUS` 中的死锁信息
2. **慢查询**: 监控相关表的查询性能
3. **锁等待**: 监控 `information_schema.innodb_locks` 表
4. **应用日志**: 监控应用层的数据库错误日志

## 预期效果

- 消除 `reminder_done` 表的死锁问题
- 提升并发插入性能
- 改善用户体验（减少操作失败）
- 提高系统稳定性
