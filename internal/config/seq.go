package config

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/gocraft/dbr/v2"
)

var seqMap = map[string]*Seq{}
var seqLock sync.RWMutex
var seqStep int64 = 1000 // 序列号步长

// Seq Seq
type Seq struct {
	CurSeq int64
	MaxSeq int64
}

// GenSeq 生产序号
func (c *Context) GenSeq(flag string) int64 {
	seqLock.RLock()
	seq := seqMap[flag]
	seqLock.RUnlock()
	key := fmt.Sprintf("seq:%s", flag)
	if seq == nil {
		// seqStr, err := c.Cache().Get(key)
		seqM, err := querySeqWithKey(c.DB(), key)
		if err != nil {
			panic(err)
		}
		if seqM == nil {
			var currSeq int64 = 1000000 // TODO: 为了兼容老的（以前放redis的）所以这里起始seq尽量大点
			err = addOrUpdateSeq(c.DB(), &seqModel{
				Key:    key,
				Step:   int(seqStep),
				MinSeq: currSeq + seqStep,
			})
			// err = c.Cache().Set(key, fmt.Sprintf("%d", seqStep))
			if err != nil {
				panic(err)
			}

			seq = &Seq{
				CurSeq: currSeq,
				MaxSeq: currSeq + seqStep,
			}
		} else {
			seq = &Seq{
				CurSeq: seqM.MinSeq,
				MaxSeq: seqM.MinSeq,
			}
		}
		seqLock.Lock()
		seqMap[flag] = seq
		seqLock.Unlock()
	}
	if seq.CurSeq >= seq.MaxSeq { // 超过了最大序号
		// err := c.Cache().Set(key, fmt.Sprintf("%d", seq.CurSeq+seqStep))
		err := addOrUpdateSeq(c.DB(), &seqModel{
			Key:    key,
			Step:   int(seqStep),
			MinSeq: seq.CurSeq + seqStep,
		})
		if err != nil {
			panic(err)
		}
		seq.MaxSeq += seqStep
	}
	return atomic.AddInt64(&seq.CurSeq, 1)

}

func addOrUpdateSeq(session *dbr.Session, m *seqModel) error {
	_, err := session.InsertBySql("insert into `seq`(`key`,min_seq,step) values(?,?,?) ON DUPLICATE KEY UPDATE min_seq=VALUES(min_seq)", m.Key, m.MinSeq, m.Step).Exec()
	return err
}

func querySeqWithKey(session *dbr.Session, key string) (*seqModel, error) {
	var m *seqModel
	_, err := session.Select("*").From("seq").Where("`key`=?", key).Load(&m)
	return m, err
}

type seqModel struct {
	Key    string
	MinSeq int64
	Step   int
}
