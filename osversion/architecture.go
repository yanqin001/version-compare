package osversion

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidArchitecture 表示架构约束为空或包含互相冲突的规则。
var ErrInvalidArchitecture = errors.New("invalid architecture constraint")

// ArchitectureMatcher 是编译后的架构匹配器。
//
// 匹配器会在编译时复制并规范化全部约束，之后不再修改内部状态，因此可以被多个
// goroutine 并发调用。ArchitectureMatcher 的零值不匹配任何架构，应通过
// CompileArchitectures 创建实例。
type ArchitectureMatcher struct {
	unrestricted bool
	allowed      map[string]struct{}
}

// CompileArchitectures 编译允许的架构列表。
//
// allowed 为 nil 或空切片时表示不限制架构；"*" 和 "any" 也表示不限制，但不能与
// 具体架构混用。函数会复制并规范化输入，不会保留调用方切片的引用。
//
// 返回值是可并发复用的只读匹配器。若列表包含空白项，或通配约束与具体架构混用，
// 返回包装 ErrInvalidArchitecture 的错误。
func CompileArchitectures(allowed []string) (*ArchitectureMatcher, error) {
	if len(allowed) == 0 {
		return &ArchitectureMatcher{unrestricted: true}, nil
	}

	normalized := make(map[string]struct{}, len(allowed))
	hasWildcard := false
	hasConcrete := false
	for i, architecture := range allowed {
		value := normalizeArchitecture(architecture)
		if value == "" {
			return nil, fmt.Errorf("%w: item %d is blank", ErrInvalidArchitecture, i)
		}

		if isWildcardArchitecture(value) {
			hasWildcard = true
			continue
		}

		hasConcrete = true
		normalized[value] = struct{}{}
	}

	// 通配约束与具体约束混用通常意味着上游数据存在歧义，编译期拒绝可避免静默扩大匹配范围。
	if hasWildcard && hasConcrete {
		return nil, fmt.Errorf("%w: wildcard cannot be combined with concrete architectures", ErrInvalidArchitecture)
	}
	if hasWildcard {
		return &ArchitectureMatcher{unrestricted: true}, nil
	}

	return &ArchitectureMatcher{allowed: normalized}, nil
}

// Match 判断 actual 是否满足已编译的架构约束。
//
// actual 会先执行大小写、空白、前导斜杠和标准别名规范化。为兼容 Anolis 源码包，
// 空架构候选按 "src" 处理。nil 接收者和 ArchitectureMatcher 零值均返回 false。
func (m *ArchitectureMatcher) Match(actual string) bool {
	if m == nil {
		return false
	}
	if m.unrestricted {
		return true
	}

	value := normalizeArchitecture(actual)
	if value == "" {
		// Anolis 的源码包版本通常不携带 "/src" 后缀，空候选架构需要与 src/source 约束匹配。
		value = "src"
	}
	_, ok := m.allowed[value]
	return ok
}

// MatchArchitecture 判断 actual 是否满足 allowed 指定的架构约束。
//
// allowed 的含义与 CompileArchitectures 相同。返回值表示是否匹配；约束无效时返回
// 包装 ErrInvalidArchitecture 的错误。需要重复匹配时应直接复用 ArchitectureMatcher，
// 以避免反复编译约束。
func MatchArchitecture(actual string, allowed []string) (bool, error) {
	matcher, err := CompileArchitectures(allowed)
	if err != nil {
		return false, err
	}
	return matcher.Match(actual), nil
}

// normalizeArchitecture 将外部架构名称转换为稳定的内部名称。
func normalizeArchitecture(architecture string) string {
	value := strings.ToLower(strings.TrimSpace(architecture))
	value = strings.TrimLeft(value, "/")
	value = strings.TrimSpace(value)

	switch value {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "ppc64el", "ppc64le":
		return "ppc64le"
	case "noarch", "all":
		// noarch/all 是同一类架构无关软件包，但不会被当作匹配所有约束的通配符。
		return "noarch"
	case "src", "source":
		return "src"
	default:
		return value
	}
}

// isWildcardArchitecture 判断规范化后的约束是否表示不限制架构。
func isWildcardArchitecture(architecture string) bool {
	return architecture == "*" || architecture == "any"
}
