package versioncompare

// CompareResult 是 VersionSvc 为兼容既有调用方使用的比较结果。
type CompareResult int

const (
	// CompareLess 表示左侧版本小于右侧版本。
	CompareLess CompareResult = iota
	// CompareEqual 表示两个版本等价。
	CompareEqual
	// CompareGreater 表示左侧版本大于右侧版本。
	CompareGreater
)

const (
	// VersionTypeNormal 表示普通开源组件版本。
	VersionTypeNormal = "normal"
	// VersionTypeOS 表示使用旧算法兼容的通用 OS 版本。
	VersionTypeOS = "os"
)

// VersionSvc 定义兼容既有项目的版本服务接口。
//
// Compare 成功时返回 CompareLess、CompareEqual 或 CompareGreater 对应的整数值。
// Sort 会稳定地原地修改传入切片，并返回同一个切片。
type VersionSvc interface {
	// InExp 判断候选版本是否位于漏洞范围内。
	InExp(VersionExp) (bool, error)
	// GetMaxVersion 返回列表中的最大版本。
	GetMaxVersion(versions []string) (string, error)
	// GetMinVersion 返回列表中的最小版本。
	GetMinVersion(versions []string) (string, error)
	// Compare 比较两个普通组件版本，返回兼容的 0、1、2 结果。
	Compare(v1, v2 string) (int, error)
	// Sort 稳定地原地升序排列版本列表。
	Sort(versions []string) ([]string, error)
	// GetClosestVersion 返回列表中与目标距离最近的版本。
	GetClosestVersion(version string, versions []string) (string, error)
}

// VersionExp 描述一次普通组件或通用 OS 版本范围匹配。
// Version 是候选版本，Expression 是漏洞范围，Type 为空或 normal 时表示普通版本，
// os 表示旧算法兼容的通用 OS 版本。Fixed 可使用 "4.14:4.14.272" 或
// "4.14.272"，MatchMain 表示忽略预发布及附加修订，但仍比较 MainSupplement。
type VersionExp struct {
	// Version 是待判断的候选版本。
	Version string
	// Expression 是漏洞版本范围表达式。
	Expression string
	// Type 选择普通版本或兼容旧实现的通用 OS 版本规则。
	Type string
	// Fixed 是兼容旧接口的分支修复版本列表。
	Fixed []string
	// MatchMain 表示忽略预发布和附加修订字段。
	MatchMain bool
}

// VersionKind 指定编译 Matcher 时采用的版本规则。
type VersionKind uint8

const (
	// NormalVersion 表示普通开源组件版本。
	NormalVersion VersionKind = iota
	// GenericOSVersion 表示旧算法兼容的通用 OS 版本。
	GenericOSVersion
)

// FixedVersion 描述维护分支及其首个安全版本。
// Branch 是分支前缀，Version 是不再受漏洞影响的排他上界。
type FixedVersion struct {
	// Branch 是数字维护分支前缀，例如 "4.14"。
	Branch string
	// Version 是该分支首个不受漏洞影响的版本。
	Version string
}

// CompileOptions 控制漏洞范围的编译行为。
type CompileOptions struct {
	// Kind 选择普通组件或通用 OS 兼容规则；零值表示普通组件。
	Kind VersionKind
	// FixedVersions 用分支安全版本收紧漏洞上界。
	FixedVersions []FixedVersion
	// MatchMain 表示所有边界比较都忽略预发布和附加修订字段。
	MatchMain bool
}
