package redis

import (
	"errors"
	"time"

	rd "github.com/go-redis/redis"
)

type Field struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type Conn struct {
	client *rd.Client
}

func New(addr string, password string) *Conn {
	c := &Conn{}
	c.client = rd.NewClient(&rd.Options{
		Addr:       addr,
		MaxRetries: 3, // 失败重试次数
		Password:   password,
	})
	return c
}

func (rc *Conn) Ping() (string, error) {
	return rc.client.Ping().Result()
}

func (rc *Conn) Set(key string, value interface{}) error {

	return rc.client.Set(key, value, 0).Err()
}

// expire 过期时间
func (rc *Conn) SetAndExpire(key string, value interface{}, expire time.Duration) error {

	return rc.client.Set(key, value, expire).Err()
}

func (rc *Conn) GetString(key string) (string, error) {
	val, err := rc.client.Get(key).Result()
	if err == rd.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return val, nil

}

func (rc *Conn) Del(key string) error {

	return rc.client.Del(key).Err()
}

// list大小
func (rc *Conn) Llen(key string) (int64, error) {
	val, err := rc.client.LLen(key).Result()
	if err == rd.Nil {
		return 0, nil
	}
	return val, err
}

func (rc *Conn) Lrange(key string, start, stop int64) ([]string, error) {
	val, err := rc.client.LRange(key, start, stop).Result()
	if err == rd.Nil {
		return nil, nil
	}
	return val, err
}

// LPOP key
// 移除并且返回 key 对应的 list 的第一个元素。
func (rc *Conn) Lpop(key string) (string, error) {
	val, err := rc.client.LPop(key).Result()
	if err == rd.Nil {
		return "", nil
	}
	return val, err

}

// SMembers  获取集合所有成员
func (rc *Conn) SMembers(key string) ([]string, error) {

	result, err := rc.client.SMembers(key).Result()
	if err == rd.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, err

}

//LREM key count value
//根据参数 count 的值，移除列表中与参数 value 相等的元素。
/**
count 的值可以是以下几种：
count > 0 : 从表头开始向表尾搜索，移除与 value 相等的元素，数量为 count 。
count < 0 : 从表尾开始向表头搜索，移除与 value 相等的元素，数量为 count 的绝对值。
count = 0 : 移除表中所有与 value 相等的值。
返回值：
	被移除元素的数量。
	因为不存在的 key 被视作空表(empty list)，所以当 key 不存在时， LREM 命令总是返回 0 。
*/
func (rc *Conn) Lrem(key string, count int64, value string) (int64, error) {
	return rc.client.LRem(key, count, value).Result()
}

/*
*
LTRIM key start stop
对一个列表进行修剪(trim)，就是说，让列表只保留指定区间内的元素，不在指定区间之内的元素都将被删除。
举个例子，执行命令 LTRIM list 0 2 ，表示只保留列表 list 的前三个元素，其余元素全部删除。
下标(index)参数 start 和 stop 都以 0 为底，也就是说，以 0 表示列表的第一个元素，以 1 表示列表的第二个元素，以此类推。
你也可以使用负数下标，以 -1 表示列表的最后一个元素， -2 表示列表的倒数第二个元素，以此类推。
当 key 不是列表类型时，返回一个错误。
*/
func (rc *Conn) Ltrim(key string, start, stop int64) (string, error) {

	return rc.client.LTrim(key, start, stop).Result()
}

/*
*
HGET key field
返回 key 指定的哈希集中该字段所关联的值
该字段所关联的值。当字段不存在或者 key 不存在时返回nil。
*/
func (rc *Conn) Hget(key, field string) (string, error) {
	val, err := rc.client.HGet(key, field).Result()
	if err == rd.Nil {
		return "", nil
	}
	return val, err
}

/*
*
HMGET key field [field ...]

返回哈希表 key 中，一个或多个给定域的值。

如果给定的域不存在于哈希表，那么返回一个 nil 值。

因为不存在的 key 被当作一个空哈希表来处理，所以对一个不存在的 key 进行 HMGET 操作将返回一个只带有 nil 值的表。
*/
func (rc *Conn) Hmget(key string, field ...string) ([]string, error) {

	results, err := rc.client.HMGet(key, field...).Result()
	if err == rd.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if results != nil {
		resultStrings := make([]string, 0)
		for _, result := range results {
			resultStrings = append(resultStrings, result.(string))
		}
		return resultStrings, nil
	}
	return nil, nil
}

func (rc *Conn) Hmset(key string, fieldValues ...string) error {

	if len(fieldValues)%2 != 0 {
		return errors.New("redis hmset操作失败【fieldValues不能为单数！】")
	}
	fieldValueMap := map[string]interface{}{}
	for i := 0; i < len(fieldValues); i += 2 {
		fieldValueMap[fieldValues[i]] = fieldValues[i+1]
	}

	return rc.client.HMSet(key, fieldValueMap).Err()
}

/*
*
获取在哈希表中指定 key 的所有字段和值
*/
func (rc *Conn) Hgetall(key string) (map[string]string, error) {

	m, err := rc.client.HGetAll(key).Result()
	if err == rd.Nil {
		return nil, nil
	}
	return m, err
}

// Expire 过期时间设置
func (rc *Conn) Expire(key string, expiration time.Duration) error {
	return rc.client.Expire(key, expiration).Err()
}

func (rc *Conn) Hset(key, field, value string) error {

	return rc.client.HSet(key, field, value).Err()
}

// 删除
func (rc *Conn) Hdel(key, field string) error {

	return rc.client.HDel(key, field).Err()
}

/*
*
HINCRBY
增加 key 指定的哈希集中指定字段的数值。如果 key 不存在，会创建一个新的哈希集并与 key 关联。
如果字段不存在，则字段的值在该操作执行前被设置为 0
HINCRBY 支持的值的范围限定在 64位 有符号整数
*/
func (rc *Conn) Hincrby(key, field string, increment int) (int64, error) {

	return rc.client.HIncrBy(key, field, int64(increment)).Result()

}

/*
*
SISMEMBER key member
返回成员 member 是否是存储的集合 key的成员.
如果member元素是集合key的成员，则返回1
如果member元素不是key的成员，或者集合key不存在，则返回0
*/
func (rc *Conn) Sismember(key, member string) (int, error) {

	result, err := rc.client.SIsMember(key, member).Result()
	if err != nil {
		return 0, err
	}
	if result {
		return 1, nil
	}
	return 0, err

}

func (rc *Conn) SAdd(key string, members ...interface{}) error {

	return rc.client.SAdd(key, members...).Err()
}

func (rc *Conn) SRem(key string, member interface{}) error {

	return rc.client.SRem(key, member).Err()
}

/*
*
ZADD key score member [[score member] [score member] ...]

将一个或多个 member 元素及其 score 值加入到有序集 key 当中。

如果某个 member 已经是有序集的成员，那么更新这个 member 的 score 值，并通过重新插入这个 member 元素，来保证该 member 在正确的位置上。

score 值可以是整数值或双精度浮点数。

如果 key 不存在，则创建一个空的有序集并执行 ZADD 操作。

当 key 存在但不是有序集类型时，返回一个错误
*/
func (rc *Conn) ZAdd(key string, scoremember ...interface{}) error {

	members := make([]rd.Z, 0)
	for i := 0; i < len(scoremember); i = i + 2 {
		score := scoremember[0].(float64)
		members = append(members, rd.Z{
			Score:  score,
			Member: scoremember[1],
		})
	}
	return rc.client.ZAdd(key, members...).Err()

}

func (rc *Conn) ZRem(key string, members ...interface{}) error {

	return rc.client.ZRem(key, members...).Err()

}

func (rc *Conn) ZRemRangeByScore(key string, min string, max string) error {
	return rc.client.ZRemRangeByScore(key, min, max).Err()
}

/*
*
ZRANGEBYSCORE key min max [WITHSCORES] [LIMIT offset count]

返回有序集 key 中，所有 score 值介于 min 和 max 之间(包括等于 min 或 max )的成员。有序集成员按 score 值递增(从小到大)次序排列。

具有相同 score 值的成员按字典序(lexicographical order)来排列(该属性是有序集提供的，不需要额外的计算)。

可选的 LIMIT 参数指定返回结果的数量及区间(就像SQL中的 SELECT LIMIT offset, count )，注意当 offset 很大时，定位 offset 的操作可能需要遍历整个有序集，此过程最坏复杂度为 O(N) 时间。

可选的 WITHSCORES 参数决定结果集是单单返回有序集的成员，还是将有序集成员及其 score 值一起返回。
*/
func (rc *Conn) ZRangeByScore(key string, zrangeBy rd.ZRangeBy) ([]string, error) {
	val, err := rc.client.ZRangeByScore(key, zrangeBy).Result()
	if err == rd.Nil {
		return nil, nil
	}
	return val, err
}

/*
*
Redis INCR命令用于将键的整数值递增1。如果键不存在，则在执行操作之前将其设置为0。

	如果键包含错误类型的值或包含无法表示为整数的字符串，
	则会返回错误。此操作限于64位有符号整数。
*/
func (rc *Conn) Incr(key string) (int64, error) {

	return rc.client.Incr(key).Result()
}

// Decr 递减
func (rc *Conn) Decr(key string) (int64, error) {

	return rc.client.Decr(key).Result()
}

/*
*
设置某个key的过期时间
*/
func (rc *Conn) SetExpire(key string, expire time.Duration) error {

	return rc.client.Expire(key, expire).Err()
}

/**
geoadd用来增加地理位置的坐标，可以批量添加地理位置，命令格式为：
GEOADD key longitude latitude member [longitude latitude member ...]
*/

func (rc *Conn) GeoAdd(key string, longitude, latitude float64, member string) error {

	return rc.client.GeoAdd(key, &rd.GeoLocation{
		Name:      member,
		Longitude: longitude,
		Latitude:  latitude,
	}).Err()

}

/*
*
georadius可以根据给定地理位置坐标获取指定范围内的地理位置集合。命令格式为：
GEORADIUS key longitude latitude radius [m|km|ft|mi] [WITHCOORD] [WITHDIST] [ASC|DESC] [WITHHASH] [COUNT count]
*/
func (rc *Conn) GeoRadius(key string, longitude, latitude float64, radius float64, unit string, params ...interface{}) ([]rd.GeoLocation, error) {

	return rc.client.GeoRadius(key, longitude, latitude, &rd.GeoRadiusQuery{
		Radius: radius,
		Unit:   unit,
	}).Result()

}

func (rc *Conn) MSet(keyValues ...string) error {
	return rc.client.MSet(keyValues).Err()
}

// BLPop  BLPOP key [key ...] timeout
// timeout 为0 表示无超时 一直阻塞
// BLPOP 是阻塞式列表的弹出原语。 它是命令 LPOP 的阻塞版本，这是因为当给定列表内没有任何元素可供弹出的时候，
// 连接将被 BLPOP 命令阻塞。 当给定多个 key 参数时，按参数 key 的先后顺序依次检查各个列表，弹出第一个非空列表的头元素
func (rc *Conn) BLPop(key string, timeout time.Duration) (string, error) {
	results, err := rc.client.BLPop(timeout, key).Result()
	if err != nil {
		return "", err
	}
	if len(results) > 1 {
		return results[1], nil
	}
	return "", nil
}

// BRPoplpush BRPOPLPUSH source destination timeout
// BRPOPLPUSH 是 RPOPLPUSH 的阻塞版本。 当 source 包含元素的时候，这个命令表现得跟 RPOPLPUSH 一模一样。 当 source 是空的时候，
// Redis将会阻塞这个连接，直到另一个客户端 push 元素进入或者达到 timeout 时限。 timeout 为 0 能用于无限期阻塞客户端。
func (rc *Conn) BRPoplpush(source string, destination string, timeout time.Duration) (string, error) {

	return rc.client.BRPopLPush(source, destination, timeout).Result()
}

// Redis列表是简单的字符串列表，按照插入顺序排序。你可以添加一个元素到列表的头部（左边）或者尾部（右边）
// 一个列表最多可以包含 232 - 1 个元素 (4294967295, 每个列表超过40亿个元素)。
func (rc *Conn) LPUSH(key string, values ...interface{}) (int64, error) {
	return rc.client.LPush(key, values...).Result()
}
