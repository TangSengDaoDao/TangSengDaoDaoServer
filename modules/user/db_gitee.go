package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type giteeDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newGiteeDB(ctx *config.Context) *giteeDB {

	return &giteeDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *giteeDB) insert(m *gitUserInfoModel) error {
	_, err := d.session.InsertInto("gitee_user").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *giteeDB) insertTx(m *gitUserInfoModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("gitee_user").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}
func (d *giteeDB) queryWithLogin(login string) (*gitUserInfoModel, error) {
	var m *gitUserInfoModel
	_, err := d.session.Select("*").From("gitee_user").Where("login=?", login).Load(&m)
	return m, err
}

type gitUserInfoModel struct {
	Id                int64
	CreatedAt         db.Time
	UpdatedAt         db.Time
	Login             string
	Name              string
	Email             string
	Bio               string
	AvatarURL         string
	Blog              string
	EventsURL         string
	Followers         int
	FollowersURL      string
	Following         int
	FollowingURL      string
	GistsURL          string
	HtmlURL           string
	MemberRole        string
	OrganizationsURL  string
	PublicGists       int
	PublicRepos       int
	ReceivedEventsURL string
	Remark            string
	ReposURL          string
	Stared            int
	StarredURL        string
	SubscriptionsURL  string
	URL               string
	Watched           int
	Weibo             string
	Type              string
	GiteeCreatedAt    string
	GiteeUpdatedAt    string
}
