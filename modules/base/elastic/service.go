package elastic

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/message"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/olivere/elastic"
)

// Service Service
type Service struct {
	ctx *config.Context
	log.Log
	messageDB *message.DB
	db        *DB
	groupDB   *group.DB
}

// NewService NewService
func NewService(ctx *config.Context) *Service {

	return &Service{
		ctx:       ctx,
		Log:       log.NewTLog("Eastic"),
		messageDB: message.NewDB(ctx),
		db:        NewDB(ctx.DB()),
		groupDB:   group.NewDB(ctx),
	}
}

// Route Route
func (s *Service) Route(r *wkhttp.WKHttp) {

}

// PushMessageElasticIndexTask 创建消息索引的任务
func (s *Service) PushMessageElasticIndexTask(resps []msgResp) {
	if len(resps) <= 0 {
		return
	}
	for _, resp := range resps {
		if resp.ChannelType == common.ChannelTypePerson.Uint8() {
			elastic.NewBulkIndexRequest().Index(resp.FromUID).Type("message")
		}
	}
	elastic.NewBulkIndexRequest().Index("message")

}

// BulkIndexerItem BulkIndexerItem
type msgResp struct {
	MessageID   uint64 `json:"message_id"`   // 服务端的消息ID(全局唯一)
	FromUID     string `json:"from_uid"`     // 发送者UID
	ChannelID   string `json:"channel_id"`   // 频道ID
	ChannelType uint8  `json:"channel_type"` // 频道类型
	Timestamp   int64  `json:"timestamp"`    // 服务器消息时间戳(10位，到秒)
	Payload     []byte `json:"payload"`      // 消息内容
}
