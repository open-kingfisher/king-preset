package main

import (
	"github.com/gin-gonic/gin"
	"github.com/open-kingfisher/king-preset/router"
	"github.com/open-kingfisher/king-utils/common/log"
	"github.com/open-kingfisher/king-utils/config"
	"github.com/open-kingfisher/king-utils/kit"
)

func main() {
	// Debug Mode
	gin.SetMode(config.Mode)
	g := gin.New()
	// 设置路由
	r := router.SetupRouter(kit.EnhanceGin(g))
	// Listen and Server in 0.0.0.0:443
	// tls.crt 和 tls.key 采用secret的方式挂载
	log.Info("Listen 443")
	if err := r.RunTLS(":443", "/etc/webhook/certs/tls.crt", "/etc/webhook/certs/tls.key"); err != nil {
		log.Fatalf("Listen error: %v", err)
	}
}
