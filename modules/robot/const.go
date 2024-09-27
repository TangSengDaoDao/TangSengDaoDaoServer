package robot

// 机器人命令类型
type RobotCMDType string

const (
	None   RobotCMDType = "none"
	Inline RobotCMDType = "inline"
	Link   RobotCMDType = "link"
)

// 机器人状态
type RobotStatus int

const (
	Enable    RobotStatus = 1
	DisEnable RobotStatus = 0
)

var systemRobotMap = []*systemRobotMenu{
	{
		CMD:          "/基本信息",
		Remark:       "唐僧叨叨基本信息",
		ReplyContent: "唐僧叨叨是一款轻量级，高性能，重安全专注于私有化部署的开源即时通讯系统。唐僧叨叨官网：https://www.tsdaodao.com 各端演示地址：https://tsdaodao.com/guide/demo.html 悟空官网：https://githubim.com 在APP我的-设置-模块管理中关闭所有模块即是开源版本所有功能。",
		Type:         string(None),
	},
	{
		CMD:          "/添加好友",
		Remark:       "如何添加好友",
		ReplyContent: "您好，点击右上角【+】，选择【添加好友】-- 点击搜索 -- 输入用户的手机号、短号（任意一个添加即可）进行好友添加查找",
		Type:         string(None),
	},
	{
		CMD:          "/加群",
		Remark:       "如何创建群聊",
		ReplyContent: "您好，点击右上角【+】，选择【发起群聊】-- 选择联系人（最少三位联系人）后点击右上角“确定”按钮即可完成群聊创建",
		Type:         string(None),
	},
	{
		CMD:          "/添加表情",
		Remark:       "如何添加表情包或制作表情",
		ReplyContent: "您好，点击任意会话，点击【菜单栏】表情选项 -- 【搜索按钮】-- 【更多表情】可添加成套表情包，或搜索更多表情。点击【收藏选项】可制作单个表情。",
		Type:         string(None),
	},
	{
		CMD:          "/搜索GIF",
		Remark:       "聊天中如何发送GIF",
		ReplyContent: "您好，在聊天对话页面，点击输入框输入‘@’符号然后再输入‘gif’并输入一个空格，之后就是您要搜索的GIF的关键词。输入关键词后GIF图片就会显示在输入框上，点击任意GIF图片就能发送到当前会话了",
		Type:         string(None),
	},
	{
		CMD:          "/Android包下载",
		Remark:       "如何下载唐僧叨叨 Android包",
		ReplyContent: "您好，唐僧叨叨 Android应用下载地址 https://www.pgyer.com/tsdd",
		Type:         string(None),
	},
	// {
	// 	CMD:          "/解密失败",
	// 	Remark:       "收到消息提示【消息解密失败，无法查看】",
	// 	ReplyContent: "您好，消息解密失败是因为您和对方之间有谁卸载重装了软件，或者更换了聊天设备，导致密钥不再配对，解密不了消息。对此需要您和你的好友双方互发一条消息且双方都收到，完成密钥更新后就能正常聊天了",
	// 	Type:         string(None),
	// },
	{
		CMD:          "/未知消息",
		Remark:       "收到消息提示【未知消息，请升级客户端后查看】",
		ReplyContent: "您好，收到未知消息表示对方发送的消息类型您此版本无法识别该消息类型。您可以升级app后查看该消息",
		Type:         string(None),
	},
	{
		CMD:          "/电脑端登录",
		Remark:       "PC/Web登录",
		ReplyContent: "您好，在应用首页 点击【我的】模块 -- 点击 【电脑端登录】",
		Type:         string(None),
	},
	{
		CMD:          "/举报",
		Remark:       "举报用户或群",
		ReplyContent: "您好，在聊天对话页面，点击群/个人头像进入【聊天信息】页面，滑到页面底部点击【投诉】",
		Type:         string(None),
	},
}

type systemRobotMenu struct {
	CMD          string
	Remark       string
	ReplyContent string
	Type         string
}
