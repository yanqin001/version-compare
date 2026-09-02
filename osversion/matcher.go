package osversion

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yanqin001/version-compare/internal/rangeexpr"
	"github.com/yanqin001/version-compare/internal/versioncore"
)

type nativeBound struct {
	version   nativeVersion
	inclusive bool
	set       bool
}

type nativeInterval struct {
	lower nativeBound
	upper nativeBound
	exact bool
}

type nativeExpression struct {
	distribution string
	family       algorithm
	release      bool
	any          bool
	intervals    []nativeInterval
}

// Matcher 是编译后的 OS 软件包联合匹配器。
//
// Matcher 会复制并规范化所有约束，编译后不可变，可由多个 goroutine 并发使用。
// 零值 Matcher 不可用，应通过 Compile 创建。
type Matcher struct {
	distribution Distribution
	version      *nativeExpression
	osVersion    *nativeExpression
	architecture *ArchitectureMatcher
}

// Compile 编译软件包版本、OS 版本和架构联合约束。
// VersionExpression 必填，其他维度为空表示不限制。未知 Distribution 会使用通用兼容
// 算法；范围、原生版本或架构约束无效时会打印日志并返回可检查的错误。
func Compile(constraint Constraint) (*Matcher, error) {
	matcher, err := compile(constraint)
	if err != nil {
		return nil, logFailure("Compile", err)
	}
	return matcher, nil
}

func compile(constraint Constraint) (*Matcher, error) {
	distribution := constraint.Distribution
	if strings.TrimSpace(string(distribution)) == "" {
		distribution = DistributionGeneric
	}
	versionMatcher, err := compileNativeExpression(string(distribution), constraint.VersionExpression, false)
	if err != nil {
		return nil, err
	}
	osVersionMatcher, err := compileNativeExpression(string(distribution), constraint.OSVersionExpression, true)
	if err != nil {
		return nil, err
	}
	architectureMatcher, err := CompileArchitectures(constraint.Architectures)
	if err != nil {
		return nil, err
	}
	return &Matcher{
		distribution: distribution,
		version:      versionMatcher,
		osVersion:    osVersionMatcher,
		architecture: architectureMatcher,
	}, nil
}

// Match 判断 pkg 是否同时满足已编译的发行版、OS 版本、架构和包版本约束。
// 返回值表示是否命中。Matcher 未初始化或版本解析失败时会打印日志并返回错误；
// 普通不匹配返回 false、nil。
func (m *Matcher) Match(pkg Package) (bool, error) {
	matched, err := m.match(pkg)
	if err != nil {
		return false, logFailure("Matcher.Match", err)
	}
	return matched, nil
}

func (m *Matcher) match(pkg Package) (bool, error) {
	if m == nil || m.version == nil || m.architecture == nil {
		return false, fmt.Errorf("%w: uninitialized matcher", ErrInvalidExpression)
	}
	if !distributionMatches(pkg.Distribution, m.distribution) {
		return false, nil
	}

	version, embeddedOSVersion, embeddedArchitecture := splitEmbeddedMetadata(pkg.Version)
	architecture := pkg.Architecture
	if strings.TrimSpace(architecture) == "" {
		architecture = embeddedArchitecture
	}
	if !m.architecture.Match(architecture) {
		return false, nil
	}

	if m.osVersion != nil {
		osVersion := pkg.OSVersion
		if strings.TrimSpace(osVersion) == "" {
			osVersion = embeddedOSVersion
		}
		if strings.TrimSpace(osVersion) == "" {
			return false, nil
		}
		matched, err := m.osVersion.match(osVersion)
		if err != nil || !matched {
			return false, err
		}
	}
	return m.version.match(version)
}

func compileNativeExpression(distribution, expression string, release bool) (*nativeExpression, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		if release {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: package version expression is empty", ErrInvalidExpression)
	}
	parsed, err := rangeexpr.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidExpression, err)
	}
	result := &nativeExpression{
		distribution: distribution,
		family:       resolveAlgorithm(distribution),
		release:      release,
		any:          parsed.Any,
	}
	if result.any {
		return result, nil
	}
	result.intervals = make([]nativeInterval, 0, len(parsed.Intervals))
	for _, interval := range parsed.Intervals {
		if !interval.Lower.Set && !interval.Upper.Set {
			return nil, fmt.Errorf("%w: interval cannot have two open ends", ErrInvalidExpression)
		}
		compiled := nativeInterval{}
		if interval.Lower.Set {
			compiled.lower, err = compileNativeBound(
				result.family,
				normalizeNativeBound(distribution, interval.Lower.Version, release),
				interval.Lower.Inclusive,
			)
			if err != nil {
				return nil, err
			}
		}
		if interval.Upper.Set {
			compiled.upper, err = compileNativeBound(
				result.family,
				normalizeNativeBound(distribution, interval.Upper.Version, release),
				interval.Upper.Inclusive,
			)
			if err != nil {
				return nil, err
			}
		}
		if compiled.lower.set && compiled.upper.set {
			comparison, compareErr := compareParsedNative(compiled.lower.version, compiled.upper.version)
			if compareErr != nil {
				return nil, fmt.Errorf("%w: %v", ErrInvalidVersion, compareErr)
			}
			if comparison > 0 {
				return nil, fmt.Errorf("%w: lower bound is greater than upper bound", ErrInvalidExpression)
			}
			compiled.exact = comparison == 0 && compiled.lower.inclusive && compiled.upper.inclusive
		}
		result.intervals = append(result.intervals, compiled)
	}
	return result, nil
}

func (e *nativeExpression) match(version string) (bool, error) {
	if e == nil {
		return true, nil
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return false, ErrEmptyVersion
	}
	if e.release {
		version = normalizeOSRelease(e.distribution, version)
	}
	candidate, err := parseNativeVersion(e.family, version)
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrInvalidVersion, err)
	}
	if e.any {
		// 通配符只跳过范围判断，不跳过发行版原生语法校验，避免脏版本静默命中漏洞。
		return true, nil
	}
	for _, interval := range e.intervals {
		if interval.exact {
			comparison, compareErr := compareParsedNative(candidate, interval.lower.version)
			if errors.Is(compareErr, versioncore.ErrIncomparable) {
				return false, nil
			}
			if compareErr != nil {
				return false, fmt.Errorf("%w: %v", ErrInvalidVersion, compareErr)
			}
			if comparison == 0 {
				return true, nil
			}
			continue
		}
		if interval.lower.set {
			comparison, err := compareParsedNative(candidate, interval.lower.version)
			if err != nil {
				return false, fmt.Errorf("%w: %v", ErrInvalidVersion, err)
			}
			if comparison < 0 || (comparison == 0 && !interval.lower.inclusive) {
				continue
			}
		}
		if interval.upper.set {
			comparison, err := compareParsedNative(candidate, interval.upper.version)
			if err != nil {
				return false, fmt.Errorf("%w: %v", ErrInvalidVersion, err)
			}
			if comparison > 0 || (comparison == 0 && !interval.upper.inclusive) {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}

func compileNativeBound(family algorithm, version string, inclusive bool) (nativeBound, error) {
	parsed, err := parseNativeVersion(family, version)
	if err != nil {
		return nativeBound{}, fmt.Errorf("%w: %v", ErrInvalidVersion, err)
	}
	return nativeBound{version: parsed, inclusive: inclusive, set: true}, nil
}

func normalizeNativeBound(distribution, version string, release bool) string {
	if release {
		return normalizeOSRelease(distribution, version)
	}
	value, _, _ := splitEmbeddedMetadata(version)
	return value
}
