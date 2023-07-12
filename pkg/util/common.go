package util

import (
	"fmt"
	"sort"
)

// CheckErr CheckErr
func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}

// Substr Substr
func Substr(str string, start, length int) string {
	if length == 0 {
		return ""
	}
	runeStr := []rune(str)
	lenStr := len(runeStr)

	if start < 0 {
		start = lenStr + start
	}
	if start > lenStr {
		start = lenStr
	}
	end := start + length
	if end > lenStr {
		end = lenStr
	}
	if length < 0 {
		end = lenStr + length
	}
	if start > end {
		start, end = end, start
	}
	return string(runeStr[start:end])
}

func objToStr(v interface{}) string {
	var strV string
	switch v.(type) {

	case int:
		strV = fmt.Sprintf("%d", v)
		break
	case uint:
		strV = fmt.Sprintf("%d", v)
		break
	case int64:
		strV = fmt.Sprintf("%d", v)
		break
	case uint64:
		strV = fmt.Sprintf("%d", v)
		break
	case int8:
		strV = fmt.Sprintf("%d", v)
		break
	case uint8:
		strV = fmt.Sprintf("%d", v)
		break
	case int16:
		strV = fmt.Sprintf("%d", v)
		break
	case uint16:
		strV = fmt.Sprintf("%d", v)
		break
	case int32:
		strV = fmt.Sprintf("%d", v)
		break
	case uint32:
		strV = fmt.Sprintf("%s", v)
		break
	case string:
		strV = fmt.Sprintf("%s", v)
		break
	case float32:
		strV = fmt.Sprintf("%s", v)
		break
	case float64:
		strV = fmt.Sprintf("%s", v)
		break
	default:
		strV = fmt.Sprintf("%s", v)

	}
	return strV
}

// Sign Sign
func Sign(params map[string]interface{}, appKey string) string {
	signStr := MapToQueryParamSort(params)
	return MD5(fmt.Sprintf("%s&key=%s", signStr, appKey))
}

// MapToQueryParamSort map 以 key1=value1 & key2=value2形式排序拼接
func MapToQueryParamSort(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}
	strs := ""
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := params[k]
		if v == "" {
			continue
		}
		//strs = strs+k+"&"+v
		strs = fmt.Sprintf("%s%s=%s%s", strs, k, objToStr(v), "&")
	}
	if len(strs) > 0 {
		strs = strs[0 : len(strs)-1]
	}
	return strs
}
