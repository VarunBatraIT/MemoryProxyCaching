package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/adelowo/onecache"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func initServer() *gin.Engine {
	app := gin.Default()
	app.Use(gzip.Gzip(gzip.DefaultCompression))
	app.Use(func(c *gin.Context) {
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubdomains; preload")
	})
	app = Handlers(app)
	return app

}

func GetURLToHit(ctx *gin.Context, domainConfig DomainConfiguration) string {
	uri := ctx.Param("Uri")
	rawQuery := ctx.Request.URL.RawQuery
	url := fmt.Sprintf("https://%s%s", domainConfig.ProxyTo, uri)
	if len(rawQuery) > 0 {
		url = fmt.Sprintf("https://%s%s?%s", domainConfig.ProxyTo, uri, rawQuery)
	}
	return url
}

func Handlers(app *gin.Engine) *gin.Engine {
	app.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, map[string]interface{}{
			"success": "ok",
		})
	})
	app.DELETE(":Domain/*Uri", func(ctx *gin.Context) {
		password, exists := ctx.GetPostForm("password")
		if !exists {
			ctx.AbortWithStatus(http.StatusNotAcceptable)
			return
		}
		domain := ctx.Param("Domain")
		domainConfig, exists := appConfig.DomainConfig[domain]
		if !exists {
			log.Println("Domain is not whitelisted")
			ctx.AbortWithStatus(http.StatusNotAcceptable)
			return
		}

		if password != domainConfig.Password {
			log.Println("Pasword not mentioned")
			ctx.AbortWithStatus(http.StatusUnauthorized)
			return

		}
		url := GetURLToHit(ctx, domainConfig)
		err := cache.Delete(url)
		if err != nil {
			log.Println("Deleted Cache ", url)
			ctx.JSON(200, `{ "success": true}`)
		} else {
			log.Println(err)
			ctx.JSON(http.StatusExpectationFailed, `{"success":false}`)
		}
	})
	app.GET(":Domain/*Uri", func(ctx *gin.Context) {
		domain := ctx.Param("Domain")
		domainConfig, exists := appConfig.DomainConfig[domain]
		log.Println(domain)
		log.Println(appConfig)
		if !exists {
			log.Println("Domain is not whitelisted")
			ctx.AbortWithStatus(404)
			return
		}
		url := GetURLToHit(ctx, domainConfig)
		err := serveFromCache(domainConfig, url, ctx)
		if err == nil {
			return
		}
		log.Println(err)
		resp, body, errs := request(domainConfig, url)
		if len(errs) > 0 || (resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotModified) {
			log.Println(errs)
			ctx.AbortWithStatus(503)
			return
		}
		cachable := CachedResponse{Url: url, Body: body, ContentType: resp.Header.Get("Content-Type"), StatusCode: resp.StatusCode}
		cachable.Body = minifyResponse(domainConfig, cachable.Body, cachable)
		cachedOut(domainConfig, cachable, ctx)
		log.Println(fmt.Sprintf("Caching %s", cachable.Url))
		marshal := onecache.NewCacheSerializer()
		toCache, err := marshal.Serialize(cachable)
		if err != nil {
			log.Println(err)
			return
		}
		i, err := time.ParseDuration(domainConfig.CacheResponseTime)
		if err != nil {
			i = 60 * time.Second
		}
		cache.Set(cachable.Url, toCache, i)
	})
	return app

}
