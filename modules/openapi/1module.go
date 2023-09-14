package openapi

import (
	_ "embed"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/register"
)

//go:embed swagger/api.yaml
var swaggerContent string

func init() {
	register.AddModule(func(ctx interface{}) register.Module {
		x := ctx.(*config.Context)
		api := New(x)
		return register.Module{
			Name:    "openapi",
			Swagger: swaggerContent,
			SetupAPI: func() register.APIRouter {
				return api
			},
		}
	})
}
