package base

import (
	"embed"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
)

//go:embed sql
var sqlFS embed.FS

func init() {

	register.AddModule(func(ctx interface{}) register.Module {

		return register.Module{
			SQLDir: register.NewSQLFS(sqlFS),
		}
	})
}
