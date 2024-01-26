package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/sync/errgroup"

	"github.com/veops/messenger/docs"
	"github.com/veops/messenger/global"
	"github.com/veops/messenger/middleware"
	"github.com/veops/messenger/send"
)

// main
//
//	@externalDocs.description	Messenger README
//	@externalDocs.url			https://github.com/veops/messenger?tab=readme-ov-file#messenger
func main() {
	authConf, err := global.GetAuthConf()
	if err != nil {
		log.Fatalln(err)
	}
	appConf, err := global.GetAppConf()
	if err != nil {
		log.Fatalln(err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	g1 := r.Group("/v1").Use(middleware.Auth(authConf), middleware.Error2Resp())
	{
		g1.POST("/message", send.PushMessage)
		g1.POST("/uid/getbyphone", send.GetUIDByPhone)

		g1.POST("/senders", global.PushRemoteConf)
		g1.PUT("/senders", global.PushRemoteConf)
		g1.DELETE("/senders", global.PushRemoteConf)
	}
	g2 := r.Group("/v1").Use(middleware.Error2Resp())
	{
		g2.GET("/histories", send.QueryHistory)
	}

	r.StaticFile("/web", "./web/build/index.html")
	r.StaticFile("/manifest.json", "./web/build/manifest.json")
	r.StaticFile("/logo192.png", "./web/build/logo192.png")
	r.StaticFile("/favicon.ico", "./web/build//favicon.ico")
	r.Static("/static", "./web/build/static")
	// r.Static("/manifest.json", ".web/build/manifest.json")

	docs.SwaggerInfo.Title = "Messenger api"
	docs.SwaggerInfo.Version = ""
	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	eg := &errgroup.Group{}
	eg.Go(send.Start)
	eg.Go(func() error {
		return r.Run(fmt.Sprintf("%s:%s", appConf["ip"], appConf["port"]))
	})
	log.Println("start successfully...")
	if err := eg.Wait(); err != nil {
		log.Println(err)
	}
}
