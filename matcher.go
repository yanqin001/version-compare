package versioncompare

import (
	"errors"
	"fmt"
	"strings"

	"github.com/yanqin001/version-compare/internal/rangeexpr"
	"github.com/yanqin001/version-compare/internal/versioncore"
)

type compiledBound struct {
	version       versioncore.Version
	inclusive     bool
	set           bool
	autoMatchMain bool
}

type compiledInterval struct {
	lower compiledBound
	upper compiledBound
	exact bool
}

// Matcher 是编译后的普通组件或通用 OS 漏洞范围匹配器。
//
// Matcher 在编译后不再修改内部状态，可由多个 goroutine 并发调用。零值 Matcher
// 不可用，应通过 Compile 创建。
type Matcher struct {
	any       bool
	kind      versioncore.Kind
	intervals []compiledInterval
	fixed     []compiledFixed
	matchMain bool
}

// Compile 编译漏洞范围及固定版本分支。
// expression 是漏洞范围，options 指定版本类型、固定分支和主版本匹配行为。
// 表达式、版本类型或固定分支无效时会打印日志并返回可通过 errors.Is 检查的错误。
func Compile(expression string, options CompileOptions) (*Matcher, error) {
	matcher, err := compile(expression, options)
	if err != nil {
		return nil, logFailure("Compile", err)
	}
	return matcher, nil
}

func compile(expression string, options CompileOptions) (*Matcher, error) {
	kind, err := coreKind(options.Kind)
	if err != nil {
		return nil, err
	}
	parsedExpression, err := rangeexpr.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidExpression, err)
	}
	fixedVersions, err := compileFixedVersions(options.FixedVersions, kind)
	if err != nil {
		return nil, err
	}
	matcher := &Matcher{
		any:       parsedExpression.Any,
		kind:      kind,
		fixed:     fixedVersions,
		matchMain: options.MatchMain,
	}
	if matcher.any {
		return matcher, nil
	}

	matcher.intervals = make([]compiledInterval, 0, len(parsedExpression.Intervals))
	for _, interval := range parsedExpression.Intervals {
		compiled, compileErr := compileInterval(interval, kind)
		if compileErr != nil {
			return nil, compileErr
		}
		matcher.intervals = append(matcher.intervals, compiled)
	}
	return matcher, nil
}

// Match 判断 version 是否命中已编译范围。
// 返回值表示是否命中。空版本返回 false；Matcher 未初始化、版本无法解析或比较失败时
// 会打印日志并返回错误。
func (m *Matcher) Match(version string) (bool, error) {
	matched, err := m.match(version)
	if err != nil {
		return false, logFailure("Matcher.Match", err)
	}
	return matched, nil
}

func (m *Matcher) match(version string) (bool, error) {
	if m == nil || (len(m.intervals) == 0 && !m.any) {
		return false, fmt.Errorf("%w: uninitialized matcher", ErrInvalidExpression)
	}
	if strings.TrimSpace(version) == "" {
		return false, nil
	}
	candidate, err := versioncore.Parse(version, m.kind)
	if err != nil {
		return false, translateVersionError(err)
	}
	if m.any {
		return true, nil
	}
	for _, interval := range m.intervals {
		matched, matchErr := m.matchInterval(candidate, interval)
		if matchErr != nil {
			return false, matchErr
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func compileInterval(interval rangeexpr.Interval, kind versioncore.Kind) (compiledInterval, error) {
	if !interval.Lower.Set && !interval.Upper.Set {
		return compiledInterval{}, fmt.Errorf("%w: interval cannot have two open ends", ErrInvalidExpression)
	}
	result := compiledInterval{}
	var err error
	if interval.Lower.Set {
		result.lower, err = compileBound(interval.Lower, kind)
		if err != nil {
			return compiledInterval{}, err
		}
	}
	if interval.Upper.Set {
		result.upper, err = compileBound(interval.Upper, kind)
		if err != nil {
			return compiledInterval{}, err
		}
	}
	if result.lower.set && result.upper.set {
		comparison, compareErr := versioncore.Compare(result.lower.version, result.upper.version, false)
		if compareErr != nil {
			return compiledInterval{}, translateVersionError(compareErr)
		}
		if comparison > 0 {
			return compiledInterval{}, fmt.Errorf("%w: lower bound is greater than upper bound", ErrInvalidExpression)
		}
		result.exact = comparison == 0 && result.lower.inclusive && result.upper.inclusive

		// [6.7-rc1,6.7) 明确描述同一主版本的预发布窗口，干净端点在这种场景下
		// 必须严格比较后缀；其他干净端点继续保留历史的主版本匹配语义。
		mainComparison, mainErr := versioncore.Compare(result.lower.version, result.upper.version, true)
		if mainErr == nil && mainComparison == 0 {
			if result.lower.version.HasSuffix() != result.upper.version.HasSuffix() {
				result.lower.autoMatchMain = false
				result.upper.autoMatchMain = false
			}
		}
	}
	return result, nil
}

func compileBound(bound rangeexpr.Bound, kind versioncore.Kind) (compiledBound, error) {
	version, err := versioncore.Parse(bound.Version, kind)
	if err != nil {
		return compiledBound{}, translateVersionError(err)
	}
	return compiledBound{
		version:       version,
		inclusive:     bound.Inclusive,
		set:           true,
		autoMatchMain: !version.HasSuffix(),
	}, nil
}

func (m *Matcher) matchInterval(candidate versioncore.Version, interval compiledInterval) (bool, error) {
	if interval.exact {
		comparison, err := versioncore.Compare(candidate, interval.lower.version, m.boundMatchMain(interval.lower))
		if errors.Is(err, versioncore.ErrIncomparable) {
			return false, nil
		}
		return comparison == 0, translateVersionError(err)
	}

	if interval.lower.set {
		comparison, err := versioncore.Compare(candidate, interval.lower.version, m.boundMatchMain(interval.lower))
		if err != nil {
			return false, translateVersionError(err)
		}
		if comparison < 0 || (comparison == 0 && !interval.lower.inclusive) {
			return false, nil
		}
	}

	upper, err := m.effectiveUpper(candidate, interval.upper)
	if err != nil {
		return false, err
	}
	if upper.set {
		comparison, compareErr := versioncore.Compare(candidate, upper.version, m.boundMatchMain(upper))
		if compareErr != nil {
			return false, translateVersionError(compareErr)
		}
		if comparison > 0 || (comparison == 0 && !upper.inclusive) {
			return false, nil
		}
	}
	return true, nil
}

func (m *Matcher) effectiveUpper(candidate versioncore.Version, original compiledBound) (compiledBound, error) {
	fixedVersion, found := fixedForVersion(candidate, m.fixed)
	if !found {
		return original, nil
	}
	fixed := compiledBound{
		version:       fixedVersion,
		inclusive:     false,
		set:           true,
		autoMatchMain: !fixedVersion.HasSuffix(),
	}
	if !original.set {
		return fixed, nil
	}
	comparison, err := versioncore.Compare(fixed.version, original.version, false)
	if err != nil {
		return compiledBound{}, translateVersionError(err)
	}
	// Fixed 只允许收紧范围；修复版本更晚时保留原漏洞上界，避免扩大影响面。
	if comparison <= 0 {
		return fixed, nil
	}
	return original, nil
}

func (m *Matcher) boundMatchMain(bound compiledBound) bool {
	return m.matchMain || bound.autoMatchMain
}
