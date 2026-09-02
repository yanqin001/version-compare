package osversion

import "errors"

var (
	// ErrEmptyVersion 表示候选软件包版本为空。
	ErrEmptyVersion = errors.New("empty OS package version")
	// ErrInvalidVersion 表示软件包版本不符合所选发行版的原生格式。
	ErrInvalidVersion = errors.New("invalid OS package version")
	// ErrInvalidExpression 表示软件包或 OS 版本范围表达式不合法。
	ErrInvalidExpression = errors.New("invalid OS version expression")
)
