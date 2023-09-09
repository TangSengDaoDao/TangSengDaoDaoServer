package workplace

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	dba "github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type managerDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newManagerDB(ctx *config.Context) *managerDB {
	return &managerDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *managerDB) queryCateogryWithName(name string) (*categoryModel, error) {
	var m *categoryModel
	_, err := d.session.Select("*").From("workplace_category").Where("name=?", name).Load(&m)
	return m, err
}

func (d *managerDB) insertCategory(m *categoryModel) error {
	_, err := d.session.InsertInto("workplace_category").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *managerDB) queryMaxSortNumCategory() (*categoryModel, error) {
	var m *categoryModel
	_, err := d.session.Select("*").From("workplace_category").OrderDir("sort_num", false).Limit(1).Load(&m)
	return m, err
}

func (d *managerDB) queryAppWithAppNameAndCategoryNo(appName string) (*appModel, error) {
	var m *appModel
	_, err := d.session.Select("*").From("workplace_app").Where("name=?", appName).Load(&m)
	return m, err
}

func (d *managerDB) insertAPP(app *appModel) error {
	_, err := d.session.InsertInto("workplace_app").Columns(util.AttrToUnderscore(app)...).Record(app).Exec()
	return err
}

func (d *managerDB) insertBanner(banner *bannerModel) error {
	_, err := d.session.InsertInto("workplace_banner").Columns(util.AttrToUnderscore(banner)...).Record(banner).Exec()
	return err
}

func (d *managerDB) updateApp(app *appModel) error {
	_, err := d.session.Update("workplace_app").SetMap(map[string]interface{}{
		"app_category": app.AppCategory,
		"icon":         app.Icon,
		"name":         app.Name,
		"description":  app.Description,
		"status":       app.Status,
		"jump_type":    app.JumpType,
		"app_route":    app.AppRoute,
		"web_route":    app.WebRoute,
		"is_paid_app":  app.IsPaidApp,
	}).Where("app_id=?", app.AppID).Exec()
	return err
}
func (d *managerDB) updateBanner(banner *bannerModel) error {
	_, err := d.session.Update("workplace_banner").SetMap(map[string]interface{}{
		"cover":       banner.Cover,
		"title":       banner.Title,
		"description": banner.Description,
		"jump_type":   banner.JumpType,
		"route":       banner.Route,
	}).Where("banner_no=?", banner.BannerNo).Exec()
	return err
}

func (d *managerDB) updateCategorySortNumWithTx(categoryNo string, sortNum int, tx *dbr.Tx) error {
	_, err := tx.Update("workplace_category").SetMap(map[string]interface{}{
		"sort_num": sortNum,
	}).Where("category_no=?", categoryNo).Exec()
	return err
}

func (d *managerDB) deleteApp(appId string) error {
	_, err := d.session.DeleteFrom("workplace_app").Where("app_id=?", appId).Exec()
	return err
}

func (d *managerDB) queryCategory() ([]*categoryModel, error) {
	var models []*categoryModel
	_, err := d.session.Select("*").From("workplace_category").OrderDir("sort_num", false).Load(&models)
	return models, err
}

func (d *managerDB) deleteBanner(bannerNo string) error {
	_, err := d.session.DeleteFrom("workplace_banner").Where("banner_no=?", bannerNo).Exec()
	return err
}
func (d *managerDB) updateCategoryAppSortNumWithTx(categoryNo string, appId string, sortNum int, tx *dbr.Tx) error {
	_, err := tx.Update("workplace_category_app").SetMap(map[string]interface{}{
		"sort_num": sortNum,
	}).Where("category_no=? and app_id=?", categoryNo, appId).Exec()
	return err
}
func (d *managerDB) insertCategoryAppWithTx(m *categoryAppModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("workplace_category_app").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *managerDB) insertCategoryApp(m *categoryAppModel) error {
	_, err := d.session.InsertInto("workplace_category_app").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *managerDB) deleteCategoryApp(appId, categoryNo string) error {
	_, err := d.session.DeleteFrom("workplace_category_app").Where("app_id=? and category_no=?", appId, categoryNo).Exec()
	return err
}

func (d *managerDB) queryAllApp() ([]*appModel, error) {
	var models []*appModel
	_, err := d.session.Select("*").From("workplace_app").OrderDir("created_at", false).Load(&models)
	return models, err
}
func (d *managerDB) queryMaxSortNumCategoryApp(categoryNo string) (*categoryAppModel, error) {
	var m *categoryAppModel
	_, err := d.session.Select("*").From("workplace_category_app").Where("category_no=?", categoryNo).OrderDir("sort_num", false).Limit(1).Load(&m)
	return m, err
}

type categoryAppModel struct {
	CategoryNo string
	AppId      string
	SortNum    int // 排序号
	dba.BaseModel
}
