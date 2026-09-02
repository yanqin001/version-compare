package osversion

// CompareResult 是 OS Service 的稳定比较结果。
type CompareResult int

const (
	// CompareLess 表示左侧版本小于右侧版本。
	CompareLess CompareResult = iota
	// CompareEqual 表示两个版本等价。
	CompareEqual
	// CompareGreater 表示左侧版本大于右侧版本。
	CompareGreater
)

// Distribution 表示操作系统发行版或包管理器算法族。
type Distribution string

const (
	// DistributionGeneric 使用兼容旧项目的通用 OS 比较算法。
	DistributionGeneric Distribution = "os"
	// DistributionRPM 显式使用 RPM EVR 规则。
	DistributionRPM Distribution = "rpm"
	// DistributionDPKG 显式使用 Debian/dpkg 规则，并允许该算法族的任意发行版。
	DistributionDPKG Distribution = "dpkg"
	// DistributionAPK 显式使用 Alpine/apk 规则，并允许该算法族的任意发行版。
	DistributionAPK Distribution = "apk"
	// DistributionALPM 显式使用 Arch Linux/ALPM 规则，并允许该算法族的任意发行版。
	DistributionALPM Distribution = "alpm"
	// DistributionDebian 表示 Debian/dpkg 规则。
	DistributionDebian Distribution = "debian"
	// DistributionUbuntu 表示 Ubuntu 使用的 Debian/dpkg 规则。
	DistributionUbuntu Distribution = "ubuntu"
	// DistributionAlpine 表示 Alpine/apk 规则。
	DistributionAlpine Distribution = "alpine"
	// DistributionArch 表示 Arch Linux/ALPM 规则。
	DistributionArch Distribution = "arch"
	// DistributionAnolis 表示 Anolis OS 使用的 RPM 规则。
	DistributionAnolis Distribution = "anolis"
	// DistributionRHEL 表示 Red Hat Enterprise Linux 使用的 RPM 规则。
	DistributionRHEL Distribution = "rhel"
	// DistributionCentOS 表示 CentOS 使用的 RPM 规则。
	DistributionCentOS Distribution = "centos"
	// DistributionFedora 表示 Fedora 使用的 RPM 规则。
	DistributionFedora Distribution = "fedora"
	// DistributionOpenEuler 表示 openEuler 使用的 RPM 规则。
	DistributionOpenEuler Distribution = "openeuler"
	// DistributionSUSE 表示 SUSE/openSUSE 使用的 RPM 规则。
	DistributionSUSE Distribution = "suse"
	// DistributionRockyLinux 表示 Rocky Linux 使用的 RPM 规则。
	DistributionRockyLinux Distribution = "rocky"
	// DistributionAlmaLinux 表示 AlmaLinux 使用的 RPM 规则。
	DistributionAlmaLinux Distribution = "almalinux"
	// DistributionOracleLinux 表示 Oracle Linux 使用的 RPM 规则。
	DistributionOracleLinux Distribution = "oraclelinux"
	// DistributionAmazonLinux 表示 Amazon Linux 使用的 RPM 规则。
	DistributionAmazonLinux Distribution = "amazonlinux"
	// DistributionAlibabaCloudLinux 表示 Alibaba Cloud Linux 使用的 RPM 规则。
	DistributionAlibabaCloudLinux Distribution = "alinux"
	// DistributionOpenSUSE 表示 openSUSE 使用的 RPM 规则。
	DistributionOpenSUSE Distribution = "opensuse"
	// DistributionEulerOS 表示 EulerOS 使用的 RPM 规则。
	DistributionEulerOS Distribution = "euleros"
	// DistributionLinuxMint 表示 Linux Mint 使用的 Debian/dpkg 规则。
	DistributionLinuxMint Distribution = "linuxmint"
	// DistributionKali 表示 Kali Linux 使用的 Debian/dpkg 规则。
	DistributionKali Distribution = "kali"
	// DistributionRaspbian 表示 Raspbian 使用的 Debian/dpkg 规则。
	DistributionRaspbian Distribution = "raspbian"
	// DistributionDeepin 表示 Deepin 使用的 Debian/dpkg 规则。
	DistributionDeepin Distribution = "deepin"
	// DistributionUOS 表示 UOS 使用的 Debian/dpkg 规则。
	DistributionUOS Distribution = "uos"
	// DistributionWolfi 表示 Wolfi 使用的 APK 规则。
	DistributionWolfi Distribution = "wolfi"
	// DistributionChainguard 表示 Chainguard 使用的 APK 规则。
	DistributionChainguard Distribution = "chainguard"
	// DistributionManjaro 表示 Manjaro 使用的 ALPM 规则。
	DistributionManjaro Distribution = "manjaro"
	// DistributionEndeavourOS 表示 EndeavourOS 使用的 ALPM 规则。
	DistributionEndeavourOS Distribution = "endeavouros"
)

// Package 描述待匹配的操作系统软件包。
//
// Version 应优先传纯包版本，也兼容末尾带 "/architecture" 的旧 Anolis 格式。
// OSVersion 和 Architecture 显式值优先；为空时匹配器会尝试从已知旧格式提取。
type Package struct {
	// Distribution 是候选软件包所属发行版；为空表示未提供。
	Distribution Distribution
	// Version 是发行版原生软件包版本。
	Version string
	// OSVersion 是操作系统发行版本；为空时尝试从旧格式提取。
	OSVersion string
	// Architecture 是软件包架构；为空时尝试从 Version 的斜杠后缀提取。
	Architecture string
}

// Constraint 描述漏洞对操作系统软件包的联合约束。
//
// VersionExpression 使用与根包相同的范围语法；OSVersionExpression 为空时不限制
// 操作系统版本。Architectures 为空时不限制架构。
type Constraint struct {
	// Distribution 选择发行版及其原生版本算法。
	Distribution Distribution
	// VersionExpression 是必填的软件包漏洞范围表达式。
	VersionExpression string
	// OSVersionExpression 是可选的操作系统发行版本范围。
	OSVersionExpression string
	// Architectures 是可选的允许架构集合。
	Architectures []string
}

// VersionExp 描述一次操作系统软件包漏洞范围匹配。
type VersionExp struct {
	// Package 是待判断的软件包。
	Package Package
	// Constraint 是要匹配的漏洞约束。
	Constraint Constraint
}

// Service 定义操作系统软件包版本、OS 版本与架构匹配接口。
type Service interface {
	// InExp 判断候选软件包是否满足联合漏洞约束。
	InExp(VersionExp) (bool, error)
	// Compare 按发行版原生规则比较两个软件包版本。
	Compare(distribution Distribution, v1, v2 string) (int, error)
	// MatchOSVersion 判断操作系统发行版本是否满足范围表达式。
	MatchOSVersion(distribution Distribution, version, expression string) (bool, error)
	// MatchArchitecture 判断候选架构是否位于允许集合中。
	MatchArchitecture(actual string, allowed []string) (bool, error)
}
