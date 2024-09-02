package workplace

import (
	"errors"
	"strconv"
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
	auth := r.Group("/v1/manager/workplace", m.ctx.AuthMiddleware(r))
	{
		auth.POST("/category", m.addCategory)                                    // 添加分类
		auth.GET("/category", m.getCategory)                                     // 获取分类
		auth.PUT("/category/reorder", m.reorderCategory)                         // 排序分类
		auth.DELETE("/categorys/:category_no", m.deleteCategory)                 // 删除分类
		auth.PUT("/categorys/:category_no", m.updateCategory)                    // 修改分类
		auth.GET("/categorys/:category_no/app", m.getCategoryApps)               // 获取分类下app
		auth.PUT("/categorys/:category_no/app/reorder", m.reorderCategoryApp)    // 排序分类下app
		auth.POST("/categorys/:category_no/app", m.addCategoryApp)               // 新增分类下app
		auth.DELETE("/categorys/:category_no/apps/:app_id", m.deleteCategoryApp) // 删除分类下app
		auth.POST("/app", m.addApp)                                              // 添加app
		auth.GET("/app", m.getApps)                                              // 获取app
		auth.PUT("/apps/:app_id", m.updateApp)                                   // 修改app
		auth.DELETE("/apps/:app_id", m.deleteApp)                                // 删除app
		auth.POST("/banner", m.addBanner)                                        // 添加横幅
		auth.GET("/banner", m.getBanners)                                        // 获取横幅
		auth.DELETE("/banners/:banner_no", m.deleteBanner)                       // 删除横幅
		auth.PUT("/banners/:banner_no", m.updateBanner)                          // 修改横幅
		auth.PUT("/banner/reorder", m.reorderBanner)                             // 排序横幅
	}
}

// 排序横幅
func (m *manager) reorderBanner(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	type reqVO struct {
		BannerNos []string `json:"banner_nos"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.Error("开启事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事务失败！"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	var tempSortNum = len(req.BannerNos)
	for _, bannerNo := range req.BannerNos {
		err := m.db.updateBannerSortNumWithTx(bannerNo, tempSortNum, tx)
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

// 编辑分类
func (m *manager) updateCategory(c *wkhttp.Context) {
	categoryNo := c.Param("category_no")
	if categoryNo == "" {
		c.ResponseError(errors.New("分类ID不能为空"))
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
	category, err := m.db.queryCategoryWithNo(categoryNo)
	if err != nil {
		m.Error("获取分类错误", zap.Error(err))
		c.ResponseError(errors.New("获取分类错误"))
		return
	}
	if category == nil {
		c.ResponseError(errors.New("该分类不存在"))
		return
	}
	category.Name = req.Name
	err = m.db.updateCategory(category)
	if err != nil {
		m.Error("修改分类错误", zap.Error(err))
		c.ResponseError(errors.New("修改分类错误"))
		return
	}
	c.ResponseOK()
}

// 删除分类
func (m *manager) deleteCategory(c *wkhttp.Context) {
	categoryNo := c.Param("category_no")
	if categoryNo == "" {
		c.ResponseError(errors.New("分类ID不能为空"))
		return
	}
	err := m.db.deleteCategory(categoryNo)
	if err != nil {
		m.Error("删除分类错误", zap.Error(err))
		c.ResponseError(errors.New("删除分类错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) getApps(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	page := c.Query("page_index")
	size := c.Query("page_size")
	keyword := c.Query("keyword")
	pageIndex, _ := strconv.Atoi(page)
	pageSize, _ := strconv.Atoi(size)
	var apps []*appModel
	var count int64
	if keyword == "" {
		apps, err = m.db.queryAppWithPage(uint64(pageSize), uint64(pageIndex))
		if err != nil {
			m.Error("查询所有应用错误", zap.Error(err))
			c.ResponseError(errors.New("查询所有应用错误"))
			return
		}
		count, err = m.db.queryAppCount()
		if err != nil {
			m.Error("查询总数量错误", zap.Error(err))
			c.ResponseError(errors.New("查询总数量错误"))
			return
		}
	} else {
		apps, err = m.db.searchApp(keyword, uint64(pageSize), uint64(pageIndex))
		if err != nil {
			m.Error("搜索应用错误", zap.Error(err))
			c.ResponseError(errors.New("搜索应用错误"))
			return
		}
		count, err = m.db.queryAppCountWithKeyWord(keyword)
		if err != nil {
			m.Error("查询总数量错误", zap.Error(err))
			c.ResponseError(errors.New("查询总数量错误"))
			return
		}
	}

	list := make([]*appDetailResp, 0)
	if len(apps) > 0 {
		for _, app := range apps {
			list = append(list, &appDetailResp{
				AppID:       app.AppID,
				AppCategory: app.AppCategory,
				AppRoute:    app.AppRoute,
				WebRoute:    app.WebRoute,
				IsPaidApp:   app.IsPaidApp,
				Name:        app.Name,
				Description: app.Description,
				Icon:        app.Icon,
				Status:      app.Status,
				JumpType:    app.JumpType,
			})
		}
	}
	c.Response(map[string]interface{}{
		"count": count,
		"list":  list,
	})
}

func (m *manager) deleteCategoryApp(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	categoryNo := c.Param("category_no")
	appId := c.Param("app_id")
	if categoryNo == "" {
		c.ResponseError(errors.New("分类编号不能为空"))
		return
	}
	if appId == "" {
		c.ResponseError(errors.New("应用ID不能为空"))
		return
	}
	err = m.db.deleteCategoryApp(appId, categoryNo)
	if err != nil {
		m.Error("删除分类下app错误", zap.Error(err))
		c.ResponseError(errors.New("删除分类下app错误"))
		return
	}
	c.ResponseOK()
}

func (m *manager) addCategoryApp(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	categoryNo := c.Param("category_no")
	type reqVO struct {
		AppIds []string `json:"app_ids"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if categoryNo == "" {
		c.ResponseError(errors.New("分类编号不能为空"))
		return
	}
	if len(req.AppIds) == 0 {
		c.ResponseError(errors.New("应用ID不能为空"))
		return
	}
	appList, err := m.wpDB.queryAppWithAppIds(req.AppIds)
	if err != nil {
		m.Error("查询一批应用错误", zap.Error(err))
		c.ResponseError(errors.New("查询一批应用错误"))
		return
	}
	if len(appList) != len(req.AppIds) {
		c.ResponseError(errors.New("添加的应用不存在"))
		return
	}
	apps, err := m.wpDB.queryAppWithCategroyNo(categoryNo)
	if err != nil {
		m.Error("查询该分类下应用错误", zap.Error(err))
		c.ResponseError(errors.New("查询该分类下应用错误"))
		return
	}
	saveIds := make([]string, 0)
	for _, appId := range req.AppIds {
		var isAdd = true
		if len(apps) > 0 {
			for _, app := range apps {
				if appId == app.AppID {
					isAdd = false
					break
				}
			}
		}
		if isAdd {
			saveIds = append(saveIds, appId)
		}
	}
	if len(saveIds) == 0 {
		c.ResponseOK()
		return
	}
	maxSortNumApp, err := m.db.queryMaxSortNumCategoryApp(categoryNo)
	if err != nil {
		m.Error("查询分类应用最大序号错误", zap.Error(err))
		c.ResponseError(errors.New("查询分类应用最大序号错误"))
		return
	}
	maxNum := 0
	if maxSortNumApp != nil {
		maxNum = maxSortNumApp.SortNum
	}
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.Error("开启事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事务失败！"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	var tempSortNum = len(saveIds) + maxNum
	for _, appId := range saveIds {
		err := m.db.insertCategoryAppWithTx(&categoryAppModel{
			AppId:      appId,
			SortNum:    tempSortNum,
			CategoryNo: categoryNo,
		}, tx)
		if err != nil {
			tx.Rollback()
			m.Error("添加分类下app错误", zap.Error(err))
			c.ResponseError(errors.New("添加分类下app错误"))
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
func (m *manager) reorderCategoryApp(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	categoryNo := c.Param("category_no")
	type reqVO struct {
		AppIds []string `json:"app_ids"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if categoryNo == "" {
		c.ResponseError(errors.New("分类编号不能为空"))
		return
	}
	if len(req.AppIds) == 0 {
		c.ResponseError(errors.New("应用ID不能为空"))
		return
	}
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.Error("开启事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事务失败！"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	var tempSortNum = len(req.AppIds)
	for _, appId := range req.AppIds {
		err := m.db.updateCategoryAppSortNumWithTx(categoryNo, appId, tempSortNum, tx)
		if err != nil {
			tx.Rollback()
			m.Error("修改分类下app排序错误", zap.Error(err))
			c.ResponseError(errors.New("修改分类下app排序错误"))
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

func (m *manager) getCategoryApps(c *wkhttp.Context) {
	err := c.CheckLoginRole()
	if err != nil {
		c.ResponseError(err)
		return
	}
	categoryNo := c.Param("category_no")
	if categoryNo == "" {
		c.ResponseError(errors.New("分类编号不能为空"))
		return
	}
	apps, err := m.wpDB.queryAppWithCategroyNo(categoryNo)
	if err != nil {
		m.Error("获取分类下的app错误", zap.Error(err))
		c.ResponseError(errors.New("获取分类下的app错误"))
		return
	}

	list := make([]*appDetailResp, 0)
	if len(apps) > 0 {
		for _, app := range apps {
			list = append(list, &appDetailResp{
				AppID:       app.AppID,
				SortNum:     app.SortNum,
				Icon:        app.Icon,
				Name:        app.Name,
				Description: app.Description,
				JumpType:    app.JumpType,
				AppCategory: app.AppCategory,
				Status:      app.Status,
				AppRoute:    app.AppRoute,
				WebRoute:    app.WebRoute,
				IsPaidApp:   app.IsPaidApp,
			})
		}
	}
	c.Response(list)
}

func (m *manager) updateBanner(c *wkhttp.Context) {
	err := c.CheckLoginRoleIsSuperAdmin()
	if err != nil {
		c.ResponseError(err)
		return
	}
	bannerNo := c.Param("banner_no")
	var req bannerReq
	if err := c.BindJSON(&req); err != nil {
		m.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if bannerNo == "" {
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
		BannerNo:    bannerNo,
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
				SortNum:     banner.SortNum,
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
	bannerNo := c.Param("banner_no")
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
	appId := c.Param("app_id")
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
	if strings.TrimSpace(appId) == "" {
		c.ResponseError(errors.New("修改的应用ID不能为空"))
		return
	}
	app, err := m.wpDB.queryAppWithAppId(appId)
	if err != nil {
		m.Error("查询应用信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询应用信息错误"))
		return
	}
	if app == nil {
		c.ResponseError(errors.New("该应用不存在"))
		return
	}
	app.AppCategory = req.AppCategory
	app.Icon = req.Icon
	app.Name = req.Name
	app.Description = req.Description
	app.Status = req.Status
	app.JumpType = req.JumpType
	app.AppRoute = req.AppRoute
	app.WebRoute = req.WebRoute
	app.IsPaidApp = req.IsPaidApp
	err = m.db.updateApp(app)
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
	appId := c.Param("app_id")
	if appId == "" {
		c.ResponseError(errors.New("分类ID和应用ID均不能为空"))
		return
	}
	app, err := m.wpDB.queryAppWithAppId(appId)
	if err != nil {
		m.Error("查询app信息错误", zap.Error(err))
		c.ResponseError(errors.New("查询app信息错误"))
		return
	}
	if app == nil {
		c.ResponseOK()
		return
	}
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.Error("开启事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事务失败！"))
		return
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()
	err = m.db.deleteAppTx(appId, tx)
	if err != nil {
		tx.Rollback()
		m.Error("删除应用错误", zap.Error(err))
		c.ResponseError(errors.New("删除应用错误"))
		return
	}
	err = m.db.deleteCategoryAppTx(appId, tx)
	if err != nil {
		tx.Rollback()
		m.Error("删除分类下应用错误", zap.Error(err))
		c.ResponseError(errors.New("删除分类下应用错误"))
		return
	}
	err = m.db.deleteUserAppTx(appId, tx)
	if err != nil {
		tx.Rollback()
		m.Error("删除用户app错误", zap.Error(err))
		c.ResponseError(errors.New("删除用户app错误"))
		return
	}
	err = m.db.deleteUserRecordAppTx(appId, tx)
	if err != nil {
		tx.Rollback()
		m.Error("删除用户app使用记录错误", zap.Error(err))
		c.ResponseError(errors.New("删除用户app使用记录错误"))
		return
	}
	if err = tx.Commit(); err != nil {
		m.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		tx.Rollback()
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
	tx, err := m.ctx.DB().Begin()
	if err != nil {
		m.Error("开启事务失败！", zap.Error(err))
		c.ResponseError(errors.New("开启事务失败！"))
		return
	}
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
	app, err := m.db.queryAppWithAppNameAndCategoryNo(req.Name)
	if err != nil {
		m.Error("查询此分类下app是否存在此名称错误", zap.Error(err))
		c.ResponseError(errors.New("查询此分类下app是否存在此名称错误"))
		return
	}
	if app != nil && len(app.AppID) > 0 {
		c.ResponseError(errors.New("此应用名已存在"))
		return
	}

	err = m.db.insertAPP(&appModel{
		AppID:       util.GenerUUID(),
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
		AppCategory: req.AppCategory,
		Status:      1,
		JumpType:    req.JumpType,
		AppRoute:    req.AppRoute,
		WebRoute:    req.WebRoute,
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
	if strings.TrimSpace(req.AppRoute) == "" {
		return errors.New("应用打开地址不能为空")
	}
	if strings.TrimSpace(req.Icon) == "" {
		return errors.New("应用logo不能为空")
	}
	return nil
}

type updateAppReq struct {
	Status int `json:"status"` // 1.可用 0.禁用
	appReq
}
type appReq struct {
	Icon        string `json:"icon"`         // 应用icon
	Name        string `json:"name"`         // 应用名称
	Description string `json:"description"`  // 应用介绍
	AppCategory string `json:"app_category"` // 应用分类 [‘机器人’ ‘客服’]
	JumpType    int    `json:"jump_type"`    // 打开方式 0.网页 1.原生
	AppRoute    string `json:"app_route"`    // app打开地址
	WebRoute    string `json:"web_route"`    // web打开地址
	IsPaidApp   int    `json:"is_paid_app"`  // 是否为付费应用 0.否 1.是
}
type bannerReq struct {
	Cover       string `json:"cover"`       // 封面
	Title       string `json:"title"`       // 标题
	Description string `json:"description"` // 介绍
	JumpType    int    `json:"jump_type"`   // 打开方式 0.网页 1.原生
	Route       string `json:"route"`       // 打开地址
}
