package osversion

import "github.com/yanqin001/version-compare/internal/versioncore"

// compareGeneric 使用兼容旧 OS 数据的宽松解析规则比较版本。
// 该模式会忽略架构斜杠后的内容，主版本采用点分任意精度数字比较，正式版高于同主版本后缀。
func compareGeneric(left, right string) (int, error) {
	leftVersion, err := versioncore.Parse(left, versioncore.GenericOS)
	if err != nil {
		return 0, err
	}
	rightVersion, err := versioncore.Parse(right, versioncore.GenericOS)
	if err != nil {
		return 0, err
	}
	return versioncore.Compare(leftVersion, rightVersion, false)
}
