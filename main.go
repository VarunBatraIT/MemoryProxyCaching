package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

	"github.com/OneOfOne/xxhash"
	"github.com/caddyserver/certmagic"
	"github.com/gin-gonic/gin"
	"github.com/parnurzeal/gorequest"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
	"github.com/tdewolff/minify/v2/json"
	"github.com/tdewolff/minify/v2/svg"
	"github.com/tdewolff/minify/v2/xml"
)

type CachedResponse struct {
	Url         string
	Body        string
	ContentType string
	ETag        string
	StatusCode  int
}

var appConfig ProxyCacheConfig

func main() {

	initConfig()
	initCache()
	if !appConfig.Debug {
		log.Println("Setting Release Mode")
		gin.SetMode(gin.ReleaseMode)
		log.SetOutput(ioutil.Discard)
	}
	app := initServer()
	certmagic.DefaultACME.Email = appConfig.AcmeConfig.Email
	certmagic.DefaultACME.Agreed = true
	if appConfig.AcmeConfig.Fake {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}
	serveSSL(app)
	forever := make(chan bool)
	<-forever
}
func serveSSL(app *gin.Engine) {
	fmt.Println(appConfig)

	if len(appConfig.AcmeConfig.Domains) == 0 {
		log.Fatal("No domains are specified for SSL")
	}
	err := certmagic.HTTPS(appConfig.AcmeConfig.Domains, app)
	if err != nil {
		log.Fatal(err)
	}
}
func request(domainConfig DomainConfiguration, url string) (gorequest.Response, string, []error) {
	return gorequest.New().Get(url).Set("User-Agent", domainConfig.UserAgent).End()
}

func setEtag(cachable CachedResponse) CachedResponse {
	output := []byte(cachable.Body)
	// quicker hash
	hash := xxhash.New32()
	_, err := hash.Write([]byte(output))
	if err == nil {
		eTag := fmt.Sprintf("%x", hash.Sum(nil))
		eTag = "W/\"" + eTag + "\""
		cachable.ETag = eTag
	}
	return cachable
}

func minifyResponse(domainConfig DomainConfiguration, body string, cachable CachedResponse) string {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("image/svg+xml", svg.Minify)
	m.AddFuncRegexp(regexp.MustCompile("^(application|text)/(x-)?(java|ecma)script$"), js.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]json$"), json.Minify)
	m.AddFuncRegexp(regexp.MustCompile("[/+]xml$"), xml.Minify)
	m.Add("text/html", &html.Minifier{
		KeepDefaultAttrVals: true,
		KeepWhitespace:      true,
	})
	body, err := m.String(cachable.ContentType, body)
	if err != nil {
		panic(err)
	}
	return body
}
func cachedOut(domainConfig DomainConfiguration, cachable CachedResponse, ctx *gin.Context) {
	cachable = setEtag(cachable)
	ctx.Header("ETAG", cachable.ETag)
	ctx.Header("Content-Type", cachable.ContentType)
	ctx.Header("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable, stale-if-error=%d", domainConfig.ExpiresIn, domainConfig.ExpiresIn))
	none := ctx.GetHeader("If-None-Match")
	if none == cachable.ETag {
		ctx.AbortWithStatus(http.StatusNotModified)
		return
	}
	body := cachable.Body
	output := []byte(body)
	contentType := cachable.ContentType
	ctx.Data(http.StatusOK, contentType, output)
}
