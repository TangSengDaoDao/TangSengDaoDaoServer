package file

import (
	"bytes"
	"context"
	"fmt"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/storage"
	"go.uber.org/zap"
	"io"
)

type ServiceQiniu struct {
	log.Log
	ctx *config.Context
}

// NewServiceQiniu NewServiceQiniu
func NewServiceQiniu(ctx *config.Context) *ServiceQiniu {

	return &ServiceQiniu{
		Log: log.NewTLog("ServiceQiniu"),
		ctx: ctx,
	}
}

// UploadFile 上传文件
func (s *ServiceQiniu) UploadFile(filePath string, contentType string, copyFileWriter func(io.Writer) error) (map[string]interface{}, error) {

	qiniuCfg := s.ctx.GetConfig().Qiniu

	bucket := qiniuCfg.BucketName
	putPolicy := storage.PutPolicy{
		Scope: fmt.Sprintf("%s:%s", bucket, filePath),
	}
	mac := auth.New(qiniuCfg.AccessKey, qiniuCfg.SecretKey)
	upToken := putPolicy.UploadToken(mac)

	cfg := storage.Config{}
	// 空间对应的机房
	//cfg.Region = &storage.ZoneHuabei
	// 是否使用https域名
	cfg.UseHTTPS = false
	// 上传是否使用CDN上传加速
	cfg.UseCdnDomains = false

	formUploader := storage.NewFormUploader(&cfg)
	ret := storage.PutRet{}
	putExtra := storage.PutExtra{
		Params: map[string]string{},
	}

	data := bytes.NewBuffer(make([]byte, 0))
	err := copyFileWriter(data)
	if err != nil {
		s.Error("复制文件内容失败！", zap.Error(err))
		return nil, err
	}
	dataLen := int64(len(data.Bytes()))

	err = formUploader.Put(context.Background(), &ret, upToken, filePath, bytes.NewReader(data.Bytes()), dataLen, &putExtra)
	if err != nil {
		s.Error("上传失败", zap.Error(err))
	}
	fmt.Println(ret.Key, ret.Hash)
	return map[string]interface{}{
		"path": ret.Key,
	}, err
}

func (s *ServiceQiniu) DownloadURL(path string, filename string) (string, error) {
	qiniuCfg := s.ctx.GetConfig().Qiniu
	domain := qiniuCfg.URL
	key := path
	if key[0:1] == "/" {
		key = key[1:]
	}
	publicAccessURL := storage.MakePublicURL(domain, key)
	return publicAccessURL, nil
}
