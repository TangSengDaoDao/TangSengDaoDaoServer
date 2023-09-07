package workplace

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	dba "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type db struct {
	session *dbr.Session
	ctx     *config.Context
}

func newDB(ctx *config.Context) *db {
	return &db{
		ctx:     ctx,
		session: ctx.DB(),
	}
}
func (d *db) updateUserAppSortNumWithTx(uid, appId string, sortNum int, tx *dbr.Tx) error {
	_, err := tx.Update("workplace_user_app").SetMap(map[string]interface{}{
		"sort_num": sortNum,
	}).Where("uid=? and app_id=?", uid, appId).Exec()
	return err
}

func (d *db) insertUserApp(app *userAppModel) error {
	_, err := d.session.InsertInto("workplace_user_app").Columns(util.AttrToUnderscore(app)...).Record(app).Exec()
	return err
}

func (d *db) queryUserAppMaxSortNumWithUID(uid string) (*userAppModel, error) {
	var m *userAppModel
	_, err := d.session.Select("*").From("workplace_user_app").Where("uid=?", uid).OrderDir("sort_num", false).Limit(1).Load(&m)
	return m, err
}

func (d *db) deleteUserAppWithAppId(uid, appId string) error {
	_, err := d.session.DeleteFrom("workplace_user_app").Where("app_id=? and uid=?", appId, uid).Exec()
	return err
}

func (d *db) queryCategory() ([]*categoryModel, error) {
	var models []*categoryModel
	_, err := d.session.Select("*").From("workplace_category").OrderDir("sort_num", false).Load(&models)
	return models, err
}

func (d *db) queryAppWithAppIds(ids []string) ([]*appModel, error) {
	var models []*appModel
	_, err := d.session.Select("*").From("workplace_app").Where("app_id in ?", ids).Load(&models)
	return models, err
}

func (d *db) queryAppWithUid(uid string) ([]*appModel, error) {
	var models []*appModel
	_, err := d.session.Select("*").From("workplace_user_app").Where("uid=?", uid).Load(&models)
	return models, err
}

func (d *db) queryAppWithAppId(appId string) (*appModel, error) {
	var app *appModel
	_, err := d.session.Select("*").From("workplace_app").Where("app_id=?", appId).Load(&app)
	return app, err
}

func (d *db) queryAppWithCategroyNo(categoryNo string) ([]*cAppModel, error) {
	var apps []*cAppModel
	_, err := d.session.Select("workplace_category_app.sort_num,workplace_app.app_id,workplace_app.icon,workplace_app.name,workplace_app.description,workplace_app.app_category,workplace_app.jump_type,workplace_app.status,workplace_app.app_route,workplace_app.web_route,workplace_app.is_paid_app,workplace_app.created_at").From("workplace_category_app").LeftJoin("workplace_app", "workplace_category_app.app_id=workplace_app.app_id").Where("workplace_category_app.category_no=?", categoryNo).OrderDir("workplace_category_app.sort_num", false).Load(&apps)
	return apps, err
}

func (d *db) queryUserAppWithAPPId(uid string, appId string) (*userAppModel, error) {
	var app *userAppModel
	_, err := d.session.Select("*").From("workplace_user_app").Where("uid=? and app_id=?", uid, appId).Load(&app)
	return app, err
}

func (d *db) queryUserApp(uid string) ([]*userAppModel, error) {
	var models []*userAppModel
	_, err := d.session.Select("*").From("workplace_user_app").Where("uid=?", uid).OrderDir("sort_num", false).Load(&models)
	return models, err
}

func (d *db) queryBanner() ([]*bannerModel, error) {
	var models []*bannerModel
	_, err := d.session.Select("*").From("workplace_banner").OrderDir("created_at", false).Load(&models)
	return models, err
}

func (d *db) insertRecord(record *recordModel) error {
	_, err := d.session.InsertInto("workplace_app_user_record").Columns(util.AttrToUnderscore(record)...).Record(record).Exec()
	return err
}

func (d *db) queryRecordWithUid(uid string) ([]*recordModel, error) {
	var models []*recordModel
	_, err := d.session.Select("*").From("workplace_app_user_record").Where("uid=?", uid).OrderDir("count", false).Load(&models)
	return models, err
}

func (d *db) queryRecordWithUidAndAppId(uid, appId string) (*recordModel, error) {
	var record *recordModel
	_, err := d.session.Select("*").From("workplace_app_user_record").Where("uid=? and app_id=?", uid, appId).Load(&record)
	return record, err
}

func (d *db) updateRecordCount(record *recordModel) error {
	_, err := d.session.Update("workplace_app_user_record").SetMap(map[string]interface{}{
		"count": record.Count,
	}).Where("uid=? and app_id=?", record.Uid, record.AppId).Exec()
	return err
}
func (d *db) deleteRecord(uid, appId string) error {
	_, err := d.session.DeleteFrom("workplace_app_user_record").Where("app_id=? and uid=?", appId, uid).Exec()
	return err
}

type recordModel struct {
	Count int // 使用次数
	Uid   string
	AppId string
	dba.BaseModel
}
type categoryModel struct {
	CategoryNo string //  分类编号
	Name       string // 分类名称
	SortNum    int    //  排序编号
	dba.BaseModel
}

type bannerModel struct {
	BannerNo    string // 封面编号
	Cover       string // 封面
	Title       string // 标题
	Description string // 介绍
	JumpType    int    // 打开方式 0.网页 1.原生
	Route       string // 打开地址
	dba.BaseModel
}
type userAppModel struct {
	AppID   string // 分类项唯一id
	SortNum int    // 排序编号
	Uid     string // 所属用户uid
	dba.BaseModel
}

type cAppModel struct {
	SortNum int // 排序编号
	appModel
}

type appModel struct {
	AppID       string // 应用ID
	Icon        string // 应用icon
	Name        string // 应用名称
	Description string // 应用介绍
	AppCategory string // 应用分类 [‘机器人’ ‘客服’]
	Status      int    // 是否可用 0.禁用 1.可用
	JumpType    int    // 打开方式 0.网页 1.原生
	AppRoute    string // app打开地址
	WebRoute    string // web打开地址
	IsPaidApp   int    // 是否为付费应用 0.否 1.是
	dba.BaseModel
}
