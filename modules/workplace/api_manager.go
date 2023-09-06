package workplace

import (
	"errors"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

type manager struct {
	ctx *config.Context
	log.Log
	db   *managerDB
	wpDB *db
}

func NewManager(ctx *config.Context) *manager {
	return &manager{
		ctx:  ctx,
		Log:  log.NewTLog("Workplace_manager"),
		db:   newManagerDB(ctx),
		wpDB: newDB(ctx),
	}
}

// Route 路由配置
func (m *manager) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1/manager", m.ctx.AuthMiddleware(r))
	{
		auth.POST("/workplace/category", m.addCategory)            // 添加分类
		auth.GET("/workplace/category", m.getCategory)             // 获取分类
		auth.PUT("/workplace/category/reorder", m.reorderCategory) // 排序分类
		auth.POST("/workplace/app", m.addApp)                      // 添加app
		auth.PUT("/workplace/app", m.updateApp)                    // 修改app
		auth.DELETE("/workplace/app", m.deleteApp)                 // 删除app
		auth.POST("/workplace/banner", m.addBanner)                // 添加横幅
		auth.DELETE("/workplace/banner", m.deleteBanner)           // 删除横幅
		auth.GET("/workplace/banner", m.getBanners)                // 获取横幅
		auth.PUT("/workplace/banner", m.updateBanner)              // 修改横幅
	}
}

func (m *manager) updateBanner(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var req updateBannerReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if req.BannerNo == "" {
		c.ResponseError(errors.New("横幅编号不能为空"))
		return
	}
	if strings.TrimSpace(req.Route) == "" {
		c.ResponseError(errors.New("横幅跳转地址不能为空"))
		return
	}
	if strings.TrimSpace(req.Cover) == "" {
		c.ResponseError(errors.New("横幅封面不能为空"))
		return
	}
	err = m.db.updateBanner(&bannerModel{
		BannerNo:    req.BannerNo,
		Cover:       req.Cover,
		Title:       req.Title,
		Description: req.Description,
		Route:       req.Route,
		JumpType:    req.JumpType,
	})
	if err != nil {
		m.Error("修改横幅错误", zap.Error(err))
		c.ResponseError(errors.New("修改横幅错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) getBanners(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	banners, err := m.wpDB.queryBanner()
	if err != nil {
		m.Error("查询横幅错误", zap.Error(err))
		c.ResponseError(errors.New("查询横幅错误"))
		return
	}
	list := make([]*bannerResp, 0)
	if len(banners) > 0 {
		for _, banner := range banners {
			list = append(list, &bannerResp{
				BannerNo:    banner.BannerNo,
				Title:       banner.Title,
				Cover:       banner.Cover,
				Description: banner.Description,
				JumpType:    banner.JumpType,
				Route:       banner.Route,
				CreatedAt:   banner.CreatedAt.String(),
			})
		}
	}
	c.Response(list)
}

func (m *manager) deleteBanner(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	bannerNo := c.Query("banner_no")
	if bannerNo == "" {
		c.ResponseError(errors.New("横幅编号不能为空"))
		return
	}
	err = m.db.deleteBanner(bannerNo)
	if err != nil {
		m.Error("删除横幅错误", zap.Error(err))
		c.ResponseError(errors.New("删除横幅错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) getCategory(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	list := make([]*categoryResp, 0)
	models, err := m.db.queryCategory()
	if err != nil {
		m.Error("查询所有分类错误", zap.Error(err))
		c.ResponseError(errors.New("查询所有分类错误"))
		return
	}
	if len(models) > 0 {
		for _, m := range models {
			list = append(list, &categoryResp{
				CategoryNo: m.CategoryNo,
				Name:       m.Name,
				SortNum:    m.SortNum,
			})
		}
	}
	c.Response(list)
}

func (m *manager) addBanner(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var req bannerReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if strings.TrimSpace(req.Route) == "" {
		c.ResponseError(errors.New("横幅跳转地址不能为空"))
		return
	}
	if strings.TrimSpace(req.Cover) == "" {
		c.ResponseError(errors.New("横幅封面不能为空"))
		return
	}
	err = m.db.insertBanner(&bannerModel{
		BannerNo:    util.GenerUUID(),
		Cover:       req.Cover,
		Title:       req.Title,
		Description: req.Description,
		JumpType:    req.JumpType,
		Route:       req.Route,
	})
	if err != nil {
		m.Error("新增横幅信息错误", zap.Error(err))
		c.ResponseError(errors.New("新增横幅信息错误"))
		return
	}
	c.ResponseOK()
}
func (m *manager) updateApp(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var req updateAppReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.checkAddAPP(); err != nil {
		c.ResponseError(err)
		return
	}
	if strings.TrimSpace(req.AppId) == "" {
		c.ResponseError(errors.New("修改的应用ID不能为空"))
		return
	}
	err = m.db.updateApp(&appModel{
		AppID:       req.AppId,
		AppCategory: req.AppCategory,
		CategoryNo:  req.CategoryNo,
		Icon:        req.Icon,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		JumpType:    req.JumpType,
		Route:       req.Route,
		IsPaidApp:   req.IsPaidApp,
	})
	if err != nil {
		m.Error("修改应用信息错误", zap.Error(err))
		c.ResponseError(errors.New("修改应用信息错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) deleteApp(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	categoryNo := c.Query("category_no")
	appId := c.Query("app_id")
	if categoryNo == "" || appId == "" {
		c.ResponseError(errors.New("分类ID和应用ID均不能为空"))
		return
	}
	err = m.db.deleteApp(appId, categoryNo)
	if err != nil {
		m.Error("删除应用错误", zap.Error(err))
		c.ResponseError(errors.New("删除应用错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) reorderCategory(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	type reqVO struct {
		CategoryNos []string `json:"category_nos"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	tx, _ := m.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	var tempSortNum = len(req.CategoryNos)
	for _, categoryNo := range req.CategoryNos {
		err := m.db.updateCategorySortNumWithTx(categoryNo, tempSortNum, tx)
		if err != nil {
			tx.Rollback()
			m.Error("修改分类排序错误", zap.Error(err))
			c.ResponseError(errors.New("修改分类排序错误"))
			return
		}
		tempSortNum--
	}
	err = tx.Commit()
	if err != nil {
		m.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		tx.Rollback()
		return
	}
	c.ResponseOK()
}

func (m *manager) addApp(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	var req appReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if err := req.checkAddAPP(); err != nil {
		c.ResponseError(err)
		return
	}
	app, err := m.db.queryAppWithAppNameAndCategoryNo(req.Name, req.CategoryNo)
	if err != nil {
		m.Error("查询此分类下app是否存在此名称错误", zap.Error(err))
		c.ResponseError(errors.New("查询此分类下app是否存在此名称错误"))
		return
	}
	if app != nil && len(app.AppID) > 0 {
		c.ResponseError(errors.New("此应用名已存在"))
		return
	}

	cateogry, err := m.db.queryCategoryWithCategoryNo(req.CategoryNo)
	if err != nil {
		m.Error("查询此分类是否存在错误", zap.Error(err))
		c.ResponseError(errors.New("查询此分类是否存在错误"))
		return
	}

	if cateogry == nil || len(cateogry.CategoryNo) == 0 {
		c.ResponseError(errors.New("此分类不存在"))
		return
	}
	err = m.db.insertAPP(&appModel{
		AppID:       util.GenerUUID(),
		Name:        req.Name,
		Description: req.Description,
		CategoryNo:  req.CategoryNo,
		Icon:        req.Icon,
		AppCategory: req.AppCategory,
		Status:      1,
		JumpType:    req.JumpType,
		Route:       req.Route,
		IsPaidApp:   req.IsPaidApp,
	})
	if err != nil {
		m.Error("新增应用错误", zap.Error(err))
		c.ResponseError(errors.New("新增应用错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) addCategory(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	type reqVO struct {
		Name string `json:"name"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	category, err := m.db.queryCateogryWithName(req.Name)
	if err != nil {
		m.Error("通过分类名查询分类错误", zap.Error(err))
		c.ResponseError(errors.New("通过分类名查询分类错误"))
		return
	}
	if category != nil && len(category.CategoryNo) > 0 {
		c.ResponseError(errors.New("该分类名称已存在"))
		return
	}
	maxSortNumCategory, err := m.db.queryMaxSortNumCategory()
	if err != nil {
		m.Error("查询最大序号分类错误", zap.Error(err))
		c.ResponseError(errors.New("查询最大序号分类错误"))
		return
	}
	var sortNum = 1
	if maxSortNumCategory != nil && len(maxSortNumCategory.CategoryNo) > 0 {
		sortNum = maxSortNumCategory.SortNum + 1
	}
	err = m.db.insertCategory(&categoryModel{
		Name:       req.Name,
		CategoryNo: util.GenerUUID(),
		SortNum:    sortNum,
	})
	if err != nil {
		m.Error("新增分类错误", zap.Error(err))
		c.ResponseError(errors.New("新增分类错误"))
		return
	}
	c.ResponseOK()
}

func (req *appReq) checkAddAPP() error {
	if strings.TrimSpace(req.Name) == "" {
		return errors.New("应用名称不能为空")
	}
	if strings.TrimSpace(req.CategoryNo) == "" {
		return errors.New("应用分类ID不能为空")
	}
	if strings.TrimSpace(req.Route) == "" {
		return errors.New("应用打开地址不能为空")
	}
	if strings.TrimSpace(req.Icon) == "" {
		return errors.New("应用logo不能为空")
	}
	return nil
}

type updateAppReq struct {
	AppId  string `json:"app_id"` //应用ID
	Status int    `json:"status"` // 1.可用 0.禁用
	appReq
}
type appReq struct {
	CategoryNo  string `json:"category_no"`  //  所属分类编号
	Icon        string `json:"icon"`         // 应用icon
	Name        string `json:"name"`         // 应用名称
	Description string `json:"description"`  // 应用介绍
	AppCategory string `json:"app_category"` // 应用分类 [‘机器人’ ‘客服’]
	JumpType    int    `json:"jump_type"`    // 打开方式 0.网页 1.原生
	Route       string `json:"route"`        // 打开地址
	IsPaidApp   int    `json:"is_paid_app"`  // 是否为付费应用 0.否 1.是
}
type bannerReq struct {
	Cover       string `json:"cover"`       // 封面
	Title       string `json:"title"`       // 标题
	Description string `json:"description"` // 介绍
	JumpType    int    `json:"jump_type"`   // 打开方式 0.网页 1.原生
	Route       string `json:"route"`       // 打开地址
}
type updateBannerReq struct {
	BannerNo string `json:"banner_no"` // 横幅编号
	bannerReq
}
