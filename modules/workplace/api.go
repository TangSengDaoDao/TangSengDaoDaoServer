package workplace

import (
	"errors"

	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/common"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"go.uber.org/zap"
)

type Workplace struct {
	ctx *config.Context
	log.Log
	db *db
}

func New(ctx *config.Context) *Workplace {
	return &Workplace{
		ctx: ctx,
		Log: log.NewTLog("Workplace"),
		db:  newDB(ctx),
	}
}

// Route 路由配置
func (w *Workplace) Route(r *wkhttp.WKHttp) {
	auth := r.Group("/v1/workplace", w.ctx.AuthMiddleware(r))
	{
		auth.GET("/banner", w.getBanner)                // 获取横幅
		auth.GET("/user/app", w.getApps)                // 获取用户添加的app
		auth.POST("/user/app", w.addApp)                // 添加app
		auth.DELETE("/user/app", w.deleteApp)           // 删除app
		auth.PUT("/user/app/reorder", w.reorderApp)     // 排序app
		auth.POST("/user/app/record", w.addRecord)      // 添加使用记录
		auth.GET("/user/app/record", w.getRecord)       // 查询常用app
		auth.DELETE("/user/app/record", w.deleteRecord) // 删除使用记录
		auth.GET("/category", w.getCategory)            // 获取分类
		auth.GET("/category/app", w.getAppWithCategory) // 获取某个分类下的应用
	}
}

func (w *Workplace) deleteRecord(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	appId := c.Query("app_id")
	if appId == "" {
		c.ResponseError(errors.New("删除的应用ID不能为空"))
		return
	}
	err := w.db.deleteRecord(loginUID, appId)
	if err != nil {
		w.Error("删除使用记录错误", zap.Error(err))
		c.ResponseError(errors.New("删除使用记录错误"))
		return
	}
	c.ResponseOK()
}

func (w *Workplace) getRecord(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	records, err := w.db.queryRecordWithUid(loginUID)
	if err != nil {
		w.Error("查询使用记录错误", zap.Error(err))
		c.ResponseError(errors.New("查询使用记录错误"))
		return
	}
	list := make([]*appResp, 0)
	if len(records) > 0 {
		appIds := make([]string, 0)
		for _, record := range records {
			appIds = append(appIds, record.AppId)
		}
		apps, err := w.db.queryAppWithAppIds(appIds)
		if err != nil {
			w.Error("查询一批应用错误", zap.Error(err))
			c.ResponseError(errors.New("查询一批应用错误"))
			return
		}
		if len(apps) > 0 {
			// 查询保存的app
			saveList, err := w.db.queryAppWithUid(loginUID)
			if err != nil {
				w.Error("查询用户已保存的应用错误", zap.Error(err))
				c.ResponseError(errors.New("查询用户已保存的应用错误"))
				return
			}
			for _, app := range apps {
				isAdded := 0
				if len(saveList) > 0 {
					for _, tempApp := range saveList {
						if tempApp.AppID == app.AppID {
							isAdded = 1
							break
						}
					}
				}
				appResp := app.getAppResp(isAdded)
				list = append(list, appResp)
			}
		}
	}
	c.Response(list)
}
func (w *Workplace) addRecord(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	type reqVO struct {
		AppId string `json:"app_id"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		w.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if req.AppId == "" {
		c.ResponseError(errors.New("应用ID不能为空"))
		return
	}
	app, err := w.db.queryAppWithAppId(req.AppId)
	if err != nil {
		w.Error("查询应用错误", zap.Error(err))
		c.ResponseError(errors.New("查询应用错误"))
		return
	}
	if app == nil || app.Status == 0 {
		c.ResponseError(errors.New("该应用已删除或不可用"))
		return
	}
	record, err := w.db.queryRecordWithUidAndAppId(loginUID, req.AppId)
	if err != nil {
		w.Error("查询使用记录错误", zap.Error(err))
		c.ResponseError(errors.New("查询使用记录错误"))
		return
	}
	if record == nil {
		err := w.db.insertRecord(&recordModel{
			AppId: req.AppId,
			Count: 1,
			Uid:   loginUID,
		})
		if err != nil {
			w.Error("新增使用记录错误", zap.Error(err))
			c.ResponseError(errors.New("新增使用记录错误"))
			return
		}
	} else {
		record.Count = record.Count + 1
		err := w.db.updateRecordCount(record)
		if err != nil {
			w.Error("修改使用记录错误", zap.Error(err))
			c.ResponseError(errors.New("修改使用记录错误"))
			return
		}
	}
	c.ResponseOK()
}
func (w *Workplace) getAppWithCategory(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	categoryNo := c.Query("category_no")
	if categoryNo == "" {
		c.ResponseError(errors.New("分类编号不能为空"))
		return
	}
	apps, err := w.db.queryAppWithCategroyNo(categoryNo)
	if err != nil {
		w.Error("通过分类查询应用错误", zap.Error(err))
		c.ResponseError(errors.New("通过分类查询应用错误"))
		return
	}
	list := make([]*appResp, 0)
	if len(apps) > 0 {
		// 查询保存的app
		saveList, err := w.db.queryAppWithUid(loginUID)
		if err != nil {
			w.Error("查询用户已保存的应用错误", zap.Error(err))
			c.ResponseError(errors.New("查询用户已保存的应用错误"))
			return
		}
		for _, app := range apps {
			isAdded := 0
			if len(saveList) > 0 {
				for _, tempApp := range saveList {
					if tempApp.AppID == app.AppID {
						isAdded = 1
						break
					}
				}
			}
			appResp := app.getAppResp(isAdded)
			appResp.SortNum = app.SortNum
			list = append(list, appResp)
		}
	}
	c.Response(list)
}

// 排序用户app
func (w *Workplace) reorderApp(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	type reqVO struct {
		AppIds []string `json:"app_ids"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		w.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	tx, _ := w.ctx.DB().Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	var tempSortNum = len(req.AppIds)
	for _, appId := range req.AppIds {
		err := w.db.updateUserAppSortNumWithTx(loginUID, appId, tempSortNum, tx)
		if err != nil {
			tx.Rollback()
			w.Error("修改用户app顺序错误", zap.Error(err))
			c.ResponseError(errors.New("修改用户app顺序错误"))
			return
		}
		tempSortNum--
	}
	err := tx.Commit()
	if err != nil {
		w.Error("数据库事物提交失败", zap.Error(err))
		c.ResponseError(errors.New("数据库事物提交失败"))
		tx.Rollback()
		return
	}
	c.ResponseOK()
}

// 移除一个app
func (w *Workplace) deleteApp(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	appId := c.Query("app_id")
	if appId == "" {
		c.ResponseError(errors.New("appId不能为空"))
		return
	}
	err := w.db.deleteUserAppWithAppId(loginUID, appId)
	if err != nil {
		c.ResponseError(errors.New("删除用户app错误"))
		w.Error("删除用户app错误", zap.Error(err))
		return
	}
	c.ResponseOK()
}

// 添加app
func (w *Workplace) addApp(c *wkhttp.Context) {
	loginUID := c.GetLoginUID()
	type reqVO struct {
		AppId string `json:"app_id"`
	}
	var req reqVO
	if err := c.BindJSON(&req); err != nil {
		w.Error(common.ErrData.Error(), zap.Error(err))
		c.ResponseError(common.ErrData)
		return
	}
	if req.AppId == "" {
		c.ResponseError(errors.New("应用ID不能为空"))
		return
	}
	app, err := w.db.queryAppWithAppId(req.AppId)
	if err != nil {
		c.ResponseError(errors.New("查询应用错误"))
		w.Error("查询应用错误", zap.Error(err))
		return
	}
	if app == nil || len(app.AppID) == 0 || app.Status == 0 {
		w.Error("该应用不存在或被禁用", zap.Error(err))
		return
	}
	userApp, err := w.db.queryUserAppWithAPPId(loginUID, req.AppId)
	if err != nil {
		c.ResponseError(errors.New("查询用户某个应用错误"))
		w.Error("查询用户某个应用错误", zap.Error(err))
		return
	}
	if userApp != nil && len(userApp.AppID) > 0 {
		c.ResponseOK()
		return
	}
	maxSortNumApp, err := w.db.queryUserAppMaxSortNumWithUID(loginUID)
	if err != nil {
		c.ResponseError(errors.New("查询用户最大序号app错误"))
		w.Error("查询用户最大序号app错误", zap.Error(err))
		return
	}
	sortNum := 1
	if maxSortNumApp != nil && len(maxSortNumApp.AppID) > 0 {
		sortNum = maxSortNumApp.SortNum + 1
	}
	err = w.db.insertUserApp(&userAppModel{
		Uid:     loginUID,
		SortNum: sortNum,
		AppID:   req.AppId,
	})
	if err != nil {
		c.ResponseError(errors.New("添加用户app错误"))
		w.Error("添加用户app错误", zap.Error(err))
		return
	}
	c.ResponseOK()
}

// 获取分类
func (w *Workplace) getCategory(c *wkhttp.Context) {
	models, err := w.db.queryCategory()
	if err != nil {
		c.ResponseError(errors.New("查询分类错误"))
		w.Error("查询分类错误", zap.Error(err))
		return
	}
	list := make([]*categoryResp, 0)
	if len(models) > 0 {
		for _, m := range models {
			list = append(list, &categoryResp{
				CategoryNo: m.CategoryNo,
				SortNum:    m.SortNum,
				Name:       m.Name,
			})
		}
	}
	c.Response(list)
}

// 获取用户app
func (w *Workplace) getApps(c *wkhttp.Context) {
	models, err := w.db.queryUserApp(c.GetLoginUID())
	if err != nil {
		c.ResponseError(errors.New("查询用户应用错误"))
		w.Error("查询用户应用错误", zap.Error(err))
		return
	}
	list := make([]*appDetailResp, 0)
	appIds := make([]string, 0)
	if len(models) > 0 {
		for _, m := range models {
			appIds = append(appIds, m.AppID)
		}
		apps, err := w.db.queryAppWithAppIds(appIds)
		if err != nil {
			c.ResponseError(errors.New("通过应用ID查询应用错误"))
			w.Error("通过应用ID查询应用错误", zap.Error(err))
			return
		}
		for _, m := range models {
			var app *appModel
			for _, appModel := range apps {
				if appModel.AppID == m.AppID {
					app = appModel
				}
			}
			if app != nil {
				list = append(list, &appDetailResp{
					AppID:       m.AppID,
					SortNum:     m.SortNum,
					Icon:        app.Icon,
					Name:        app.Name,
					Description: app.Description,
					AppCategory: app.AppCategory,
					Status:      app.Status,
					JumpType:    app.JumpType,
					AppRoute:    app.AppRoute,
					WebRoute:    app.WebRoute,
					IsPaidApp:   app.IsPaidApp,
				})
			}
		}
	}

	c.Response(list)

}

// 获取横幅
func (w *Workplace) getBanner(c *wkhttp.Context) {
	models, err := w.db.queryBanner()
	if err != nil {
		c.ResponseError(errors.New("查询横幅数据错误"))
		w.Error("查询横幅数据错误", zap.Error(err))
		return
	}
	list := make([]*bannerResp, 0)
	if len(models) > 0 {
		for _, m := range models {
			list = append(list, &bannerResp{
				BannerNo:    m.BannerNo,
				Cover:       m.Cover,
				Title:       m.Title,
				Description: m.Description,
				JumpType:    m.JumpType,
				Route:       m.Route,
			})
		}
	}
	c.Response(list)
}

func (app *appModel) getAppResp(isAdded int) *appResp {
	var appResp = &appResp{}
	appResp.IsAdded = isAdded
	appResp.AppID = app.AppID
	appResp.AppCategory = app.AppCategory
	appResp.Icon = app.Icon
	appResp.Name = app.Name
	appResp.Description = app.Description
	appResp.Status = app.Status
	appResp.JumpType = app.JumpType
	appResp.AppRoute = app.AppRoute
	appResp.WebRoute = app.WebRoute
	appResp.IsPaidApp = app.IsPaidApp
	return appResp
}

type bannerResp struct {
	BannerNo    string `json:"banner_no"`   // 横幅编号
	Cover       string `json:"cover"`       // 封面
	Title       string `json:"title"`       // 标题
	Description string `json:"description"` // 介绍
	JumpType    int    `json:"jump_type"`   // 打开方式 0.网页 1.原生
	Route       string `json:"route"`       // 打开地址
	CreatedAt   string `json:"created_at"`  // 创建时间
}

type appResp struct {
	IsAdded int `json:"is_added"` // 1.已经添加 0.未添加
	appDetailResp
}

type appDetailResp struct {
	AppID       string `json:"app_id"`       // 分类项唯一id
	SortNum     int    `json:"sort_num"`     // 排序编号
	Icon        string `json:"icon"`         // 应用icon
	Name        string `json:"name"`         // 应用名称
	Description string `json:"description"`  // 应用介绍
	AppCategory string `json:"app_category"` // 应用分类 [‘机器人’ ‘客服’]
	Status      int    `json:"status"`       // 是否可用 0.禁用 1.可用
	JumpType    int    `json:"jump_type"`    // 打开方式 0.网页 1.原生
	AppRoute    string `json:"app_route"`    // app打开地址
	WebRoute    string `json:"web_route"`    // web打开地址
	IsPaidApp   int    `json:"is_paid_app"`  // 是否为付费应用 0.否 1.是
}
type categoryResp struct {
	CategoryNo string `json:"category_no"` //  分类编号
	Name       string `json:"name"`        // 分类名称
	SortNum    int    `json:"sort_num"`    //  排序编号
}
