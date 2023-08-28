package file

// Type 文件类型
type Type string

const (
	// TypeChat 聊天文件
	TypeChat Type = "chat"
	// TypeMoment 动态文件
	TypeMoment Type = "moment"
	// TypeMomentCover 动态封面
	TypeMomentCover Type = "momentcover"
	// TypeSticker 表情
	TypeSticker Type = "sticker"
	// TypeReport 举报
	TypeReport Type = "report"
	// TypeCommon 通用
	TypeCommon Type = "common"
	// TypeChatBg 聊天背景
	TypeChatBg Type = "chatbg"
	// TypeWorkplaceBanner
	TypeWorkplaceBanner Type = "workplacebanner"
	// TypeWorkplaceAppIcon
	TypeWorkplaceAppIcon Type = "workplaceappicon"
)
