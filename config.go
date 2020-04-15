package main

import (
	"fmt"

	"github.com/spf13/viper"
)

var config = viper.New()

type ProxyCacheConfig struct {
	// Logs for Debug as well as GIN
	Debug bool `mapstructure:"debug"`
	// Garbage collection percentage to keep check with memory usage
	GarbageCollectionPercentage int `mapstructure:"garbage_collection_percentage"`
	// Information regarding SSL certificates that are supposed to be created
	AcmeConfig AcmeConfig `mapstructure:"acme_config"`
	// Configuration per proxy
	DomainConfig map[string]DomainConfiguration `mapstructure:"domain_config"`
}

type AcmeConfig struct {
	// Fake certificate can be requested from ACME
	Fake bool `mapstructure:"fake"`
	// Email address to be used for Certificate
	Email string `mapstructure:"email"`
	// Only domains for which certificates will be generated
	Domains []string `mapstructure:"domains"`
}

type DomainConfiguration struct {
	// Re-check time - the time response will be kep in memory.
	CacheResponseTime string `mapstructure:"cache_response_time"`
	// The URL which will act as a proxy to (Backend)
	ProxyTo string `mapstructure:"proxy_to"`
	// If you want to minify the response from the backend.
	Minify bool `mapstructure:"minify"`
	// The useragent you want to use to hit the backend.
	UserAgent string `mapstructure:"user_agent"`
	// Expires in header
	ExpiresIn int `mapstructure:"expires_in"`
	// Password to delete cache pre-time
	Password string `mapstructure:"password"`
}

func initConfig() {
	config.AddConfigPath("./")
	config.SetConfigName("config")
	config.SetConfigType("json")
	err := config.ReadInConfig() // Find and read the config file
	if err != nil {              // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
	err = config.Unmarshal(&appConfig)
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

}
