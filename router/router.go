package router

import (
	"github.com/gin-gonic/gin"
	"github.com/open-kingfisher/king-preset/impl"
	"github.com/open-kingfisher/king-utils/common"
	"net/http"
)

func SetupRouter(r *gin.Engine) *gin.Engine {

	//重新定义404
	r.NoRoute(NoRoute)
	// Pod IP 地址固定
	r.POST(common.PresetPath+"mutate/fixpodip", impl.MutateFixPodIP)
	r.POST(common.PresetPath+"validate/fixpodip", impl.ValidateFixPodIP)
	// EndPoint Extend
	r.POST(common.PresetPath+"mutate/endpointextendip", impl.MutateEndpointExtendIp)
	r.POST(common.PresetPath+"validate/endpointextendip", impl.ValidateEndpointExtendIp)

	return r
}

// 重新定义404错误
func NoRoute(c *gin.Context) {
	responseData := common.ResponseData{Code: http.StatusNotFound, Msg: "404 Not Found"}
	c.JSON(http.StatusNotFound, responseData)
}
