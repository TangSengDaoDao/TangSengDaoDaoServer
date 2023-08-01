package report

import (
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

// 查询举报列表
func (m *managerDB) list(pageSize, page uint64, channelType int) ([]*managerReportModel, error) {
	var list []*managerReportModel
	_, err := m.session.Select("report.*,report_category.category_name").From("report").LeftJoin("report_category", "report.category_no=report_category.category_no").Where("report.channel_type=?", channelType).Offset((page-1)*pageSize).Limit(pageSize).OrderDir("report.created_at", false).Load(&list)
	return list, err
}

// 查询总用户
func (m *managerDB) queryReportCount(channelType int) (int64, error) {
	var count int64
	_, err := m.session.Select("count(*)").From("report").Where("channel_type=?", channelType).Load(&count)
	return count, err
}

type managerReportModel struct {
	UID          string
	CategoryNo   string
	ChannelID    string
	ChannelType  uint8
	Imgs         string
	Remark       string
	CategoryName string
	dba.BaseModel
}
