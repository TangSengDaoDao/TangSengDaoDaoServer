package config

import (
	"fmt"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
)

// rtc 挂断
func (c *Context) SendRTCCallResult(req P2pRtcMessageReq) error {

	content := ""
	resultType := req.ResultType
	switch resultType {
	case common.RTCResultTypeCancel:
		content = "通话取消"
	case common.RTCResultTypeMissed:
		content = "未接听"
	case common.RTCResultTypeRefuse:
		content = "通话拒绝"
	case common.RTCResultTypeHangup:
		content = fmt.Sprintf("通话时长：%s", formatSecond(req.Second))
	}

	return c.SendMessage(&MsgSendReq{
		Header: MsgHeader{
			RedDot: 1,
		},
		FromUID:     req.FromUID,
		ChannelID:   req.ToUID,
		ChannelType: uint8(common.ChannelTypePerson),
		Payload: []byte(util.ToJson(map[string]interface{}{
			"type":        common.VideoCallResult,
			"content":     content,
			"second":      req.Second,
			"call_type":   req.CallType,
			"result_type": req.ResultType,
		})),
	})
}

func formatSecond(t int) string {
	second := t % 60
	min := t / 60

	secondStr := fmt.Sprintf("%d", second)
	if second < 10 {
		secondStr = fmt.Sprintf("0%d", second)
	}
	minStr := fmt.Sprintf("%d", min)
	if min < 10 {
		minStr = fmt.Sprintf("0%d", min)
	}
	return fmt.Sprintf("%s:%s", minStr, secondStr)
}

type P2pRtcMessageReq struct {
	FromUID    string
	ToUID      string
	CallType   common.RTCCallType
	ResultType common.RTCResultType
	Second     int
}
