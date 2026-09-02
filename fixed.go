package versioncompare

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yanqin001/version-compare/internal/versioncore"
)

type compiledFixed struct {
	branch  versioncore.Version
	version versioncore.Version
}

func parseLegacyFixed(values []string, kind VersionKind) ([]FixedVersion, error) {
	if len(values) == 0 {
		return nil, nil
	}
	coreVersionKind, err := coreKind(kind)
	if err != nil {
		return nil, err
	}
	result := make([]FixedVersion, 0, len(values))
	for _, rawValue := range values {
		value := strings.TrimSpace(rawValue)
		if value == "" {
			return nil, fmt.Errorf("%w: empty fixed version", ErrInvalidExpression)
		}

		branch := ""
		fixed := value
		if separator := strings.IndexByte(value, ':'); separator >= 0 {
			branch = strings.TrimSpace(value[:separator])
			fixed = strings.TrimSpace(value[separator+1:])
			if branch == "" || fixed == "" {
				return nil, fmt.Errorf("%w: invalid fixed mapping %q", ErrInvalidExpression, rawValue)
			}
		}

		parsedFixed, parseErr := versioncore.Parse(fixed, coreVersionKind)
		if parseErr != nil {
			return nil, translateVersionError(parseErr)
		}
		if branch == "" {
			segments := parsedFixed.MainSegments()
			if len(segments) >= 3 {
				segments = segments[:len(segments)-1]
			}
			branch = strings.Join(segments, ".")
		}
		result = append(result, FixedVersion{Branch: branch, Version: fixed})
	}
	return result, nil
}

func compileFixedVersions(values []FixedVersion, kind versioncore.Kind) ([]compiledFixed, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make([]compiledFixed, 0, len(values))
	branches := make(map[string]struct{}, len(values))
	for _, value := range values {
		branchText := strings.TrimSpace(value.Branch)
		versionText := strings.TrimSpace(value.Version)
		if branchText == "" || versionText == "" {
			return nil, fmt.Errorf("%w: fixed branch and version are required", ErrInvalidExpression)
		}
		branch, err := versioncore.Parse(branchText, kind)
		if err != nil {
			return nil, translateVersionError(err)
		}
		if branch.IsOpaque() || branch.HasSuffix() {
			return nil, fmt.Errorf("%w: invalid fixed branch %q", ErrInvalidExpression, value.Branch)
		}
		// 分支键保留数字段数量：1 与 1.0 分别表示主分支和更具体的维护分支，
		// 二者可以同时存在并由最长前缀规则决定优先级。
		branchKey := branch.MainString() + "\x00" + branch.Supplement()
		if _, exists := branches[branchKey]; exists {
			return nil, fmt.Errorf("%w: duplicate fixed branch %q", ErrInvalidExpression, value.Branch)
		}
		branches[branchKey] = struct{}{}

		fixedVersion, err := versioncore.Parse(versionText, kind)
		if err != nil {
			return nil, translateVersionError(err)
		}
		if fixedVersion.IsOpaque() {
			return nil, fmt.Errorf("%w: opaque fixed version %q", ErrInvalidExpression, value.Version)
		}
		if !versioncore.HasMainPrefix(fixedVersion, branch) {
			return nil, fmt.Errorf(
				"%w: fixed version %q does not belong to branch %q",
				ErrInvalidExpression,
				value.Version,
				value.Branch,
			)
		}
		result = append(result, compiledFixed{branch: branch, version: fixedVersion})
	}

	// 先匹配更具体的分支，消除旧 map 遍历造成的非确定性。
	sort.Slice(result, func(left, right int) bool {
		leftLength := len(result[left].branch.MainSegments())
		rightLength := len(result[right].branch.MainSegments())
		if leftLength != rightLength {
			return leftLength > rightLength
		}
		return result[left].branch.MainString() > result[right].branch.MainString()
	})
	return result, nil
}

func fixedForVersion(version versioncore.Version, fixed []compiledFixed) (versioncore.Version, bool) {
	for _, item := range fixed {
		if versioncore.HasMainPrefix(version, item.branch) {
			return item.version, true
		}
	}
	return versioncore.Version{}, false
}
