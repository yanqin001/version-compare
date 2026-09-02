package versioncompare

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/yanqin001/version-compare/internal/versioncore"
)

type versionSvc struct{}

var _ VersionSvc = (*versionSvc)(nil)

// NewVersionSvc 创建无状态、可并发安全使用的版本服务。
func NewVersionSvc() VersionSvc {
	return &versionSvc{}
}

// InExp 判断 VersionExp.Version 是否位于漏洞范围内。
// 返回值表示是否命中；表达式、固定版本或候选版本不合法时会打印日志并返回错误，
// 不执行任何持久化操作。
func (svc *versionSvc) InExp(versionExp VersionExp) (bool, error) {
	kind, err := kindFromType(versionExp.Type)
	if err != nil {
		return false, logFailure("InExp", err)
	}
	fixedVersions, err := parseLegacyFixed(versionExp.Fixed, kind)
	if err != nil {
		return false, logFailure("InExp", err)
	}
	matcher, err := compile(versionExp.Expression, CompileOptions{
		Kind:          kind,
		FixedVersions: fixedVersions,
		MatchMain:     versionExp.MatchMain,
	})
	if err != nil {
		return false, logFailure("InExp", err)
	}
	matched, err := matcher.match(versionExp.Version)
	if err != nil {
		return false, logFailure("InExp", err)
	}
	return matched, nil
}

// Compare 使用普通组件规则比较 v1 与 v2。
// 返回值固定为 0（小于）、1（等于）或 2（大于）；输入为空、格式无效或无法建立
// 可靠顺序时会打印日志并返回错误。
func (svc *versionSvc) Compare(v1, v2 string) (int, error) {
	left, err := versioncore.Parse(v1, versioncore.Normal)
	if err != nil {
		return 0, logFailure("Compare", translateVersionError(err))
	}
	right, err := versioncore.Parse(v2, versioncore.Normal)
	if err != nil {
		return 0, logFailure("Compare", translateVersionError(err))
	}
	result, err := versioncore.Compare(left, right, false)
	if err != nil {
		return 0, logFailure("Compare", translateVersionError(err))
	}
	return compatibilityResult(result), nil
}

func kindFromType(value string) (VersionKind, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", VersionTypeNormal, "component", "openssl":
		// openssl 仅作为迁移期输入别名，解析和比较不包含任何专用分支。
		return NormalVersion, nil
	case VersionTypeOS:
		return GenericOSVersion, nil
	default:
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedVersionType, value)
	}
}

func coreKind(kind VersionKind) (versioncore.Kind, error) {
	switch kind {
	case NormalVersion:
		return versioncore.Normal, nil
	case GenericOSVersion:
		return versioncore.GenericOS, nil
	default:
		return 0, fmt.Errorf("%w: kind %d", ErrUnsupportedVersionType, kind)
	}
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

func translateVersionError(err error) error {
	switch {
	case errors.Is(err, versioncore.ErrEmptyVersion):
		return fmt.Errorf("%w: %v", ErrEmptyVersion, err)
	case errors.Is(err, versioncore.ErrIncomparable):
		return fmt.Errorf("%w: %v", ErrIncomparableVersion, err)
	case errors.Is(err, versioncore.ErrInvalidVersion):
		return fmt.Errorf("%w: %v", ErrInvalidVersion, err)
	default:
		return err
	}
}

func logFailure(operation string, err error) error {
	log.Printf("versioncompare: %s failed: %v", operation, err)
	return err
}
