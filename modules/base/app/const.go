package app

// Status app状态
type Status int

const (
	// StatusDisable app被禁用
	StatusDisable Status = iota
	// StatusEnable app被启用
	StatusEnable
)

func (s Status) Int() int {
	return int(s)
}
