package util

import (
	"fmt"
	"sort"
)

func GetSignStr(params map[string]interface{}) string {
	var signStr string
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		v := params[k]
		if v == "" {
			continue
		}
		vs := ObjToStr(v)

		signStr = fmt.Sprintf("%s=%s", k, vs)

		if i != len(keys)-1 {
			signStr += "&"
		}
	}
	return signStr
}

func ObjToStr(v interface{}) string {
	var strV string
	switch v.(type) {

	case int:
		strV = fmt.Sprintf("%d", v)
	case uint:
		strV = fmt.Sprintf("%d", v)
	case int64:
		strV = fmt.Sprintf("%d", v)
	case uint64:
		strV = fmt.Sprintf("%d", v)
	case int8:
		strV = fmt.Sprintf("%d", v)
	case uint8:
		strV = fmt.Sprintf("%d", v)
	case int16:
		strV = fmt.Sprintf("%d", v)
	case uint16:
		strV = fmt.Sprintf("%d", v)
	case int32:
		strV = fmt.Sprintf("%d", v)
	case uint32:
		strV = fmt.Sprintf("%s", v)
	case string:
		strV = fmt.Sprintf("%s", v)
	case float32:
		strV = fmt.Sprintf("%s", v)
	case float64:
		strV = fmt.Sprintf("%s", v)
	default:
		strV = fmt.Sprintf("%s", v)
	}
	return strV
}
