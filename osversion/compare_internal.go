package osversion

import (
	"fmt"
	"strings"

	"github.com/yanqin001/version-compare/internal/versioncore"
)

// algorithm 标识操作系统软件包采用的原生版本比较规则。
type algorithm uint8

const (
	algorithmGeneric algorithm = iota
	algorithmRPM
	algorithmDebian
	algorithmAPK
	algorithmALPM
)

// nativeVersion 保存已经按发行版原生规则解析的版本。
// 每次只使用 family 对应的字段，避免 Matcher 在遍历多个区间时重复解析版本字符串。
type nativeVersion struct {
	family  algorithm
	rpm     rpmVersion
	debian  debianVersion
	apk     string
	alpm    alpmVersion
	generic versioncore.Version
}

// resolveAlgorithm 将发行版名称及常用包管理器别名映射为稳定的算法族。
// 未识别名称沿用 generic，避免旧数据中的自由文本突然失去可比性。
func resolveAlgorithm(distribution string) algorithm {
	key := normalizeDistribution(distribution)
	if key == "" || key == "os" || key == "generic" {
		return algorithmGeneric
	}

	if resolved, ok := lookupAlgorithm(key); ok {
		return resolved
	}
	// 历史 Type 常见 os:rhel、os_ubuntu 及 oscentos 等写法，规范化后统一去掉 os 前缀再解析。
	if strings.HasPrefix(key, "os") && len(key) > len("os") {
		if resolved, ok := lookupAlgorithm(key[len("os"):]); ok {
			return resolved
		}
	}
	return algorithmGeneric
}

// compareNative 按 distribution 对应的原生规则比较 a 与 b。
// 返回值固定为 -1、0 或 1；格式不符合对应包管理器约束时返回错误。
func compareNative(distribution, a, b string) (int, error) {
	family := resolveAlgorithm(distribution)
	left, err := parseNativeVersion(family, a)
	if err != nil {
		return 0, err
	}
	right, err := parseNativeVersion(family, b)
	if err != nil {
		return 0, err
	}
	return compareParsedNative(left, right)
}

// parseNativeVersion 按确定的算法族解析版本，返回的对象可被重复比较。
func parseNativeVersion(family algorithm, value string) (nativeVersion, error) {
	result := nativeVersion{family: family}
	var err error
	switch family {
	case algorithmRPM:
		result.rpm, err = parseRPM(value)
	case algorithmDebian:
		result.debian, err = parseDebian(value)
	case algorithmAPK:
		err = validateAPK(value)
		result.apk = value
	case algorithmALPM:
		result.alpm, err = parseALPM(value)
	default:
		result.generic, err = versioncore.Parse(value, versioncore.GenericOS)
	}
	if err != nil {
		return nativeVersion{}, err
	}
	return result, nil
}

// compareParsedNative 比较两个已解析的同族版本。
func compareParsedNative(left, right nativeVersion) (int, error) {
	if left.family != right.family {
		return 0, fmt.Errorf("OS version algorithms differ: %d and %d", left.family, right.family)
	}
	switch left.family {
	case algorithmRPM:
		return compareParsedRPM(left.rpm, right.rpm), nil
	case algorithmDebian:
		return compareParsedDebian(left.debian, right.debian), nil
	case algorithmAPK:
		return compareValidatedAPK(left.apk, right.apk), nil
	case algorithmALPM:
		return compareParsedALPM(left.alpm, right.alpm), nil
	default:
		return versioncore.Compare(left.generic, right.generic, false)
	}
}

// lookupAlgorithm 查找已经去除分隔符且转换为小写的发行版名称。
func lookupAlgorithm(distribution string) (algorithm, bool) {
	switch distribution {
	case "rpm",
		"rhel", "redhat", "redhatenterprise", "redhatenterpriselinux",
		"centos", "centoslinux", "centosstream",
		"fedora",
		"anolis", "anolisos", "openanolis",
		"openeuler", "euleros",
		"suse", "sles", "sled", "suselinuxenterprise",
		"opensuse", "opensuseleap", "opensusetumbleweed",
		"rocky", "rockylinux", "alma", "almalinux",
		"oraclelinux", "ol",
		"amazonlinux", "amzn",
		"alinux", "alibabacloudlinux":
		return algorithmRPM, true
	case "deb", "dpkg", "debian", "ubuntu",
		"linuxmint", "mint", "kali", "kalilinux", "raspbian",
		"deepin", "uos", "uniontechos":
		return algorithmDebian, true
	case "apk", "alpine", "alpinelinux", "wolfi", "chainguard":
		return algorithmAPK, true
	case "alpm", "pacman", "arch", "archlinux", "manjaro", "endeavouros":
		return algorithmALPM, true
	default:
		return algorithmGeneric, false
	}
}

// normalizeDistribution 仅规范化发行版标识，不得用于版本字符串本身。
func normalizeDistribution(distribution string) string {
	distribution = strings.TrimSpace(distribution)
	if distribution == "" {
		return ""
	}

	var normalized strings.Builder
	normalized.Grow(len(distribution))
	for index := 0; index < len(distribution); index++ {
		character := distribution[index]
		switch {
		case character >= 'A' && character <= 'Z':
			normalized.WriteByte(character + ('a' - 'A'))
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			normalized.WriteByte(character)
		}
	}
	return normalized.String()
}

// compareNumericStrings 比较任意长度的非负十进制整数，避免转换为固定宽度整数产生溢出。
func compareNumericStrings(left, right string) int {
	leftStart := 0
	for leftStart < len(left) && left[leftStart] == '0' {
		leftStart++
	}
	rightStart := 0
	for rightStart < len(right) && right[rightStart] == '0' {
		rightStart++
	}

	left = left[leftStart:]
	right = right[rightStart:]
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return compareBytes(left, right)
}

// compareBytes 按字节序比较字符串并将结果收敛到 -1、0、1。
func compareBytes(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func isASCIIDigit(character byte) bool {
	return character >= '0' && character <= '9'
}

func isASCIIAlpha(character byte) bool {
	return character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z'
}

func isASCIIAlphanumeric(character byte) bool {
	return isASCIIDigit(character) || isASCIIAlpha(character)
}

// invalidNativeVersion 构造包含算法、输入和字节位置的稳定错误，便于定位脏数据。
func invalidNativeVersion(family, input string, offset int, reason string) error {
	return fmt.Errorf("osversion: invalid %s version %q at byte %d: %s", family, input, offset, reason)
}
