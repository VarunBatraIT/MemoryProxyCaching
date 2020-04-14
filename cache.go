package main

import (
	"log"
	"runtime/debug"

	"github.com/adelowo/onecache"
	"github.com/adelowo/onecache/memory"
	"github.com/gin-gonic/gin"
)

var cache *memory.InMemoryStore

func initCache() {
	cache = memory.NewInMemoryStore()
	debug.SetGCPercent(50)
}
func serveFromCache(domainConfig DomainConfiguration, url string, ctx *gin.Context) error {
	res, err := cache.Get(url)
	if err == nil {
		marshal := onecache.NewCacheSerializer()
		log.Println("Serving from cache")
		cachable := CachedResponse{}
		err := marshal.DeSerialize(res, &cachable)
		if err != nil {
			return err
		}
		cachedOut(domainConfig, cachable, ctx)
		return nil
	}
	return err
}
