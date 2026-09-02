package osversion

import (
	"fmt"
	"log"
	"strings"
)

type service struct{}

var _ Service = (*service)(nil)

// NewService 创建无状态、可并发使用的 OS 软件包版本服务。
func NewService() Service {
	return &service{}
}

// InExp 判断候选软件包是否同时满足发行版、OS 版本、架构和包版本范围。
// 返回值表示是否命中；约束或候选格式无效时会打印日志并返回错误。
func (svc *service) InExp(versionExp VersionExp) (bool, error) {
	matcher, err := compile(versionExp.Constraint)
	if err != nil {
		return false, logFailure("InExp", err)
	}
	matched, err := matcher.match(versionExp.Package)
	if err != nil {
		return false, logFailure("InExp", err)
	}
	return matched, nil
}

// Compare 按 distribution 对应的原生包管理器规则比较 v1 与 v2。
// 返回值固定为 0（小于）、1（等于）或 2（大于）。未知发行版回退通用算法；
// 输入为空或不符合对应原生格式时会打印日志并返回错误。
func (svc *service) Compare(distribution Distribution, v1, v2 string) (int, error) {
	left, _, _ := splitEmbeddedMetadata(v1)
	right, _, _ := splitEmbeddedMetadata(v2)
	if strings.TrimSpace(left) == "" {
		return 0, logFailure("Compare", fmt.Errorf("%w: v1", ErrEmptyVersion))
	}
	if strings.TrimSpace(right) == "" {
		return 0, logFailure("Compare", fmt.Errorf("%w: v2", ErrEmptyVersion))
	}
	result, err := compareNative(string(distribution), left, right)
	if err != nil {
		return 0, logFailure("Compare", fmt.Errorf("%w: %v", ErrInvalidVersion, err))
	}
	return compatibilityResult(result), nil
}

// MatchOSVersion 判断 OS 发行版本是否满足 expression。
// 返回值表示是否命中。expression 为空表示不限制，未知发行版使用通用比较规则；
// 非空约束或候选版本无效时会打印日志并返回错误。
func (svc *service) MatchOSVersion(distribution Distribution, version, expression string) (bool, error) {
	matcher, err := compileNativeExpression(string(distribution), expression, true)
	if err != nil {
		return false, logFailure("MatchOSVersion", err)
	}
	if matcher == nil {
		return true, nil
	}
	matched, err := matcher.match(normalizeOSRelease(string(distribution), version))
	if err != nil {
		return false, logFailure("MatchOSVersion", err)
	}
	return matched, nil
}

// MatchArchitecture 判断 actual 是否位于 allowed 架构集合中。
// 返回值表示是否命中；allowed 包含歧义或空白约束时会打印日志并返回错误。
func (svc *service) MatchArchitecture(actual string, allowed []string) (bool, error) {
	matched, err := MatchArchitecture(actual, allowed)
	if err != nil {
		return false, logFailure("MatchArchitecture", err)
	}
	return matched, nil
}

func compatibilityResult(result int) int {
	switch {
	case result < 0:
		return int(CompareLess)
	case result > 0:
		return int(CompareGreater)
	default:
		return int(CompareEqual)
	}
}

func logFailure(operation string, err error) error {
	log.Printf("versioncompare/osversion: %s failed: %v", operation, err)
	return err
}
