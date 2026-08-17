package constants

const (
	DependencyTypePython  = "python"
	DependencyTypeNode    = "node"
	DependencyTypeGo      = "go"
	DependencyTypeJava    = "java"
	DependencyTypeBrowser = "browser"
)

const (
	DependencyStatusInstalling   = "installing"
	DependencyStatusInstalled    = "installed"
	DependencyStatusUninstalling = "uninstalling"
	DependencyStatusUninstalled  = "uninstalled"
	DependencyStatusError        = "error"
	DependencyStatusAbnormal     = "abnormal"
)

const (
	DependencyFileTypeRequirementsTxt = "requirements.txt"
	DependencyFileTypePackageJson     = "package.json"
	DependencyFileTypeGoMod           = "go.mod"
	DependencyFileTypePomXml          = "pom.xml"
)
const (
	DependencyActionSync  = "sync"
	DependencyActionSetup = "setup"
)
