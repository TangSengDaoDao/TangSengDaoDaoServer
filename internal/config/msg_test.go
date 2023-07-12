package config

import (
	"fmt"
	"testing"
)

func TestSetting(t *testing.T) {
	setting := SettingFromUint8(160)
	fmt.Println(setting.Signal)
}
