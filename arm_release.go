package versioncompare

import (
	"fmt"
	"strings"

	"github.com/yanqin001/version-compare/internal/rangeexpr"
)

// normalizeArmInterval 按单个区间的端点识别 rMpN。仅基础发布号可以作为该策略的边界，
// 避免把含发布包后缀的边界或混合编号体系交给通用解析器推断排序。
func normalizeArmInterval(interval rangeexpr.Interval) (rangeexpr.Interval, bool, error) {
	lower, lowerSuffix, lowerArm := splitArmRelease(interval.Lower.Version)
	upper, upperSuffix, upperArm := splitArmRelease(interval.Upper.Version)
	if !lowerArm && !upperArm {
		return interval, false, nil
	}
	if (interval.Lower.Set && !lowerArm) || (interval.Upper.Set && !upperArm) {
		return interval, false, fmt.Errorf("%w: %q and %q use different version formats", ErrIncomparableVersion, interval.Lower.Version, interval.Upper.Version)
	}
	if lowerSuffix != "" || upperSuffix != "" {
		return interval, false, fmt.Errorf("%w: rMpN boundaries must omit package suffixes: %q and %q", ErrInvalidExpression, interval.Lower.Version, interval.Upper.Version)
	}
	interval.Lower.Version, interval.Upper.Version = lower, upper
	return interval, true, nil
}

// splitArmRelease 从完整标签读取两段数字，并单独返回显式后缀。
// 后缀由调用场景校验，路径和其他拼接文本不能被当作 rMpN 标签。
func splitArmRelease(value string) (base, suffix string, ok bool) {
	value = strings.TrimSpace(value)
	if len(value) < 4 || (value[0] != 'r' && value[0] != 'R') {
		return "", "", false
	}
	majorLength := digitPrefixLength(value[1:])
	separator := 1 + majorLength
	if majorLength == 0 || separator == len(value) || (value[separator] != 'p' && value[separator] != 'P') {
		return "", "", false
	}
	patch := value[separator+1:]
	patchLength := digitPrefixLength(patch)
	if patchLength == 0 {
		return "", "", false
	}
	suffix = patch[patchLength:]
	if suffix != "" && suffix[0] != '-' && suffix[0] != '+' {
		return "", "", false
	}
	return value[1:separator] + "." + patch[:patchLength], suffix, true
}

// isArmPackageSuffix 识别已知的 Arm 发布包后缀，例如 -01eac0 和 -01rel0。
// 基础发布号区间只比较 r 与 p 两段，不推断不同发布包后缀的先后。
func isArmPackageSuffix(suffix string) bool {
	if len(suffix) < 6 || suffix[0] != '-' {
		return false
	}
	build := suffix[1:]
	buildLength := digitPrefixLength(build)
	tail := build[buildLength:]
	if buildLength == 0 || len(tail) < 4 {
		return false
	}
	channel, revision := tail[:3], tail[3:]
	return (strings.EqualFold(channel, "eac") || strings.EqualFold(channel, "rel")) &&
		digitPrefixLength(revision) == len(revision)
}

// digitPrefixLength 返回开头连续 ASCII 数字的字节数，保留原文本以支持任意长度的版本号。
func digitPrefixLength(value string) int {
	index := 0
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	return index
}
