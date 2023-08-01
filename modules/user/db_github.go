package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/gocraft/dbr/v2"
)

type githubDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newGithubDB(ctx *config.Context) *githubDB {

	return &githubDB{
		ctx:     ctx,
		session: ctx.DB(),
	}
}

func (d *githubDB) insert(m *githubUserInfoModel) error {
	_, err := d.session.InsertInto("github_user").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}

func (d *githubDB) insertTx(m *githubUserInfoModel, tx *dbr.Tx) error {
	_, err := tx.InsertInto("github_user").Columns(util.AttrToUnderscore(m)...).Record(m).Exec()
	return err
}
func (d *githubDB) queryWithLogin(login string) (*githubUserInfoModel, error) {
	var m *githubUserInfoModel
	_, err := d.session.Select("*").From("github_user").Where("login=?", login).Load(&m)
	return m, err
}

type githubUserInfoModel struct {
	ID                      int64
	CreatedAt               db.Time
	UpdatedAt               db.Time
	Login                   string
	NodeID                  string
	AvatarURL               string
	GravatarID              string
	URL                     string
	HtmlUrl                 string
	FollowersURL            string
	FollowingURL            string
	GistsURL                string
	StarredURL              string
	SubscriptionsURL        string
	OrganizationsURL        string
	ReposURL                string
	EventsURL               string
	ReceivedEventsURL       string
	Type                    string
	SiteAdmin               bool
	Name                    string
	Company                 string
	Blog                    string
	Location                string
	Email                   string
	Hireable                bool
	Bio                     string
	TwitterUsername         string
	PublicRepos             int
	PublicGists             int
	Followers               int
	Following               int
	GithubCreatedAt         string
	GithubUpdatedAt         string
	PrivateGists            int
	TotalPrivateRepos       int
	OwnedPrivateRepos       int
	DiskUsage               int
	Collaborators           int
	TwoFactorAuthentication bool
}
