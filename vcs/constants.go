package vcs

const (
	GitRemoteNameOrigin      = "origin"
	GitRemoteNameUpstream    = "upstream"
	GitRemoteNameAstianGoHub = "astiango-hub"
)
const GitDefaultRemoteName = GitRemoteNameOrigin

const (
	GitBranchNameMaster  = "master"
	GitBranchNameMain    = "main"
	GitBranchNameRelease = "release"
	GitBranchNameTest    = "test"
	GitBranchNameDevelop = "develop"
)
const GitDefaultBranchName = GitBranchNameMaster

type GitAuthType int

const (
	GitAuthTypeNone GitAuthType = iota
	GitAuthTypeHTTP
	GitAuthTypeSSH
)

type GitInitType int

const (
	GitInitTypeFs GitInitType = iota
	GitInitTypeMem
)

const (
	GitRefTypeBranch = "branch"
	GitRefTypeTag    = "tag"
)
