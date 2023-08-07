package report

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
)

// Report 举报
type Report struct {
	ctx *config.Context
	db  *db
}

// New 创建一个举报对象
func New(ctx *config.Context) *Report {
	return &Report{
		ctx: ctx,
		db:  newDB(ctx),
	}
}

// Route 配置路由规则
func (r *Report) Route(l *wkhttp.WKHttp) {
	v := l.Group("/v1/report")
	{
		v.GET("/categories", r.categoies)
		v.GET("/html", r.reportHTML)
	}
	auth := l.Group("/v1/reports", r.ctx.AuthMiddleware(l))
	{
		auth.POST("", r.report)

	}
}

func (r *Report) reportHTML(c *wkhttp.Context) {

	mode := c.Query("mode")
	if mode == "" {
		mode = "light"
	}

	c.Redirect(http.StatusMovedPermanently,
		fmt.Sprintf("%s/report.html?lang=%s&uid=%s&token=%s&channel_id=%s&channel_type=%s&mode=%s",
			r.ctx.GetConfig().External.H5BaseURL,
			c.Query("lang"),
			c.Query("uid"),
			c.Query("token"),
			c.Query("channel_id"),
			c.Query("channel_type"),
			mode,
		),
	)
}

// 举报
func (r *Report) report(c *wkhttp.Context) {
	var req reportReq
	if err := c.BindJSON(&req); err != nil {
		c.ResponseErrorf("请求数据格式有误！", err)
		return
	}
	if err := req.check(); err != nil {
		c.ResponseError(err)
		return
	}

	imgsStr := ""
	if len(req.Imgs) > 0 {
		imgsStr = strings.Join(req.Imgs, ",")
	}

	err := r.db.insert(&model{
		UID:         c.GetLoginUID(),
		CategoryNo:  req.CategoryNo,
		Imgs:        imgsStr,
		Remark:      req.Remark,
		ChannelID:   req.ChannelID,
		ChannelType: req.ChannelType,
	})
	if err != nil {
		c.ResponseErrorf("添加举报数据失败！", err)
		return
	}

	c.ResponseOK()

}

// 举报类别
func (r *Report) categoies(c *wkhttp.Context) {
	lang := c.Query("lang")
	if lang == "" {
		lang = c.GetHeader("Accept-Language")
	}

	en := false
	if strings.Contains(lang, "en") {
		en = true
	}

	categoryModels, err := r.db.queryCategoryAll()
	if err != nil {
		c.ResponseErrorf("查询类别失败！", err)
		return
	}

	rootCategories := r.findRootCategories(en, categoryModels)
	if len(rootCategories) > 0 {
		for _, rootCategory := range rootCategories {
			r.fillParentCategory(en, rootCategory, categoryModels)
		}
	}
	c.Response(rootCategories)
}

// 填充父类
func (r *Report) fillParentCategory(en bool, parent *categoryResp, categories []*categoryModel) {
	if len(categories) == 0 {
		return
	}
	for _, category := range categories {
		if parent.CategoryNo == category.ParentCategoryNo && parent.CategoryNo != category.CategoryNo {
			if parent.Children == nil {
				parent.Children = make([]*categoryResp, 0)
			}
			categoryNode := newCategoryResp(en, category)
			parent.Children = append(parent.Children, categoryNode)
			r.fillParentCategory(en, categoryNode, categories)
		}
	}
}

// 获取根元素
func (r *Report) findRootCategories(en bool, categories []*categoryModel) []*categoryResp {
	if len(categories) > 0 {
		categoryResps := []*categoryResp{}
		for _, category := range categories {
			if category.ParentCategoryNo == "" {
				categoryResps = append(categoryResps, newCategoryResp(en, category))
			}
		}
		return categoryResps
	}

	return nil
}

type categoryResp struct {
	CategoryNo       string          `json:"category_no"`
	CategoryName     string          `json:"category_name"`
	ParentCategoryNo string          `json:"parent_category_no"`
	Children         []*categoryResp `json:"children,omitempty"`
}

func newCategoryResp(en bool, m *categoryModel) *categoryResp {
	categoryName := m.CategoryName
	if en {
		categoryName = m.CategoryEname
	}
	return &categoryResp{
		CategoryNo:       m.CategoryNo,
		CategoryName:     categoryName,
		ParentCategoryNo: m.ParentCategoryNo,
	}
}

type reportReq struct {
	ChannelID   string   `json:"channel_id"`   // 频道id
	ChannelType uint8    `json:"channel_type"` // 频道类型
	CategoryNo  string   `json:"category_no"`  // 类别编号
	Imgs        []string `json:"imgs"`         // 举报图片内容
	Remark      string   `json:"remark"`       // 举报备注
}

func (r reportReq) check() error {
	if r.ChannelID == "" {
		return errors.New("频道ID不能为空！")
	}
	if r.ChannelType <= 0 {
		return errors.New("频道类型不能为空！")
	}
	if r.CategoryNo == "" {
		return errors.New("举报类别不能为空！")
	}
	return nil
}
