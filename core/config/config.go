package config

import (
	"errors"
	"strings"
	"sync"

	"github.com/apex/log"
	"github.com/fsnotify/fsnotify"
	"github.com/goastian/astiango-hub/core/interfaces"
	"github.com/goastian/astiango-hub/core/utils"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Name string
	interfaces.Logger
}

func (c *Config) Init() {
	// Load .env file before setting up viper
	c.loadDotEnv()

	// Set default values
	c.setDefaults()

	// config
	if c.Name != "" {
		viper.SetConfigFile(c.Name) // if config file is set, load it accordingly
	} else {
		viper.AddConfigPath("./conf") // if no config file is set, load by default
		viper.SetConfigName("config")
	}

	// config type as yaml
	viper.SetConfigType("yaml")

	// auto env
	viper.AutomaticEnv()

	// env prefix
	viper.SetEnvPrefix("ASTIANGO")

	// replacer
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	// read in config
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			if c.Logger != nil {
				c.Warn("No config file found. Using default values.")
			}
		}
	}

	// init log level
	c.initLogLevel()
}

func (c *Config) WatchConfig() {
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		if c.Logger != nil {
			c.Infof("Config file changed: %s", e.Name)
		}
	})
}

func (c *Config) setDefaults() {
	viper.SetDefault("mongo.host", "localhost")
	viper.SetDefault("mongo.port", 27017)
	viper.SetDefault("mongo.db", "astiango_hub_test")
	viper.SetDefault("mongo.username", "")
	viper.SetDefault("mongo.password", "")
	viper.SetDefault("mongo.authSource", "admin")
	viper.SetDefault("jwt.issuer", "astiango-hub")
	viper.SetDefault("jwt.audience", "astiango-hub-api")
	viper.SetDefault("jwt.access_ttl", "15m")
	viper.SetDefault("jwt.refresh_ttl", "168h")
	viper.SetDefault("jwt.leeway", "30s")
	viper.SetDefault("api.allow.origin", "")
	viper.SetDefault("api.allow.credentials", "false")
	viper.SetDefault("api.limits.body_bytes", 1<<20)
	viper.SetDefault("api.timeouts.request", "30s")
	viper.SetDefault("task.sandbox.image", "goastian/astiango-hub-base:sec-009-011")
	viper.SetDefault("task.sandbox.user", "1000:1000")
	viper.SetDefault("task.sandbox.network", "none")
	viper.SetDefault("task.sandbox.cpus", "1")
	viper.SetDefault("task.sandbox.memory", "512m")
	viper.SetDefault("task.sandbox.pids", 128)
	viper.SetDefault("task.sandbox.disk", "1g")
	viper.SetDefault("task.sandbox.timeout", "30m")
	viper.SetDefault("task.sandbox.tmpfs_size", "64m")
}

func (c *Config) initLogLevel() {
	// set log level
	logLevel := viper.GetString("log.level")
	l, err := log.ParseLevel(logLevel)
	if err != nil {
		l = log.InfoLevel
	}
	log.SetLevel(l)
}

func (c *Config) loadDotEnv() {
	// Try to load .env file, but don't fail if it doesn't exist
	if err := godotenv.Load(); err != nil {
		if c.Logger != nil {
			c.Debug("No .env file found or unable to load .env file")
		}
	} else {
		if c.Logger != nil {
			c.Info("Loaded .env file successfully")
		}
	}
}

func newConfig() *Config {
	return &Config{
		Logger: utils.NewLogger("Config"),
	}
}

var _config *Config
var _configOnce sync.Once

func GetConfig() *Config {
	_configOnce.Do(func() {
		_config = newConfig()
		_config.Init()
	})
	return _config
}

func InitConfig() {
	// config instance
	c := GetConfig()

	// watch config change and load responsively
	c.WatchConfig()
}
