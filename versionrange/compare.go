package versionrange

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"slices"
	"strings"

	"deps.dev/util/semver"
	versioncompare "github.com/yanqin001/version-compare"
	"github.com/yanqin001/version-compare/osversion"
)

// 标签归一化沿用迁移前的完整匹配规则，只转换已知包装，避免把任意 Git 引用误当数字版本。
var numericVersion = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*$`)
var fullCommit = regexp.MustCompile(`(?i)^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)
var calendarVersion = regexp.MustCompile(`^[0-9]+(?:[._-][0-9]+)*$`)
var releaseAlias = regexp.MustCompile(`(?i)^(v?[0-9]+(?:\.[0-9]+)*)-(?:stable|final|ga|custom)$`)
var releaseWrapper = regexp.MustCompile(`(?i)^release[-_](v?[0-9].*)$`)
var releaseTag = regexp.MustCompile(`(?i)^r([0-9]+)r?v([0-9]+)(?:v([0-9]+))?([-+][0-9a-z][0-9a-z.+_-]*)?$`)

// Comparator 是不可变的单一版本比较策略；通过 NewComparator 创建，可并发复用。
// 不执行包管理器的依赖解析规则；范围开闭语义由调用方单独判断。
type Comparator struct {
	classification Classification
}

// NewComparator 校验分类并构建比较器；GIT/UNKNOWN 和缺少顺序的 custom 返回 ErrUnordered。
// classification 必须显式声明；缺失、未知枚举或字段冲突返回 ErrInvalidClassification。
func NewComparator(classification Classification) (*Comparator, error) {
	c := classification.Canonical()
	if !c.Present() {
		return nil, fmt.Errorf("%w: 比较器必须声明分类", ErrInvalidClassification)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if c.RangeType != TypeVersion || c.VersionScheme == "custom" && len(c.VersionOrder) == 0 {
		return nil, ErrUnordered
	}
	return &Comparator{classification: c}, nil
}

// Compare 比较原始版本 a 与 b，成功返回 -1、0、1；错误时顺序值不可使用。
// 非法版本返回根包 ErrInvalidVersion，缺少排序依据返回 ErrUnordered；未初始化返回 ErrInvalidClassification。
// 边界与候选应用相同的包装规则；不会将 Git 哈希或未知标签按字典序比较。
func (c *Comparator) Compare(a, b string) (int, error) {
	if c == nil || c.classification.RangeType != TypeVersion {
		return 0, fmt.Errorf("%w: 比较器未初始化", ErrInvalidClassification)
	}
	if fullCommit.MatchString(strings.TrimSpace(a)) || fullCommit.MatchString(strings.TrimSpace(b)) {
		return 0, fmt.Errorf("%w: 完整提交 ID 不参与发布版本比较", ErrUnordered)
	}
	if c.classification.VersionScheme == "custom" {
		i, j := slices.Index(c.classification.VersionOrder, a), slices.Index(c.classification.VersionOrder, b)
		if i < 0 || j < 0 {
			return 0, fmt.Errorf("%w: %q 或 %q 不在 version_order 中", ErrUnordered, a, b)
		}
		return sign(i - j), nil
	}
	a, err := c.normalize(a)
	if err != nil {
		return 0, err
	}
	b, err = c.normalize(b)
	if err != nil {
		return 0, err
	}
	scheme := c.classification.VersionScheme
	switch scheme {
	case "numeric":
		return compareNumeric(a, b)
	case "calver":
		if !calendarVersion.MatchString(a) || !calendarVersion.MatchString(b) {
			return 0, fmt.Errorf("%w: 日历版本格式无效: %q / %q", versioncompare.ErrInvalidVersion, a, b)
		}
		replacer := strings.NewReplacer("-", ".", "_", ".")
		return compareNumeric(replacer.Replace(a), replacer.Replace(b))
	case "dpkg", "rpm", "apk", "alpm":
		// 旧库使用 0/1/2 表示小于/等于/大于，适配到标准比较器的 -1/0/1。
		order, err := osversion.NewService().Compare(osversion.Distribution(scheme), a, b)
		if err != nil {
			return 0, fmt.Errorf("%w: %w", versioncompare.ErrInvalidVersion, err)
		}
		return order - 1, nil
	default:
		system := ecosystemSystem(scheme)
		left, err := system.Parse(a)
		if err != nil {
			return 0, fmt.Errorf("%w: %s 版本 %q 无效: %w", versioncompare.ErrInvalidVersion, scheme, a, err)
		}
		right, err := system.Parse(b)
		if err != nil {
			return 0, fmt.Errorf("%w: %s 版本 %q 无效: %w", versioncompare.ErrInvalidVersion, scheme, b, err)
		}
		if left.IsWildcard() || right.IsWildcard() {
			return 0, fmt.Errorf("%w: %s 比较对象不能包含通配版本", versioncompare.ErrInvalidVersion, scheme)
		}
		return sign(left.Compare(right)), nil
	}
}

// InBranch 判断 version 是否属于数字维护分支 branch，供调用方应用分支修复上界。
// 无法解析分支时返回 false，不把任意前缀误当成版本分支（例如 1.2 不能匹配 1.20）。
func (c *Comparator) InBranch(version, branch string) bool {
	if c == nil || c.classification.RangeType != TypeVersion {
		return false
	}
	value, err := c.normalize(version)
	if err != nil {
		return false
	}
	branch = strings.TrimPrefix(strings.TrimSpace(branch), "v")
	value = strings.TrimPrefix(value, "v")
	if !numericVersion.MatchString(branch) || !strings.HasPrefix(value, branch) {
		return false
	}
	return len(value) == len(branch) || strings.ContainsRune(".-+", rune(value[len(branch)]))
}

// normalize 仅剥离声明的标签包装及正式版别名，保留生态限定符和 OS 原生版本字符。
func (c *Comparator) normalize(version string) (string, error) {
	value := strings.TrimSpace(version)
	prefix, suffix := c.classification.VersionPrefix, c.classification.VersionSuffix
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) || len(value) <= len(prefix)+len(suffix) {
		return "", fmt.Errorf("%w: %q 不符合已声明的版本包装", ErrUnordered, version)
	}
	value = value[len(prefix) : len(value)-len(suffix)]
	switch c.classification.VersionScheme {
	case "dpkg", "rpm", "apk", "alpm", "custom":
		return value, nil
	}
	if parts := releaseWrapper.FindStringSubmatch(value); parts != nil {
		value = parts[1]
	}
	if c.classification.VersionScheme == "numeric" || c.classification.VersionScheme == "semver" {
		if parts := releaseTag.FindStringSubmatch(value); parts != nil {
			value = parts[1] + "." + parts[2]
			if parts[3] != "" {
				value += "." + parts[3]
			}
			value += parts[4]
		}
	}
	// 已知 r 标签先转换为数字再应用正式版别名，避免 r1rv78-stable 与 1.78 得出不同顺序。
	if parts := releaseAlias.FindStringSubmatch(value); parts != nil {
		value = parts[1]
	}
	if len(value) > 1 && (value[0] == 'v' || value[0] == 'V') && value[1] >= '0' && value[1] <= '9' {
		value = value[1:]
	}
	if c.classification.VersionScheme == "go" {
		value = "v" + value
	}
	return value, nil
}

// ValidateRange 校验原始边界 begin/end 的顺序；endInclusive 为 true 时允许相等。
// 空分类允许历史数据沿用旧规则；逆序或半开空区间返回根包 ErrInvalidExpression。
// GIT/UNKNOWN 或未提供顺序的 custom 不推断大小；可排序体系的解析失败、逆序及空区间返回错误。
func (c Classification) ValidateRange(begin, end string, endInclusive bool) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if !c.Present() {
		return nil
	}
	comparator, err := NewComparator(c)
	if errors.Is(err, ErrUnordered) {
		return nil
	}
	if err != nil {
		return err
	}
	order, err := comparator.Compare(begin, end)
	if err != nil {
		return err
	}
	if order > 0 || order == 0 && !endInclusive {
		return fmt.Errorf("%w: begin/end 版本顺序不合法；%s 要求 begin < end（闭区间允许相等）", versioncompare.ErrInvalidExpression, c.VersionScheme)
	}
	return nil
}

func ecosystemSystem(scheme string) semver.System {
	switch scheme {
	case "pep440":
		return semver.PyPI
	case "maven":
		return semver.Maven
	case "go":
		return semver.Go
	case "npm":
		return semver.NPM
	case "cargo":
		return semver.Cargo
	case "nuget":
		return semver.NuGet
	case "rubygems":
		return semver.RubyGems
	case "composer":
		return semver.Composer
	default:
		return semver.DefaultSystem
	}
}

// compareNumeric 使用任意精度数字避免构建号溢出，缺失的尾部段按零处理。
func compareNumeric(a, b string) (int, error) {
	if !numericVersion.MatchString(a) || !numericVersion.MatchString(b) {
		return 0, fmt.Errorf("%w: 数字版本无效: %q / %q", versioncompare.ErrInvalidVersion, a, b)
	}
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for index := 0; index < max(len(left), len(right)); index++ {
		x, y := new(big.Int), new(big.Int)
		if index < len(left) {
			x.SetString(left[index], 10)
		}
		if index < len(right) {
			y.SetString(right[index], 10)
		}
		if order := x.Cmp(y); order != 0 {
			return order, nil
		}
	}
	return 0, nil
}

func sign(value int) int {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}
