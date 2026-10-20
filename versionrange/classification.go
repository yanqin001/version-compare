// Package versionrange 提供显式分类的发布版本比较和结构化区间校验。
// 调用方声明版本体系；本包不推断来源，不解析 Git 祖先关系，不改动根包的历史表达式规则。
package versionrange

import (
	"errors"
	"fmt"
	"strings"

	versioncompare "github.com/yanqin001/version-compare"
)

const (
	// TypeVersion 表示可以按已声明版本体系排序的发布版本范围。
	TypeVersion = "VERSION"
	// TypeGit 表示依赖仓库引用或提交关系的范围，不能按字符串大小排序。
	TypeGit = "GIT"
	// TypeUnknown 表示没有足够信息确定比较语义，不能退回通用版本猜测。
	TypeUnknown = "UNKNOWN"
)

// ErrUnordered 与根包 ErrIncomparableVersion 相同，表示版本缺少可用的排序依据。
// 调用方不能将不可比较解释为未命中或安全。
var ErrUnordered = versioncompare.ErrIncomparableVersion

// ErrInvalidClassification 表示分类缺失、枚举无效或字段组合冲突，可通过 errors.Is 判断。
var ErrInvalidClassification = errors.New("invalid version classification")

// Classification 是每个范围携带的机器可读比较协议；所有字段为空表示历史数据。
// RangeType 为 VERSION/GIT/UNKNOWN，VersionScheme 指定 VERSION 的比较算法；枚举不合法返回错误。
// VersionPrefix/VersionSuffix 是明确声明的固定包装，边界与候选均必须满足。
// 自定义标签只能按 VersionOrder 中的显式顺序比较，列表之外的版本返回 ErrUnordered。
type Classification struct {
	// RangeType 声明可排序发布版本、Git 引用或未知语义，对应 TypeVersion/TypeGit/TypeUnknown。
	RangeType string `json:"range_type,omitempty"`
	// VersionScheme 选择 VERSION 的数字、生态或自定义比较算法；GIT/UNKNOWN 必须留空。
	VersionScheme string `json:"version_scheme,omitempty"`
	// RefKind 为 tag、commit、branch 或 alias；普通包版本留空。
	RefKind string `json:"ref_kind,omitempty"`
	// VersionPrefix 是比较前必须匹配并剥离的固定前缀，区分大小写。
	VersionPrefix string `json:"version_prefix,omitempty"`
	// VersionSuffix 是比较前必须匹配并剥离的固定后缀，区分大小写。
	VersionSuffix string `json:"version_suffix,omitempty"`
	// VersionOrder 仅供 custom 使用，列出从旧到新的完整原始标签；比较器会复制此切片。
	VersionOrder []string `json:"version_order,omitempty"`
}

// Present 判断是否声明过分类字段；部分缺失也返回 true，避免损坏元数据退回旧比较器。
func (c Classification) Present() bool {
	return c.RangeType != "" || c.VersionScheme != "" || c.RefKind != "" || c.VersionPrefix != "" || c.VersionSuffix != "" || len(c.VersionOrder) > 0
}

// Canonical 返回大小写规范化后的独立副本，不修改原始标签和包装字符串。
func (c Classification) Canonical() Classification {
	c.RangeType = strings.ToUpper(strings.TrimSpace(c.RangeType))
	c.VersionScheme = strings.ToLower(strings.TrimSpace(c.VersionScheme))
	c.RefKind = strings.ToLower(strings.TrimSpace(c.RefKind))
	c.VersionOrder = append([]string(nil), c.VersionOrder...)
	return c
}

// Validate 校验分类字段组合，不读取仓库或推测自定义标签顺序。
// 空分类仅供旧数据使用；不完整分类、未知枚举、重复排序项或冲突字段返回错误。
func (c Classification) Validate() error {
	if !c.Present() {
		return nil
	}
	c = c.Canonical()
	switch c.RefKind {
	case "", "tag", "commit", "branch", "alias":
	default:
		return fmt.Errorf("%w: 未知 ref_kind %q", ErrInvalidClassification, c.RefKind)
	}
	switch c.RangeType {
	case TypeVersion:
		if c.RefKind != "" && c.RefKind != "tag" {
			return fmt.Errorf("%w: VERSION 只能描述发布版本或 tag，不能比较 commit/branch/alias", ErrInvalidClassification)
		}
		switch c.VersionScheme {
		case "numeric", "semver", "pep440", "maven", "go", "npm", "cargo", "nuget", "rubygems", "composer", "calver", "dpkg", "rpm", "apk", "alpm", "custom":
		default:
			return fmt.Errorf("%w: 未知 version_scheme %q", ErrInvalidClassification, c.VersionScheme)
		}
	case TypeGit:
		if c.RefKind == "" {
			return fmt.Errorf("%w: GIT 必须声明 ref_kind", ErrInvalidClassification)
		}
		fallthrough
	case TypeUnknown:
		if c.VersionScheme != "" || c.VersionPrefix != "" || c.VersionSuffix != "" || len(c.VersionOrder) > 0 {
			return fmt.Errorf("%w: GIT/UNKNOWN 不能同时声明数字版本比较策略", ErrInvalidClassification)
		}
	default:
		return fmt.Errorf("%w: 未知 range_type %q", ErrInvalidClassification, c.RangeType)
	}
	if len(c.VersionOrder) > 0 && c.VersionScheme != "custom" {
		return fmt.Errorf("%w: version_order 仅用于 custom 版本体系", ErrInvalidClassification)
	}
	if c.VersionScheme == "custom" && (c.VersionPrefix != "" || c.VersionSuffix != "") {
		return fmt.Errorf("%w: custom 顺序必须使用完整原始标签，不能同时剥离包装", ErrInvalidClassification)
	}
	seen := make(map[string]bool, len(c.VersionOrder))
	for _, version := range c.VersionOrder {
		if version == "" || strings.TrimSpace(version) != version || seen[version] {
			return fmt.Errorf("%w: version_order 不能包含空白或重复标签", ErrInvalidClassification)
		}
		seen[version] = true
	}
	return nil
}
