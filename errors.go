package versioncompare

import "errors"

var (
	// ErrEmptyVersion 表示待解析或匹配的版本为空。
	ErrEmptyVersion = errors.New("empty version")
	// ErrInvalidVersion 表示版本不符合所选版本类型的格式。
	ErrInvalidVersion = errors.New("invalid version")
	// ErrInvalidExpression 表示漏洞版本范围表达式不合法。
	ErrInvalidExpression = errors.New("invalid version expression")
	// ErrUnsupportedVersionType 表示 VersionExp.Type 不是受支持的版本类型。
	ErrUnsupportedVersionType = errors.New("unsupported version type")
	// ErrEmptyVersionList 表示聚合操作收到空版本列表。
	ErrEmptyVersionList = errors.New("empty version list")
	// ErrIncomparableVersion 表示两个不透明版本无法形成可靠的大小关系。
	ErrIncomparableVersion = errors.New("incomparable version")
)
