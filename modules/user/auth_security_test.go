package user

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
	"github.com/gocraft/dbr/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func securityTestDB(t *testing.T) *DB {
	t.Helper()
	conn, err := dbr.Open("sqlite3", ":memory:", nil)
	require.NoError(t, err)
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { conn.Close() })
	_, err = conn.Exec(`CREATE TABLE user (uid TEXT PRIMARY KEY, username TEXT, name TEXT, zone TEXT, phone TEXT, password TEXT, role TEXT, status INTEGER, is_destroy INTEGER, auth_version TEXT NOT NULL DEFAULT '')`)
	require.NoError(t, err)
	return &DB{session: conn.NewSession(nil)}
}

func TestSecurityPublicRecoveryRejectsAdministrators(t *testing.T) {
	for _, role := range []string{"admin", "superAdmin"} {
		t.Run(role, func(t *testing.T) {
			db := securityTestDB(t)
			_, err := db.session.InsertBySql(`INSERT INTO user (uid,username,name,zone,phone,password,role,status,is_destroy) VALUES ('u','admin','Admin','0086','13000000002','original',?,1,0)`, role).Exec()
			require.NoError(t, err)
			// A nil context/SMS provider ensures rejection precedes reading a fixed code or sending SMS.
			u := &User{db: db}
			r := wkhttp.New()
			r.POST("/reset", u.pwdforget)
			req := httptest.NewRequest("POST", "/reset", strings.NewReader(`{"zone":"0086","phone":"13000000002","code":"123456","pwd":"attacker-password"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "管理账号不支持")
			var password string
			require.NoError(t, db.session.Select("password").From("user").Where("uid='u'").LoadOne(&password))
			require.Equal(t, "original", password)
		})
	}
	require.NoError(t, checkPublicPasswordRecovery(&Model{Status: 1}))
	require.Error(t, checkPublicPasswordRecovery(&Model{Status: 0}))
	require.Error(t, checkPublicPasswordRecovery(&Model{Status: 1, IsDestroy: 1}))
}

func TestSecurityDisabledManagerCannotLogin(t *testing.T) {
	for _, state := range []struct{ status, destroyed int }{{0, 0}, {1, 1}} {
		t.Run(fmt.Sprint(state), func(t *testing.T) {
			db := securityTestDB(t)
			_, err := db.session.InsertBySql(`INSERT INTO user (uid,username,name,password,role,status,is_destroy) VALUES ('u','admin','Admin',?,'superAdmin',?,?)`, util.MD5(util.MD5("correct-password")), state.status, state.destroyed).Exec()
			require.NoError(t, err)
			m := &Manager{db: &managerDB{session: db.session}, userDB: db}
			r := wkhttp.New()
			r.POST("/login", m.login)
			req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"username":"admin","password":"correct-password"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusBadRequest, w.Code)
			require.Contains(t, w.Body.String(), "禁用或注销")
		})
	}
}

func TestSecurityEveryPasswordWriteRotatesSessionVersion(t *testing.T) {
	db := securityTestDB(t)
	_, err := db.session.InsertBySql(`INSERT INTO user (uid,name,password,role,status,is_destroy) VALUES ('u','Admin','original','admin',1,0)`).Exec()
	require.NoError(t, err)
	raw, err := db.encodeLoginToken("u", "Admin", "admin", "original")
	require.NoError(t, err)
	first, err := wkhttp.ParseTokenCacheInfo(raw)
	require.NoError(t, err)
	require.NotEmpty(t, first.AuthVersion)
	for _, update := range []func() error{
		func() error { return db.UpdateUsersWithField("password", "changed", "u") },
		func() error { return db.updatePassword("original", "u") }, // Reusing an old password must not revive tokens.
		func() error { return db.updateUser(map[string]interface{}{"password": "changed"}, "u") },
	} {
		require.NoError(t, update())
		var version string
		require.NoError(t, db.session.Select("auth_version").From("user").Where("uid='u'").LoadOne(&version))
		require.NotEmpty(t, version)
		require.NotEqual(t, first.AuthVersion, version)
		first.AuthVersion = version
	}
	_, err = db.encodeLoginToken("u", "Admin", "admin", "original")
	require.Error(t, err, "a stale login snapshot must not issue a valid token")
}
