package file

import (
	"bytes"
	"io"
	"net/url"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"go.uber.org/zap"
)

type ServiceOSS struct {
	log.Log
	ctx *config.Context
}

// NewServiceOSS NewServiceOSS
func NewServiceOSS(ctx *config.Context) *ServiceOSS {

	return &ServiceOSS{
		Log: log.NewTLog("ServiceOSS"),
		ctx: ctx,
	}
}

// UploadFile 上传文件
func (s *ServiceOSS) UploadFile(filePath string, contentType string, copyFileWriter func(io.Writer) error) (map[string]interface{}, error) {
	ossCfg := s.ctx.GetConfig().OSS
	client, err := oss.New(ossCfg.Endpoint, ossCfg.AccessKeyID, ossCfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucketName := s.ctx.GetConfig().OSS.BucketName
	// strs := strings.Split(filePath, "/")
	// if len(strs) > 0 {
	// 	bucketName = strs[0]
	// }

	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		err = client.CreateBucket(bucketName, oss.ACL(oss.ACLPublicRead))
		if err != nil {
			return nil, err
		}
		bucket, err = client.Bucket(bucketName)
		if err != nil {
			return nil, err
		}
	}
	buff := bytes.NewBuffer(make([]byte, 0))
	err = copyFileWriter(buff)
	if err != nil {
		s.Error("复制文件内容失败！", zap.Error(err))
		return nil, err
	}
	err = bucket.PutObject(filePath, buff, oss.ContentType(contentType), oss.ContentLength(int64(len(buff.Bytes()))))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{}, nil
}

func (s *ServiceOSS) DownloadURL(path string, filename string) (string, error) {
	ossCfg := s.ctx.GetConfig().OSS

	rpath, _ := url.JoinPath(ossCfg.BucketURL, path)
	return rpath, nil
}
