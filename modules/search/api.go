package search

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/group"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
	"github.com/TangSengDaoDao/TangSengDaoDaoServer/pkg/log"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/wkhttp"
)

type Search struct {
	ctx *config.Context
	log.Log
	userService  user.IService
	groupService group.IService
}

func New(ctx *config.Context) *Search {
	s := &Search{
		ctx:          ctx,
		Log:          log.NewTLog("search"),
		userService:  user.NewService(ctx),
		groupService: group.NewService(ctx),
	}
	return s
}

func (s *Search) Route(r *wkhttp.WKHttp) {
	searchs := r.Group("/v1/search", s.ctx.AuthMiddleware(r))
	{
		searchs.GET("/gobal", s.gobal)        // 全局搜索
		searchs.POST("/message", s.search)    // 搜索消息
		searchs.GET("/channel", s.getChannel) // 获取channel
		searchs.GET("/sender", s.getFrom)     // 获取发送者
	}
}

func (s *Search) gobal(c *wkhttp.Context) {

}

func (s *Search) search(c *wkhttp.Context) {

}

func (s *Search) getChannel(c *wkhttp.Context) {

}

func (s *Search) getFrom(c *wkhttp.Context) {

}
