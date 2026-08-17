package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const (
	DefaultWorkspace                  = "astiango_hub_workspace"
	DefaultTaskLogPath                = "/var/log/astiango-hub/tasks"
	DefaultServerHost                 = "0.0.0.0"
	DefaultServerPort                 = 8000
	DefaultGrpcHost                   = "localhost"
	DefaultGrpcPort                   = 9666
	DefaultGrpcServerHost             = "0.0.0.0"
	DefaultGrpcServerPort             = 9666
	DefaultAuthKey                    = "AstianGO Hub2024!"
	DefaultApiEndpoint                = "http://localhost:8000"
	DefaultApiAllowOrigin             = "*"
	DefaultApiAllowCredentials        = "true"
	DefaultApiAllowMethods            = "DELETE, POST, OPTIONS, GET, PUT"
	DefaultApiAllowHeaders            = "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With"
	DefaultApiPort                    = 8080
	DefaultApiPath                    = "/api"
	DefaultNodeMaxRunners             = 20  // Default max concurrent task runners per node
	DefaultTaskQueueSize              = 100 // Default task queue size per node
	DefaultInstallRoot                = "/app/install"
	DefaultInstallEnvs                = ""
	MetadataConfigDirName             = ".astiango-hub"
	MetadataConfigName                = "config.json"
	DefaultPyenvPath                  = "/root/.pyenv"
	DefaultNodeModulesPath            = "/usr/lib/node_modules"
	DefaultNodeBinPath                = "/usr/lib/node_bin"
	DefaultGoPath                     = "/root/go"
	DefaultMCPServerHost              = "0.0.0.0"
	DefaultMCPServerPort              = 9777
	DefaultMCPClientBaseUrl           = "http://localhost:9777/sse"
	DefaultOpenAPIUrlPath             = "/openapi.json"
	DefaultSyncDownloadMaxConcurrency = 16
	DefaultMinFileDescriptorLimit     = 8192
)

func IsDev() bool {
	res := viper.GetBool("dev")
	return res
}

func IsAuthDisabled() bool {
	return viper.GetBool("auth.disabled")
}

func GetAllowOrigin() string {
	if res := viper.GetString("api.allow.origin"); res != "" {
		return res
	}
	return DefaultApiAllowOrigin
}

func GetAllowCredentials() string {
	if res := viper.GetString("api.allow.credentials"); res != "" {
		return res
	}
	return DefaultApiAllowCredentials
}

func GetAllowMethods() string {
	if res := viper.GetString("api.allow.methods"); res != "" {
		return res
	}
	return DefaultApiAllowMethods
}

func GetAllowHeaders() string {
	if res := viper.GetString("api.allow.headers"); res != "" {
		return res
	}
	return DefaultApiAllowHeaders
}

func GetGinMode() string {
	if res := viper.GetString("gin.mode"); res != "" {
		return res
	}
	return gin.ReleaseMode
}

func IsPro() bool {
	return viper.GetString("edition") == "global.edition.pro"
}

func GetWorkspace() string {
	homedirPath, err := homedir.Dir()
	if err != nil {
		logger.Warnf("cannot find home directory: %v", err)
		return DefaultWorkspace
	}
	if res := viper.GetString("workspace"); res != "" {
		return res
	}
	if !Exists(filepath.Join(homedirPath, DefaultWorkspace)) {
		err := os.MkdirAll(filepath.Join(homedirPath, DefaultWorkspace), os.ModePerm)
		if err != nil {
			logger.Warnf("cannot create workspace directory: %v", err)
		}
	}
	return filepath.Join(homedirPath, DefaultWorkspace)
}

func GetTaskLogPath() string {
	if res := viper.GetString("log.path"); res != "" {
		return res
	}
	return DefaultTaskLogPath
}

func GetServerAddress() string {
	host := viper.GetString("server.host")
	if host == "" {
		host = DefaultServerHost
	}

	port := viper.GetInt("server.port")
	if port == 0 {
		port = DefaultServerPort
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func GetMasterHost() string {
	return viper.GetString("master.host")
}

func GetGrpcAddress() string {
	host := viper.GetString("grpc.host")
	if host == "" {
		masterHost := GetMasterHost()
		if masterHost != "" {
			host = masterHost
		} else {
			host = DefaultGrpcHost
		}
	}

	port := viper.GetInt("grpc.port")
	if port == 0 {
		port = DefaultGrpcPort
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func GetGrpcServerAddress() string {
	host := viper.GetString("grpc.server.host")
	if host == "" {
		host = DefaultGrpcServerHost
	}

	port := viper.GetInt("grpc.server.port")
	if port == 0 {
		port = DefaultGrpcServerPort
	}

	return fmt.Sprintf("%s:%d", host, port)
}

func GetAuthKey() string {
	if res := viper.GetString("auth.key"); res != "" {
		return res
	}
	return DefaultAuthKey
}

func GetApiPort() int {
	if viper.GetInt("api.port") > 0 {
		return viper.GetInt("api.port")
	}
	return DefaultApiPort
}

func GetApiPath() string {
	if viper.GetString("api.path") != "" {
		apiPath := viper.GetString("api.path")
		if !strings.HasPrefix(apiPath, "/") {
			apiPath = "/" + apiPath
		}
		return apiPath
	}
	return DefaultApiPath
}

func GetApiEndpoint() string {
	if res := viper.GetString("api.endpoint"); res != "" {
		return res
	}
	masterHost := GetMasterHost()
	if masterHost != "" {
		scheme := "http"
		apiHttps := viper.GetBool("api.https")
		if apiHttps {
			scheme = "https"
		}
		return fmt.Sprintf("%s://%s:%d%s", scheme, masterHost, GetApiPort(), GetApiPath())
	}
	return DefaultApiEndpoint
}

func IsMaster() bool {
	return EnvIsTrue("node.master", false)
}

func GetNodeType() string {
	if IsMaster() {
		return "master"
	} else {
		return "worker"
	}
}

func GetNodeKey() string {
	if res := viper.GetString("node.key"); res != "" {
		return res
	}
	return NewUUIDString()
}

func GetNodeName() string {
	if res := viper.GetString("node.name"); res != "" {
		return res
	}
	return GetNodeKey()
}

func GetNodeMaxRunners() int {
	if res := viper.GetInt("node.maxRunners"); res != 0 {
		return res
	}
	return DefaultNodeMaxRunners
}

func GetTaskQueueSize() int {
	if res := viper.GetInt("task.queue.size"); res != 0 {
		return res
	}
	return DefaultTaskQueueSize
}

func GetMetadataConfigPath() string {
	var homeDirPath, err = homedir.Dir()
	if err != nil {
		logger.Errorf("failed to get home directory: %v", err)
		logger.Errorf("please set metadata directory path using either ASTIANGO_METADATA environment variable or the metadata path in the configuration file")
		panic(err)
	}

	if viper.GetString("metadata") != "" {
		metadataPath := viper.GetString("metadata")
		return filepath.Join(metadataPath, MetadataConfigName)
	}

	return filepath.Join(homeDirPath, MetadataConfigDirName, MetadataConfigName)
}

func GetInstallRoot() string {
	if res := viper.GetString("install.root"); res != "" {
		return res
	}
	return DefaultInstallRoot
}

func GetInstallEnvs() []string {
	if res := viper.GetStringSlice("install.envs"); res != nil {
		return res
	}
	return strings.Split(DefaultInstallEnvs, ",")
}

func GetPyenvPath() string {
	if res := viper.GetString("install.pyenv.path"); res != "" {
		return res
	}
	return DefaultPyenvPath
}

func GetNodeModulesPath() string {
	if res := viper.GetString("install.node.path"); res != "" {
		return res
	}
	return DefaultNodeModulesPath
}

func GetNodeBinPath() string {
	if res := viper.GetString("install.node.bin"); res != "" {
		return res
	}
	return DefaultNodeBinPath
}

func GetGoPath() string {
	if res := viper.GetString("install.go.path"); res != "" {
		return res
	}
	return DefaultGoPath
}

func GetMCPServerAddress() string {
	host := viper.GetString("mcp.server.host")
	if host == "" {
		host = DefaultMCPServerHost
	}
	port := viper.GetInt("mcp.server.port")
	if port == 0 {
		port = DefaultMCPServerPort
	}
	return fmt.Sprintf("%s:%d", host, port)
}

func GetMCPClientBaseUrl() string {
	if res := viper.GetString("mcp.client.base_url"); res != "" {
		return res
	}
	return DefaultMCPClientBaseUrl
}

func GetOpenAPIUrl() string {
	if res := viper.GetString("openapi.url"); res != "" {
		return res
	}
	return GetApiEndpoint() + DefaultOpenAPIUrlPath
}

func GetSyncDownloadMaxConcurrency() int64 {
	if res := viper.GetInt("sync.download.max_concurrency"); res > 0 {
		return int64(res)
	}
	return int64(DefaultSyncDownloadMaxConcurrency)
}

func GetMinFileDescriptorLimit() uint64 {
	if res := viper.GetUint64("system.fd_min"); res > 0 {
		return res
	}
	return DefaultMinFileDescriptorLimit
}

func IsSyncGrpcEnabled() bool {
	return viper.GetBool("sync.useGrpc")
}
