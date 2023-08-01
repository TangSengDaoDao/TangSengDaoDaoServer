package webhook

import (
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/stretchr/testify/assert"
)

func TestHMSPush(t *testing.T) {
	hms := NewHMSPush("101827411", "649dedfb617dbd699715c05b9b430ce54013ad404cea0a5a2a16302fb01911a2", "com.xinbida.wukongchat")
	accessToken, _, err := hms.GetHMSAccessToken()
	assert.NoError(t, err)
	payloadInfo := &PayloadInfo{
		Title:   "title",
		Content: "content2222",
		Badge:   1,
	}
	err = hms.Push("ANqYJlGemvmj_H5U8L3629mb-OT7slBYJTdB8-vfpveu-oQzsJH8qtxCmEfzEiUemP1Gc7KV5M32rbiuhafNaZSu2VRPxAASLp3c_1_Ky-kUPN8FU06fZWHxLlA-6tJjCg", NewHMSPayload(payloadInfo, accessToken))
	assert.NoError(t, err)
}

func TestMIPush(t *testing.T) {
	mi := NewMIPush("2882303761519001722", "XIf41QWNIBRZPJVKUOOoYQ==", "com.xinbida.wukongchat", "")

	payloadInfo := &PayloadInfo{
		Title:   "title",
		Content: "content",
		Badge:   1,
	}

	err := mi.Push("deviceToken", NewMIPayload(payloadInfo, "11"))
	assert.NoError(t, err)
}

func TestOPPOPush(t *testing.T) {
	oppo := NewOPPOPush("30755393", "aece2f965eb64a9a82e01db87b23030e", "d7205515e1ab4fe6ace46f0f5df1105f", "dd6e2ec2e89e4669bb4afe4433b28ac1", &config.Context{})
	payloadInfo := &PayloadInfo{
		Title:   "标题",
		Content: "内容",
		Badge:   1,
	}
	err := oppo.Push("OPPO_CN_5831bbbefd00814c2bd82dbd40382869", NewOPPOPayload(payloadInfo, "11"))
	assert.NoError(t, err)
}

func TestVIVOPush(t *testing.T) {
	vivo := NewVIVOPush("105542118", "d7aacd9d36621e75a9efb7ce69b5c567", "be82d800-0078-42cf-91d2-4127781361a9", &config.Context{})
	payloadInfo := &PayloadInfo{
		Title:   "标题",
		Content: "内容",
		Badge:   1,
	}
	err := vivo.Push("16569158930074211800064", NewVIVOPayload(payloadInfo, "11"))
	assert.NoError(t, err)
}
