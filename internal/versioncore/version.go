// Package versioncore 实现普通组件版本与兼容 OS 版本的无正则解析和比较。
package versioncore

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrEmptyVersion 表示输入版本为空。
	ErrEmptyVersion = errors.New("empty version")
	// ErrInvalidVersion 表示输入中不存在可识别的版本。
	ErrInvalidVersion = errors.New("invalid version")
	// ErrIncomparable 表示不透明版本之间没有可靠的顺序关系。
	ErrIncomparable = errors.New("incomparable version")
)

// Kind 指定内部解析规则。
type Kind uint8

const (
	// Normal 表示普通开源组件版本。
	Normal Kind = iota
	// GenericOS 表示兼容旧实现的通用 OS 版本。
	GenericOS
)

type token struct {
	value   string
	numeric bool
}

// Version 是已解析且可重复比较的不可变版本。
type Version struct {
	original   string
	normalized string
	main       []string
	supplement string
	suffix     []token
	opaque     bool
}

// Parse 将 input 解析为可比较版本。
// kind 控制连字符数字的解释方式；返回值不引用 input 之外的可变数据。
func Parse(input string, kind Kind) (Version, error) {
	if kind != Normal && kind != GenericOS {
		return Version{}, fmt.Errorf("%w: kind %d", ErrInvalidVersion, kind)
	}

	original := input
	normalized := normalize(input)
	if normalized == "" {
		return Version{}, ErrEmptyVersion
	}
	if isOpaqueVersion(normalized) {
		return Version{original: original, normalized: normalized, opaque: true}, nil
	}

	start := findVersionStart(normalized, kind)
	if start < 0 {
		return Version{}, fmt.Errorf("%w: %q", ErrInvalidVersion, input)
	}

	end := versionEnd(normalized, start)
	candidate := normalized[start:end]
	parsed, err := parseCandidate(candidate, kind)
	if err != nil {
		return Version{}, fmt.Errorf("%w: %q: %v", ErrInvalidVersion, input, err)
	}
	parsed.original = original
	parsed.normalized = candidate
	return parsed, nil
}

// Original 返回调用方传入的原始版本字符串。
func (v Version) Original() string {
	return v.original
}

// IsOpaque 报告版本是否只能进行精确相等判断。
func (v Version) IsOpaque() bool {
	return v.opaque
}

// HasSuffix 报告版本是否包含预发布或附加修订字段。
func (v Version) HasSuffix() bool {
	return len(v.suffix) != 0
}

// MainSegments 返回主版本数字段的副本。
func (v Version) MainSegments() []string {
	result := make([]string, len(v.main))
	copy(result, v.main)
	return result
}

// MainString 返回规范化后的点分主版本。
func (v Version) MainString() string {
	return strings.Join(v.main, ".")
}

// Supplement 返回直接跟在数字主版本后的字母补丁号。
func (v Version) Supplement() string {
	return v.supplement
}

// Compare 比较两个已解析版本，返回 -1、0 或 1。
// matchMain 为 true 时忽略预发布及附加修订，但仍比较字母补丁号。
func Compare(left, right Version, matchMain bool) (int, error) {
	if left.opaque || right.opaque {
		if left.opaque && right.opaque && left.normalized == right.normalized {
			return 0, nil
		}
		return 0, fmt.Errorf("%w: %q and %q", ErrIncomparable, left.original, right.original)
	}

	if result := compareMain(left.main, right.main); result != 0 {
		return result, nil
	}
	if result := strings.Compare(left.supplement, right.supplement); result != 0 {
		return sign(result), nil
	}
	if matchMain {
		return 0, nil
	}
	return compareSuffix(left.suffix, right.suffix), nil
}

// HasMainPrefix 报告 version 是否属于 branch 指定的数字维护分支。
func HasMainPrefix(version, branch Version) bool {
	if version.opaque || branch.opaque || len(branch.main) > len(version.main) {
		return false
	}
	for index := range branch.main {
		if compareNumber(version.main[index], branch.main[index]) != 0 {
			return false
		}
	}
	return branch.supplement == "" || version.supplement == branch.supplement
}

func parseCandidate(candidate string, kind Kind) (Version, error) {
	if candidate == "" || !isDigit(candidate[0]) {
		return Version{}, ErrInvalidVersion
	}

	main, position := parseMain(candidate, kind)
	if len(main) == 0 {
		return Version{}, ErrInvalidVersion
	}

	var supplement string
	if position < len(candidate) && isLetter(candidate[position]) {
		letterEnd := position + 1
		for letterEnd < len(candidate) && isLetter(candidate[letterEnd]) {
			letterEnd++
		}
		letters := candidate[position:letterEnd]
		// 多字母的常见预发布标识不属于 OpenSSL 式补丁字母。
		if !isKnownPrerelease(letters) {
			supplement = letters
			position = letterEnd
		}
	}

	suffix := tokenizeSuffix(candidate[position:])
	return Version{main: main, supplement: supplement, suffix: suffix}, nil
}

func parseMain(candidate string, kind Kind) ([]string, int) {
	position := 0
	main := make([]string, 0, 4)
	for {
		start := position
		for position < len(candidate) && isDigit(candidate[position]) {
			position++
		}
		if start == position {
			break
		}
		main = append(main, canonicalNumber(candidate[start:position]))
		if position >= len(candidate) || candidate[position] != '.' || position+1 >= len(candidate) || !isDigit(candidate[position+1]) {
			break
		}
		position++
	}

	// 普通版本兼容 VER-2-2-0 一类标签；OS 模式必须保留 '-' 后的 release。
	if kind == Normal && len(main) == 1 && position < len(candidate) && candidate[position] == '-' {
		probe := position
		extra := make([]string, 0, 3)
		for probe < len(candidate) && candidate[probe] == '-' && probe+1 < len(candidate) && isDigit(candidate[probe+1]) {
			probe++
			start := probe
			for probe < len(candidate) && isDigit(candidate[probe]) {
				probe++
			}
			extra = append(extra, canonicalNumber(candidate[start:probe]))
		}
		if len(extra) >= 2 {
			main = append(main, extra...)
			position = probe
		}
	}
	return main, position
}

func tokenizeSuffix(value string) []token {
	result := make([]token, 0, 4)
	for position := 0; position < len(value); {
		for position < len(value) && !isDigit(value[position]) && !isLetter(value[position]) {
			position++
		}
		if position >= len(value) {
			break
		}
		start := position
		numeric := isDigit(value[position])
		for position < len(value) && isDigit(value[position]) == numeric && (isDigit(value[position]) || isLetter(value[position])) {
			position++
		}
		part := value[start:position]
		if numeric {
			part = canonicalNumber(part)
		}
		result = append(result, token{value: part, numeric: numeric})
	}
	return result
}

func compareMain(left, right []string) int {
	length := len(left)
	if len(right) > length {
		length = len(right)
	}
	for index := 0; index < length; index++ {
		leftPart := "0"
		rightPart := "0"
		if index < len(left) {
			leftPart = left[index]
		}
		if index < len(right) {
			rightPart = right[index]
		}
		if result := compareNumber(leftPart, rightPart); result != 0 {
			return result
		}
	}
	return 0
}

func compareSuffix(left, right []token) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	// 普通组件遵循既有业务规则：正式版高于同主版本的任意后缀版本。
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}

	length := len(left)
	if len(right) < length {
		length = len(right)
	}
	for index := 0; index < length; index++ {
		leftToken := left[index]
		rightToken := right[index]
		if leftToken.numeric != rightToken.numeric {
			if leftToken.numeric {
				return -1
			}
			return 1
		}
		if leftToken.numeric {
			if result := compareNumber(leftToken.value, rightToken.value); result != 0 {
				return result
			}
			continue
		}
		if result := strings.Compare(leftToken.value, rightToken.value); result != 0 {
			return sign(result)
		}
	}
	// 兼容旧 Additional 语义：rc 高于 rc1，较长的附加修订更早。
	if len(left) < len(right) {
		return 1
	}
	if len(left) > len(right) {
		return -1
	}
	return 0
}

func compareNumber(left, right string) int {
	left = canonicalNumber(left)
	right = canonicalNumber(right)
	if len(left) < len(right) {
		return -1
	}
	if len(left) > len(right) {
		return 1
	}
	return sign(strings.Compare(left, right))
}

func canonicalNumber(value string) string {
	position := 0
	for position+1 < len(value) && value[position] == '0' {
		position++
	}
	return value[position:]
}

func findVersionStart(value string, kind Kind) int {
	bestIndex := -1
	bestScore := -1
	for index := 0; index < len(value); index++ {
		if !isDigit(value[index]) || (index > 0 && isDigit(value[index-1])) {
			continue
		}
		score := candidateScore(value, index, kind)
		if score <= 0 {
			continue
		}
		if score > bestScore {
			bestIndex = index
			bestScore = score
		}
	}
	return bestIndex
}

func candidateScore(value string, start int, kind Kind) int {
	position := start
	for position < len(value) && isDigit(value[position]) {
		position++
	}
	score := 1
	if position < len(value) && value[position] == '.' && position+1 < len(value) && isDigit(value[position+1]) {
		score = 4
	} else if position < len(value) && value[position] == '-' && position+1 < len(value) && isDigit(value[position+1]) {
		score = 3
	} else if position < len(value) && isLetter(value[position]) {
		score = 2
	}
	// 旧 generic OS 规则忽略 epoch。对于 1:2 这类没有点号的 EVR，冒号后的
	// 数字必须优先于 epoch 本身，否则会把主版本错误解析成 1。
	if kind == GenericOS && start > 0 && value[start-1] == ':' {
		score = 5
	}
	// 字母数字标识符中的孤立数字通常是包名的一部分，而不是版本。
	if start > 0 && isLetter(value[start-1]) && score < 3 {
		return 0
	}
	return score
}

func versionEnd(value string, start int) int {
	for position := start; position < len(value); position++ {
		switch value[position] {
		case '@', '#', '/', '\\', ' ', '\t', '\r', '\n':
			return position
		}
	}
	return len(value)
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "x86_64", "")
	value = strings.ReplaceAll(value, "~~", "-")
	value = strings.ReplaceAll(value, "~", "-")
	value = strings.ReplaceAll(value, "_", ".")
	value = strings.ReplaceAll(value, "+", "-")
	return value
}

func isOpaqueVersion(value string) bool {
	if len(value) == 40 {
		for index := range value {
			if !isDigit(value[index]) && !isLetter(value[index]) {
				return false
			}
		}
		return true
	}
	if value == "" || !isLetter(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !isLetter(value[index]) && !isDigit(value[index]) {
			return false
		}
	}
	return true
}

func isKnownPrerelease(value string) bool {
	switch value {
	case "alpha", "beta", "dev", "m", "milestone", "pre", "preview", "rc", "snapshot":
		return true
	default:
		return false
	}
}

func isDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func isLetter(value byte) bool {
	return value >= 'a' && value <= 'z'
}

func sign(value int) int {
	if value < 0 {
		return -1
	}
	if value > 0 {
		return 1
	}
	return 0
}
