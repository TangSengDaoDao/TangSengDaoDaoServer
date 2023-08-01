package file_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/file"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/testutil"
	"github.com/stretchr/testify/assert"
)

func TestOSSUpload(t *testing.T) {
	cfg := config.New()
	ctx := testutil.NewTestContext(cfg)
	cfg.OSS.Endpoint = "oss-cn-shanghai.aliyuncs.com"
	cfg.OSS.AccessKeyID = "xxxx"
	cfg.OSS.AccessKeySecret = "xxxxxx"

	service := file.NewServiceOSS(ctx)
	_, err := service.UploadFile("chat/zdd/fjj.txt", "*", func(writer io.Writer) error {
		_, err := writer.Write(bytes.NewBufferString("this is test content").Bytes())
		return err
	})
	assert.NoError(t, err)

}
