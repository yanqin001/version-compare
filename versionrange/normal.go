package versionrange

import (
	"fmt"
	"strings"

	versioncompare "github.com/yanqin001/version-compare"
	"github.com/yanqin001/version-compare/internal/versioncore"
)

// parseNormal 复用根包的通用解析，保留字母补丁和完整后缀，避免复制一套发布标签白名单。
// value 已经过相同的边界/候选包装规范化；完整提交、不透明命名或范围控制字符不能当作普通发布版本。
func parseNormal(value string) (versioncore.Version, error) {
	if fullCommit.MatchString(strings.TrimSpace(value)) {
		return versioncore.Version{}, fmt.Errorf("%w: 完整提交 ID 不参与发布版本比较", ErrUnordered)
	}
	if strings.ContainsAny(value, "*<>=|,[]") {
		return versioncore.Version{}, fmt.Errorf("%w: 普通版本不能包含范围运算符或通配符: %q", versioncompare.ErrInvalidVersion, value)
	}
	parsed, err := versioncore.Parse(value, versioncore.Normal)
	if err != nil {
		return versioncore.Version{}, fmt.Errorf("%w: %v", versioncompare.ErrInvalidVersion, err)
	}
	if parsed.IsOpaque() {
		return versioncore.Version{}, fmt.Errorf("%w: normal 无法确定命名版本的顺序: %q", versioncompare.ErrInvalidVersion, value)
	}
	return parsed, nil
}
