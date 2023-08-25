package file

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// File 文件操作
type File struct {
	ctx *config.Context
	log.Log
	service IService
}

// New New
func New(ctx *config.Context) *File {
	return &File{
		ctx:     ctx,
		Log:     log.NewTLog("File"),
		service: NewService(ctx),
	}
}

// Route 路由
func (f *File) Route(r *wkhttp.WKHttp) {
	api := r.Group("/v1/file")
	{ // 文件上传
		// api.POST("/upload/*path", f.upload)
		// 组合图片
		api.POST("/compose/*path", f.makeImageCompose)
		// 获取文件
		api.GET("/preview/*path", f.getFile)
	}
	auth := r.Group("/v1/file", f.ctx.AuthMiddleware(r))
	{
		//获取上传文件地址
		auth.GET("/upload", f.getFilePath)
		//上传文件
		auth.POST("/upload", f.uploadFile)
	}
}

func (f *File) makeImageCompose(c *wkhttp.Context) {
	var imageURLs []string
	if err := c.BindJSON(&imageURLs); err != nil {
		f.Error("数据格式有误！", zap.Error(err))
		c.ResponseError(errors.New("数据格式有误！"))
		return
	}
	if len(imageURLs) <= 0 {
		c.ResponseError(errors.New("图片不能为空！"))
		return
	}
	if len(imageURLs) > 9 {
		c.ResponseError(errors.New("图片数量不能大于9！"))
		return
	}
	uploadPath := c.Param("path")
	// 下载并组合图片
	resultMap, err := f.service.DownloadAndMakeCompose(uploadPath, imageURLs)
	if err != nil {
		f.Error("组合图片失败！", zap.String("uploadPath", uploadPath), zap.Any("imageURLs", imageURLs), zap.Error(err))
		c.ResponseError(errors.New("组合图片失败！"))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"path": resultMap["fid"].(string),
	})
}

// 获取上传文件地址
func (f *File) getFilePath(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	uploadPath := c.Query("path")
	fileType := c.Query("type")
	err := f.checkReq(Type(fileType), uploadPath)
	if err != nil {
		c.ResponseError(err)
		return
	}
	var path string
	if Type(fileType) == TypeMomentCover {
		// 动态封面
		path = fmt.Sprintf("%s/file/upload?type=%s&path=/%s.png", f.ctx.GetConfig().External.APIBaseURL, fileType, loginUID)
	} else if Type(fileType) == TypeSticker {
		// 自定义表情
		path = fmt.Sprintf("%s/file/upload?type=%s&path=/%s/%s.gif", f.ctx.GetConfig().External.APIBaseURL, fileType, loginUID, util.GenerUUID())
	} else if Type(fileType) == TypeWorkplaceBanner {
		// 工作台横幅
		path = fmt.Sprintf("%s/file/upload?type=%s&path=/workplace/banner/%s", f.ctx.GetConfig().External.APIBaseURL, fileType, path)
	} else if Type(fileType) == TypeWorkplaceAppIcon {
		// 工作台appIcon
		path = fmt.Sprintf("%s/file/upload?type=%s&path=/workplace/appicon/%s", f.ctx.GetConfig().External.APIBaseURL, fileType, path)
	} else {
		path = fmt.Sprintf("%s/file/upload?type=%s&path=%s", f.ctx.GetConfig().External.APIBaseURL, fileType, uploadPath)
	}
	c.Response(map[string]string{
		"url": path,
	})
}

// 上传文件
func (f *File) uploadFile(c *wkhttp.Context) {
	uploadPath := c.Query("path")
	fileType := c.Query("type")
	contentType := c.DefaultPostForm("contenttype", "application/octet-stream")
	err := f.checkReq(Type(fileType), uploadPath)
	if err != nil {
		c.ResponseError(err)
		return
	}
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		f.Error("读取文件失败！", zap.Error(err))
		c.ResponseError(errors.New("读取文件失败！"))
		return
	}
	path := uploadPath
	if !strings.HasPrefix(path, "/") {
		path = fmt.Sprintf("/%s", path)
	}
	_, err = f.service.UploadFile(fmt.Sprintf("%s%s", fileType, path), contentType, func(w io.Writer) error {
		_, err := io.Copy(w, file)
		return err
	})
	defer file.Close()
	if err != nil {
		f.Error("上传文件失败！", zap.Error(err))
		c.ResponseError(errors.New("上传文件失败！"))
		return
	}

	c.Response(map[string]string{
		"path": fmt.Sprintf("file/preview/%s%s", fileType, path),
	})
}

// 获取文件
func (f *File) getFile(c *wkhttp.Context) {
	ph := c.Param("path")
	if ph == "" {
		c.Response(errors.New("访问路径不能为空"))
		return
	}
	filename := c.Query("filename")
	if filename == "" {
		paths := strings.Split(ph, "/")
		if len(paths) > 0 {
			filename = paths[len(paths)-1]
		}
	}
	downloadURL, err := f.service.DownloadURL(ph, filename)
	if err != nil {
		c.ResponseError(err)
		return
	}
	c.Redirect(http.StatusFound, downloadURL)
}

func (f *File) checkReq(fileType Type, path string) error {
	if fileType == "" {
		return errors.New("文件类型不能为空")
	}
	if path == "" && fileType != TypeMomentCover && fileType != TypeSticker {
		return errors.New("上传路径不能为空")
	}
	if fileType != TypeChat && fileType != TypeMoment && fileType != TypeMomentCover && fileType != TypeSticker && fileType != TypeReport && fileType != TypeChatBg && fileType != TypeCommon {
		return errors.New("文件类型错误")
	}
	return nil
}
