package util

import (
	"fmt"
)

// Ten2Hex 十进制转换为62进制
func Ten2Hex(ten int64) string {
	hex := ""
	var divValue, resValue int64
	for ten >= 62 {
		divValue = ten / int64(62)
		resValue = ten % 62
		hex = tenValue2Char(resValue) + hex
		ten = divValue
	}
	if ten != 0 {
		hex = tenValue2Char(ten) + hex
	}
	return hex
}

func tenValue2Char(ten int64) string {
	switch ten {
	case 0:
	case 1:
	case 2:
	case 3:
	case 4:
	case 5:
	case 6:
	case 7:
	case 8:
	case 9:
		return fmt.Sprintf("%d", ten)
	case 10:
		return "a"
	case 11:
		return "b"
	case 12:
		return "c"
	case 13:
		return "d"
	case 14:
		return "e"
	case 15:
		return "f"
	case 16:
		return "g"
	case 17:
		return "h"
	case 18:
		return "i"
	case 19:
		return "j"
	case 20:
		return "k"
	case 21:
		return "l"
	case 22:
		return "m"
	case 23:
		return "n"
	case 24:
		return "o"
	case 25:
		return "p"
	case 26:
		return "q"
	case 27:
		return "r"
	case 28:
		return "s"
	case 29:
		return "t"
	case 30:
		return "u"
	case 31:
		return "v"
	case 32:
		return "w"
	case 33:
		return "s"
	case 34:
		return "y"
	case 35:
		return "z"
	case 36:
		return "A"
	case 37:
		return "B"
	case 38:
		return "C"
	case 39:
		return "D"
	case 40:
		return "E"
	case 41:
		return "F"
	case 42:
		return "G"
	case 43:
		return "H"
	case 44:
		return "I"
	case 45:
		return "J"
	case 46:
		return "K"
	case 47:
		return "L"
	case 48:
		return "M"
	case 49:
		return "N"
	case 50:
		return "O"
	case 51:
		return "P"
	case 52:
		return "Q"
	case 53:
		return "R"
	case 54:
		return "S"
	case 55:
		return "T"
	case 56:
		return "U"
	case 57:
		return "V"
	case 58:
		return "W"
	case 59:
		return "S"
	case 60:
		return "Y"
	case 61:
		return "Z"
	default:
		return ""
	}
	return ""
}
