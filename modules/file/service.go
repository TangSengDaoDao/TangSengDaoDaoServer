package file

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	limlog "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/disintegration/imaging"
	"go.uber.org/zap"
)

type IUploadService interface {
	UploadFile(filePath string, contentType string, copyFileWriter func(io.Writer) error) (map[string]interface{}, error)
	// 获取下载地址
	DownloadURL(path string, filename string) (string, error)
}

// IService IService
type IService interface {
	IUploadService
	DownloadAndMakeCompose(uploadPath string, downloadURLs []string) (map[string]interface{}, error)
	DownloadImage(url string, ctx context.Context) (io.ReadCloser, error)
}

// NewService NewService
func NewService(ctx *config.Context) IService {
	var uploadService IUploadService
	service := ctx.GetConfig().FileService
	if service == config.FileServiceMinio {
		uploadService = NewServiceMinio(ctx)
	} else if service == config.FileServiceAliyunOSS {
		uploadService = NewServiceOSS(ctx)
	} else {
		uploadService = NewSeaweedFS(ctx)
	}
	return &Service{
		Log: log.NewTLog("Service"),
		ctx: ctx,
		downloadClient: &http.Client{
			Timeout: time.Second * 30,
		},
		uploadService: uploadService,
	}
	// return NewServiceMinio(ctx)
}

// Service Service
type Service struct {
	downloadClient *http.Client
	log.Log
	ctx           *config.Context
	uploadService IUploadService
}

func (s *Service) UploadFile(filePath string, contentType string, copyFileWriter func(io.Writer) error) (map[string]interface{}, error) {
	return s.uploadService.UploadFile(filePath, contentType, copyFileWriter)
}

func (s *Service) DownloadURL(path string, filename string) (string, error) {

	return s.uploadService.DownloadURL(path, filename)
}

func (s *Service) DownloadImage(url string, ctx context.Context) (io.ReadCloser, error) {
	reader, err := s.downloadImage(url, ctx)
	if err != nil {
		return nil, err
	}
	return reader, nil
}

// DownloadAndMakeCompose 下载并组合图片
func (s *Service) DownloadAndMakeCompose(uploadPath string, downloadURLs []string) (map[string]interface{}, error) {
	if len(downloadURLs) == 0 {
		return nil, nil
	}

	s.Debug("DownloadAndMakeCompose", zap.Strings("downloadURLs", downloadURLs))
	var w = sync.WaitGroup{}
	readers := make([]io.ReadCloser, 0, len(downloadURLs))
	cancels := make([]context.CancelFunc, 0, len(downloadURLs))
	for _, downloadURL := range downloadURLs {
		w.Add(1)
		go func(srcUrl string) {
			defer w.Done()
			timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
			cancels = append(cancels, cancel)
			imgReader, err := s.downloadImage(srcUrl, timeoutCtx)
			if err != nil {
				s.Error("下载图片失败！", zap.String("srcPath", srcUrl), zap.Error(err))
				return
			}
			if imgReader == nil {
				return
			}
			readers = append(readers, imgReader)

		}(downloadURL)
	}

	w.Wait()

	if len(readers) == 0 {
		return nil, errors.New("下载图片失败！")
	}
	s.Debug("readers-->", zap.Int("len", len(readers)))
	defer func() {
		for _, reader := range readers {
			if reader != nil {
				reader.Close()
			}
		}
		for _, cancel := range cancels {
			cancel()
		}
	}()
	// 拼图
	img, err := s.MakeCompose(readers)
	if err != nil {
		s.Error("组合图片失败！", zap.Error(err))
		return nil, err
	}

	// uploadURL := fmt.Sprintf("%s/public%s", s.ctx.GetConfig().UploadURL, uploadPath)
	// 上传文件
	resultMap, err := s.UploadFile(uploadPath, "image/png", func(w io.Writer) error {
		return png.Encode(w, img)
	})
	if err != nil {
		s.Error("上传文件失败！", zap.Error(err))
		return nil, err
	}
	return resultMap, nil
}

// MakeCompose 组合图片
func (s *Service) MakeCompose(srcImgFiles []io.ReadCloser) (image.Image, error) {

	var minwidth int = 42
	var minheight int = 42
	var maxwidth int = 64
	var maxheight int = 64
	var borderWidth = 4

	groupWith := 128 + borderWidth
	groupHeight := 128 + borderWidth
	// bounds := image.Rect(0, 0, groupWith, groupHeight)
	num := len(srcImgFiles)
	backgroundColor := color.RGBA{255, 255, 255, 255}
	newImg := imaging.New(groupWith, groupHeight, backgroundColor)
	newImg = opacityAdjust(newImg, 0) // 透明
	wx := 0                           //第几列
	hy := 0                           // 第几行
	for i := 0; i < num; i++ {

		if num == 3 {
			if i == 0 || i == 1 {
				hy += 1
				wx = 0
			}
		} else if num == 4 {
			if i > 0 && i%2 == 0 {
				hy += 1
				wx = 0
			}
		} else if num == 5 {
			if i == 1 || i == 0 {
				hy = 0
			} else if i == 2 {
				wx = 0
				hy += 1
			}
		} else if num == 7 {
			if i > 0 {
				if (i-1)%3 == 0 {
					hy = hy + 1
					wx = 0
				}
			}
		} else if num == 8 {
			if i > 1 {
				if (i-2)%3 == 0 {
					hy = hy + 1
					wx = 0
				}
			}
		} else {
			if i > 0 && i%3 == 0 {
				hy += 1
				wx = 0
			}
		}
		file := srcImgFiles[i]

		// fileExt := filepath.Ext(file.Name())

		var memberImg image.Image
		var err error
		var format string

		memberImg, format, err = image.Decode(file)
		if err != nil {
			s.Error("图片编码错误", zap.String("useFormat", format), zap.Error(err))
			continue
		}

		// 画圆角
		imgWidth := memberImg.Bounds().Dx()
		imgHeight := memberImg.Bounds().Dy()
		c := radius{p: image.Point{X: memberImg.Bounds().Dx(), Y: memberImg.Bounds().Dy()}, r: int(float32(imgWidth) * 0.4)}
		smallImgWithRadiuRGBA := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
		draw.DrawMask(smallImgWithRadiuRGBA, smallImgWithRadiuRGBA.Bounds(), memberImg, image.Point{}, &c, image.Point{}, draw.Over)

		var mbounds image.Rectangle
		var smallImgWithRadiu image.Image
		//缩略图
		if num >= 5 {
			smallImgWithRadiu = imaging.Resize(smallImgWithRadiuRGBA, minwidth, minheight, imaging.Lanczos)
		} else {
			smallImgWithRadiu = imaging.Resize(smallImgWithRadiuRGBA, maxwidth, maxheight, imaging.Lanczos)
		}

		var x, y int
		var width, height int
		if num == 1 {
			width = maxwidth
			height = width
			x, y = (groupWith-maxwidth)/2, (groupHeight-maxheight)/2
		} else if num == 2 { // 两张图
			width = (groupWith - borderWidth) / 2
			height = width
			if i == 0 {
				x, y = 0, (groupHeight-height)/2
			} else {
				x, y = borderWidth+width, (groupHeight-height)/2
			}
		} else if num == 3 {
			width = (groupWith - borderWidth) / 2
			height = width
			if i == 0 {
				x, y = (groupWith-width)/2, (groupHeight-(height*2+borderWidth))/2
			} else {
				x, y = wx*width+wx*borderWidth, (groupHeight-(height*2+borderWidth))/2+height+borderWidth
			}
		} else if num == 4 {
			width = (groupWith - borderWidth) / 2
			height = width
			x, y = wx*width+wx*borderWidth, (groupHeight-(height*2+borderWidth))/2+hy*(height+borderWidth)

		} else if num == 5 {
			width = (groupWith - borderWidth*2) / 3
			height = width
			if i == 0 || i == 1 {
				offset := (groupWith - (width*2 + borderWidth)) / 2
				x, y = offset+wx*width+wx*borderWidth, (groupHeight-(height*2+borderWidth))/2
			} else {
				x, y = wx*width+wx*borderWidth, (groupHeight-(height*2+borderWidth))/2+hy*(height+borderWidth)
			}
		} else if num == 6 {
			width = (groupWith - borderWidth*2) / 3
			height = width
			x, y = wx*width+wx*borderWidth, (groupHeight-(height*2+borderWidth))/2+hy*(height+borderWidth)
		} else if num == 7 {
			width = (groupWith - borderWidth*2) / 3
			height = width
			if i == 0 {
				offset := (groupWith - width) / 2
				x, y = offset+wx*width+wx*borderWidth, (groupHeight-(height*3+borderWidth*2))/2
			} else {
				x, y = wx*width+wx*borderWidth, hy*(height+borderWidth)
			}
		} else if num == 8 {
			width = (groupWith - borderWidth*2) / 3
			height = width
			if i == 0 || i == 1 {
				offset := (groupWith - (width*2 + borderWidth)) / 2
				x, y = offset+wx*width+wx*borderWidth, (groupHeight-(height*3+borderWidth*2))/2
			} else {
				x, y = wx*width+wx*borderWidth, (groupHeight-(height*3+borderWidth*2))/2+hy*(height+borderWidth)
			}
		} else if num == 9 {
			width = (groupWith - borderWidth*2) / 3
			height = width
			x, y = wx*width+wx*borderWidth, (groupHeight-(height*3+borderWidth*2))/2+hy*(height+borderWidth)
		}
		mbounds = image.Rect(x, y, width+x, height+y)
		smallImgWithRadiu = imaging.Resize(smallImgWithRadiuRGBA, width, height, imaging.Lanczos)
		draw.Draw(newImg, mbounds, smallImgWithRadiu, image.Point{}, draw.Src)
		wx++
	}

	return newImg, nil
}

// 圆角
type radius struct {
	p image.Point // 矩形右下角位置
	r int
}

func (c *radius) ColorModel() color.Model {
	return color.AlphaModel
}
func (c *radius) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.p.X, c.p.Y)
}

// 对每个像素点进行色值设置，分别处理矩形的四个角，在四个角的内切圆的外侧，色值设置为全透明，其他区域不透明
func (c *radius) At(x, y int) color.Color {
	var xx, yy, rr float64
	var inArea bool
	// left up
	if x <= c.r && y <= c.r {
		xx, yy, rr = float64(c.r-x)+0.5, float64(y-c.r)+0.5, float64(c.r)
		inArea = true
	}
	// right up
	if x >= (c.p.X-c.r) && y <= c.r {
		xx, yy, rr = float64(x-(c.p.X-c.r))+0.5, float64(y-c.r)+0.5, float64(c.r)
		inArea = true
	}
	// left bottom
	if x <= c.r && y >= (c.p.Y-c.r) {
		xx, yy, rr = float64(c.r-x)+0.5, float64(y-(c.p.Y-c.r))+0.5, float64(c.r)
		inArea = true
	}
	// right bottom
	if x >= (c.p.X-c.r) && y >= (c.p.Y-c.r) {
		xx, yy, rr = float64(x-(c.p.X-c.r))+0.5, float64(y-(c.p.Y-c.r))+0.5, float64(c.r)
		inArea = true
	}
	if inArea && xx*xx+yy*yy >= rr*rr {
		return color.Alpha{}
	}
	return color.Alpha{A: 255}
}

func imageTypeToRGBA64(m *image.NRGBA) *image.RGBA64 {
	bounds := (*m).Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()
	newRgba := image.NewRGBA64(bounds)
	for i := 0; i < dx; i++ {
		for j := 0; j < dy; j++ {
			colorRgb := (*m).At(i, j)
			r, g, b, a := colorRgb.RGBA()
			nR := uint16(r)
			nG := uint16(g)
			nB := uint16(b)
			alpha := uint16(a)
			newRgba.SetRGBA64(i, j, color.RGBA64{R: nR, G: nG, B: nB, A: alpha})
		}
	}
	return newRgba

}

// 将输入图像m的透明度变为原来的倍数。若原来为完成全不透明，则percentage = 0.5将变为半透明
func opacityAdjust(m *image.NRGBA, percentage float64) *image.NRGBA {
	bounds := m.Bounds()
	dx := bounds.Dx()
	dy := bounds.Dy()
	// newRgba := image.NewRGBA64(bounds)
	for i := 0; i < dx; i++ {
		for j := 0; j < dy; j++ {
			colorRgb := m.At(i, j)
			r, g, b, a := colorRgb.RGBA()
			opacity := uint16(float64(a) * percentage)
			//颜色模型转换，至关重要！
			v := m.ColorModel().Convert(color.NRGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: opacity})
			//Alpha = 0: Full transparent
			rr, gg, bb, aa := v.RGBA()
			m.SetRGBA64(i, j, color.RGBA64{R: uint16(rr), G: uint16(gg), B: uint16(bb), A: uint16(aa)})
		}
	}
	return m
}

var uploadClient *http.Client
var onceUploadClient sync.Once

// UploadFile 上传文件
func uploadFile(uploadURL, fileName string, copyFileWriter func(io.Writer) error) (map[string]interface{}, error) {
	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)
	fileWriter, err := bodyWriter.CreateFormFile("file", fileName)
	if err != nil {
		limlog.Error("构建formFile失败！", zap.Error(err))
		return nil, err
	}
	err = copyFileWriter(fileWriter)
	if err != nil {
		limlog.Error("复制文件内容失败！", zap.String("uploadURL", uploadURL), zap.Error(err))
		return nil, err
	}
	bodyWriter.Close()
	fRequest, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		limlog.Error("创建上传请求失败！", zap.String("uploadURL", uploadURL), zap.Error(err))
		return nil, err
	}
	fRequest.Header.Set("Content-Type", bodyWriter.FormDataContentType())
	onceUploadClient.Do(func() {
		uploadClient = &http.Client{
			Timeout: time.Second * 120,
		}
	})
	resp, err := uploadClient.Do(fRequest)
	if err != nil {
		limlog.Error("上传文件失败！", zap.String("uploadURL", uploadURL), zap.Error(err))
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("上传文件返回失败！")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		limlog.Error("文件上传返回状态有误！", zap.Int("status", resp.StatusCode), zap.String("uploadURL", uploadURL), zap.String("fileName", fileName))
		return nil, errors.New("文件上传返回状态有误！")
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		limlog.Error("读取上传返回的数据失败！", zap.Error(err))
		return nil, err
	}
	var resultMap map[string]interface{}
	err = util.ReadJsonByByte(respData, &resultMap)
	if err != nil {
		limlog.Error("上传返回的json格式有误！", zap.String("uploadURL", uploadURL), zap.String("fileName", fileName), zap.String("resp", string(respData)), zap.Error(err))
		return nil, err
	}
	return resultMap, err
}

func (s *Service) downloadImage(url string, ctx context.Context) (io.ReadCloser, error) {

	s.Debug("开始下载图片！", zap.String("url", url))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	// resp, err := s.downloadClient.Get(url)
	resp, err := client.Do(req)
	// resp, err := s.downloadClient.Do(req)
	if err != nil {
		s.Error("下载图片错误！", zap.String("url", url), zap.Error(err))
		return nil, err
	}
	if resp == nil {
		s.Error("没有返回数据，下载图片失败！", zap.String("url", url))
		return nil, errors.New("没有返回数据，下载图片失败！")
	}
	if resp.StatusCode != http.StatusOK {
		s.Error("下载图片返回状态有误！", zap.Int("status", resp.StatusCode), zap.String("url", url))
		return nil, errors.New("下载图片返回状态有误！")
	}
	return resp.Body, nil
}

func (s *Service) removeFile(filePaths []string) {
	for _, url := range filePaths {
		err := os.RemoveAll(url)
		if err != nil {
			s.Warn("移除文件失败！", zap.String("filePath", url), zap.Error(err))
		}
	}
}
