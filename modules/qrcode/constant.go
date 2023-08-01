package qrcode

import (
	"bytes"
	"encoding/json"
)

// Forward Forward
type Forward string

const (
	// ForwardNative 原生跳转
	ForwardNative Forward = "native"
	// ForwardH5 h5跳转
	ForwardH5 Forward = "h5"
)

// MarshalJSON MarshalJSON
func (q Forward) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(string(q))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON UnmarshalJSON
func (q *Forward) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	*q = Forward(j)
	return nil
}

// HandlerType HandlerType
type HandlerType string

const (
	// HandlerTypeWebView HandlerTypeWebView
	HandlerTypeWebView HandlerType = "webview"
	// HandlerTypeGroup 群组
	HandlerTypeGroup HandlerType = "group"
	// HandlerTypeLoginConfirm 扫描登录确认
	HandlerTypeLoginConfirm HandlerType = "loginConfirm"
	// HandlerTypeUserInfo 跳转到用户资料页面
	HandlerTypeUserInfo HandlerType = "userInfo"
)

// MarshalJSON MarshalJSON
func (q HandlerType) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(string(q))
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

// UnmarshalJSON UnmarshalJSON
func (q *HandlerType) UnmarshalJSON(b []byte) error {
	var j string
	err := json.Unmarshal(b, &j)
	if err != nil {
		return err
	}
	*q = HandlerType(j)
	return nil
}
