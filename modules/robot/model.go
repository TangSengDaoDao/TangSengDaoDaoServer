package robot

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
)

type robotEvent struct {
	EventID     int64               `json:"event_id,omitempty"` // 更新ID
	Message     *config.MessageResp `json:"message,omitempty"`  // 消息对象
	InlineQuery *InlineQuery        `json:"inline_query,omitempty"`
	Expire      int64               `json:"expire,omitempty"` // 过期时间
}

type InlineQuery struct {
	SID         string `json:"sid"`
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
	FromUID     string `json:"from_uid"` // 发送者uid
	Query       string `json:"query"`    // 查询关键字
	Offset      string `json:"offset"`
}

type InlineQueryResult struct {
	InlineQuerySID string `json:"inline_query_sid"`
	// 结果类型
	Type ResultType `json:"type"`
	// 结果ID
	ID         string                   `json:"id"`
	Results    []map[string]interface{} `json:"results,omitempty"`
	NextOffset string                   `json:"next_offset,omitempty"`
}

func (i *InlineQueryResult) Check() error {
	if i.Type == "" {
		return errors.New("type不能为空！")
	}
	return nil
}

type ResultType string

const (
	ResultTypeGIF ResultType = "gif"
)

// gif 结果
type GifResult struct {
	URL string `json:"url"` // gif完整路径
	// option
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type MessageReq struct {
	Setting     uint8                  `json:"setting"`
	ChannelID   string                 `json:"channel_id"`
	ChannelType uint8                  `json:"channel_type"`
	StreamNo    string                 `json:"stream_no"`
	Entities    []*Entitiy             `json:"entities"`
	Payload     map[string]interface{} `json:"payload"`
}

type Entitiy struct {
	Length int    `json:"length"`
	Offset int    `json:"offset"`
	Type   string `json:"type"`
}

type TypingReq struct {
	ChannelID   string `json:"channel_id"`
	ChannelType uint8  `json:"channel_type"`
}
