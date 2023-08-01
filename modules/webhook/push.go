package webhook

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
)

// Payload 推送内容
type Payload interface {
	GetTitle() string   // 推送标题
	GetContent() string // 推送正文
	GetBadge() int      // 推送红点

	GetRTCPayload() RTCPayload // 获取rtc的payload
}

type RTCPayload interface {
	GetCallType() common.RTCCallType // 音视频呼叫类型
	GetOperation() string            // 音视频操作 invite: 邀请音视频 cancel：取消邀请
	GetFromUID() string              // 发起人的uid
}

// BasePayload 基础负载
type BasePayload struct {
	title   string
	content string
	badge   int
}

// GetTitle 推送标题
func (p *BasePayload) GetTitle() string {
	return p.title
}

// GetContent 推送正文
func (p *BasePayload) GetContent() string {
	return p.content
}

// GetBadge 推送红点
func (p *BasePayload) GetBadge() int {
	return p.badge
}

func (p *BasePayload) GetRTCPayload() RTCPayload {
	return nil
}

type BaseRTCPayload struct {
	BasePayload
	callType  common.RTCCallType
	operation string
	fromUID   string
}

func (b *BaseRTCPayload) GetCallType() common.RTCCallType {
	return b.callType
}

func (b *BaseRTCPayload) GetOperation() string {
	return b.operation
}

func (b *BaseRTCPayload) GetFromUID() string {
	return b.fromUID
}

func (b *BaseRTCPayload) GetRTCPayload() RTCPayload {
	return b
}

// Push Push
type Push interface {
	GetPayload(msg msgOfflineNotify, ctx *config.Context, toUser *user.Resp) (Payload, error)
	Push(deviceToken string, payload Payload) error
}
