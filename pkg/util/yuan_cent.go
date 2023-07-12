package util

import "fmt"

// YuanToCent 元转分
func YuanToCent(yuan float64) int64 {
	dec, err := NewFromString(fmt.Sprintf("%0.2f", yuan))
	CheckErr(err)
	m, err := NewFromString("100")
	CheckErr(err)

	return dec.Mul(m).IntPart()
}

// CentToYuan 分转元
func CentToYuan(cent int64) float64 {
	centDec, err := NewFromString(fmt.Sprintf("%d", cent))
	CheckErr(err)
	mDec, err := NewFromString(fmt.Sprintf("%d", 100))
	CheckErr(err)

	result, _ := centDec.Div(mDec).Round(2).Float64()
	CheckErr(err)
	return result
}
