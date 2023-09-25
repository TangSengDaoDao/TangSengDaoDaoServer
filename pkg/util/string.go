package util

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// GenerUUID 生成uuid
func GenerUUID() string {

	return strings.Replace(NewV4().String(), "-", "", -1)
}

func isUpper(b byte) bool {
	return 'A' <= b && b <= 'Z'
}

func isLower(b byte) bool {
	return 'a' <= b && b <= 'z'
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

func toLower(b byte) byte {
	if isUpper(b) {
		return b - 'A' + 'a'
	}
	return b
}

// UnderscoreName 驼峰式写法转为下划线写法
func UnderscoreName(name string) string {
	var buf strings.Builder
	buf.Grow(len(name) * 2)

	for i := 0; i < len(name); i++ {
		buf.WriteByte(toLower(name[i]))
		if i != len(name)-1 && isUpper(name[i+1]) &&
			(isLower(name[i]) || isDigit(name[i]) ||
				(i != len(name)-2 && isLower(name[i+2]))) {
			buf.WriteByte('_')
		}
	}

	return buf.String()
}

// CamelName 下划线写法转为驼峰写法
func CamelName(name string) string {
	name = strings.Replace(name, "_", " ", -1)
	name = strings.Title(name)
	return strings.Replace(name, " ", "", -1)
}

// RemoveRepeatedElement 移除重复元素
func RemoveRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

// GetRandomSalt return len=8  salt
func GetRandomSalt() string {
	return GetRandomString(8)
}

// GetRandomString 生成随机字符串
func GetRandomString(num int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < num; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

// Names 注册用户随机名字
var names = []string{"独角王", "老鼋", "灵感大王", "如意真仙", "蝎女妖", "六耳猕猴", "罗刹女", "牛魔王",
	"羊力大仙", "鹿力大仙", "虎力大仙", "鳖龙", "红孩儿", "青狮道人", "熊山君", "特处士", "玉面公主",
	"九头虫", "黄眉老祖", "大蟒精", "赛太岁", "蜘蛛精", "多目怪", "青狮魔王", "白象魔王", "大鹏魔王", "虎威魔王",
	"狮吼魔王", "狮毛怪", "美后", "国丈", "地涌夫人", "金钱豹王", "黄狮精", "九灵元圣", "辟寒大王", "辟暑大王",
	"辟尘大王", "玄鹤老", "玉兔精", "蠹妖", "蛙怪", "麋妖", "古柏老", "灵龟老", "峰五老", "赤蛇精", "虺妖", "蚖妖",
	"蝮子怪", "蝎小妖", "狐妖", "凤管娘子", "鸾萧夫人", "七情大王", "六欲大王", "三尸魔王", "阴沉魔王", "独角魔王",
	"啸风魔王", "兴云魔王", "六耳魔王", "迷识魔王", "消阳魔王", "铄阴魔王", "耗气魔王", "黑鱼精", "蜂妖", "灵鹊",
	"玄武灵", "美蔚君", "福缘君", "善庆君", "孟浪魔王", "慌张魔王", "司视魔", "司听魔", "逐香魔", "具体魔", "驰神魔",
	"逐味魔", "千里眼", "顺风耳", "金童", "玉女", "雷公", "电母", "风伯", "雨师", "游奕灵官", "翊圣真君", "大力鬼王",
	"七仙女", "太白金星", "赤脚大仙", "嫦娥", "玉兔", "吴刚", "猪八戒", "孙悟空", "唐僧", "沙悟净", "白龙马", "九天玄女",
	"九曜星", "日游神", "夜游神", "太阴星君", "太阳星君", "武德星君", "佑圣真君", "李靖", "金吒", "木吒", "哪吒",
	"巨灵神", "月老", "左辅右弼", "二郎神杨戬", "萨真人", "文昌帝君", "增长天王", "持国天王", "多闻天王", "广目天王",
	"张道陵", "许逊", "邱弘济", "葛洪", "渔人", "林黛玉", "薛宝钗", "贾宝玉", "秦可卿", "贾巧姐", "王熙凤", "史湘云",
	"妙玉", "李纨", "贾惜春", "贾探春", "贾迎春", "贾元春", "王妈妈", "西门庆", "武松", "武大郎", "宋江", "鲁智深",
	"高俅", "闻太师", "卢俊义", "吴用", "公孙胜", "关胜", "林冲", "秦明", "呼延灼", "花荣", "阮小七", "燕青",
	"皇甫端", "扈三娘", "王英", "安道全", "金大坚", "萧峰", "段誉", "童猛", "陶宗旺", "郑天寿", "王定六", "段景住",
	"寅将军", "黑熊精", "白衣秀士", "凌虚子", "黄风怪", "白骨精", "奎木狼", "金角大王", "银角大王",
}

// GetRandomName 获取随机的名字
func GetRandomName() string {
	return names[rand.Intn(len(names)-1)]
}

func FormatSecondToDisplayTime(second int64) string {
	if second < 60 {
		return fmt.Sprintf("%d秒", second)
	}
	if second < 60*60 {
		return fmt.Sprintf("%d分钟", second/60)
	}
	if second < 60*60*24 {
		return fmt.Sprintf("%d小时", second/60/60)
	}
	if second < 60*60*24*30 {
		return fmt.Sprintf("%d天", second/60/60/24)
	}
	if second < 60*60*24*30*12 {
		return fmt.Sprintf("%d月", second/60/60/24/30)
	}
	return fmt.Sprintf("%d年", second/60/60/24/30/12)
}
