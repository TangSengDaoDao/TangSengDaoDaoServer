package util

import (
	"gopkg.in/go-playground/assert.v1"
	"testing"
)

func TestStructAttrToUnderscore(t *testing.T) {

	names := AttrToUnderscore(&struct {
		MessageID uint64
		Name      string
		UserAge   int
	}{})
	assert.Equal(t, "message_id", names[0])
	assert.Equal(t, "name", names[1])
	assert.Equal(t, "user_age", names[2])
}
