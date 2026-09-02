package osversion

import "strings"

// debianVersion 保存 dpkg 的 Epoch、upstream version 和 Debian revision 视图。
type debianVersion struct {
	epoch    string
	upstream string
	revision string
}

// compareDebian 按 dpkg 规则比较两个完整 Debian 版本。
func compareDebian(left, right string) (int, error) {
	leftVersion, err := parseDebian(left)
	if err != nil {
		return 0, err
	}
	rightVersion, err := parseDebian(right)
	if err != nil {
		return 0, err
	}

	return compareParsedDebian(leftVersion, rightVersion), nil
}

// compareParsedDebian 比较两个已完成格式校验的 Debian 版本。
func compareParsedDebian(leftVersion, rightVersion debianVersion) int {
	if result := compareNumericStrings(leftVersion.epoch, rightVersion.epoch); result != 0 {
		return result
	}
	if result := compareDebianPart(leftVersion.upstream, rightVersion.upstream); result != 0 {
		return result
	}
	return compareDebianPart(leftVersion.revision, rightVersion.revision)
}

// parseDebian 解析 [epoch:]upstream_version[-debian_revision]。
// 最后一个连字符分隔 revision；缺失 revision 在比较时按 0 处理，而不是通配符。
func parseDebian(input string) (debianVersion, error) {
	if input == "" {
		return debianVersion{}, invalidNativeVersion("Debian", input, 0, "version is empty")
	}

	result := debianVersion{epoch: "0", revision: "0"}
	remainder := input
	if colon := strings.IndexByte(input, ':'); colon >= 0 {
		if colon == 0 {
			return debianVersion{}, invalidNativeVersion("Debian", input, 0, "epoch is empty")
		}
		for index := 0; index < colon; index++ {
			if !isASCIIDigit(input[index]) {
				return debianVersion{}, invalidNativeVersion("Debian", input, index, "epoch must be decimal digits")
			}
		}
		result.epoch = input[:colon]
		remainder = input[colon+1:]
		if remainder == "" {
			return debianVersion{}, invalidNativeVersion("Debian", input, colon+1, "upstream version is empty")
		}
	}

	hasRevision := false
	if separator := strings.LastIndexByte(remainder, '-'); separator >= 0 {
		result.upstream = remainder[:separator]
		result.revision = remainder[separator+1:]
		hasRevision = true
	} else {
		result.upstream = remainder
	}

	upstreamOffset := len(input) - len(remainder)
	if err := validateDebianUpstream(input, result.upstream, upstreamOffset); err != nil {
		return debianVersion{}, err
	}
	if hasRevision {
		revisionOffset := upstreamOffset + len(result.upstream) + 1
		if err := validateDebianRevision(input, result.revision, revisionOffset); err != nil {
			return debianVersion{}, err
		}
	}
	return result, nil
}

// validateDebianUpstream 校验 upstream version 的字符集合和首字符约束。
func validateDebianUpstream(input, upstream string, offset int) error {
	if upstream == "" {
		return invalidNativeVersion("Debian", input, offset, "upstream version is empty")
	}
	if !isASCIIDigit(upstream[0]) {
		return invalidNativeVersion("Debian", input, offset, "upstream version must start with a digit")
	}
	for index := 0; index < len(upstream); index++ {
		character := upstream[index]
		if isASCIIAlphanumeric(character) {
			continue
		}
		switch character {
		case '.', '+', '-', ':', '~':
			continue
		default:
			return invalidNativeVersion("Debian", input, offset+index, "unsupported character in upstream version")
		}
	}
	return nil
}

// validateDebianRevision 校验 Debian revision；其中连字符不合法，因为最后一个连字符已经用于分隔字段。
func validateDebianRevision(input, revision string, offset int) error {
	if revision == "" {
		return invalidNativeVersion("Debian", input, offset, "Debian revision is empty")
	}
	for index := 0; index < len(revision); index++ {
		character := revision[index]
		if isASCIIAlphanumeric(character) {
			continue
		}
		switch character {
		case '.', '+', '~':
			continue
		default:
			return invalidNativeVersion("Debian", input, offset+index, "unsupported character in Debian revision")
		}
	}
	return nil
}

// compareDebianPart 比较 upstream version 或 Debian revision。
// 非数字字段使用 dpkg 的特殊字符权重，数字字段按任意精度整数比较。
func compareDebianPart(left, right string) int {
	leftPosition := 0
	rightPosition := 0
	for leftPosition < len(left) || rightPosition < len(right) {
		for leftPosition < len(left) && !isASCIIDigit(left[leftPosition]) ||
			rightPosition < len(right) && !isASCIIDigit(right[rightPosition]) {
			leftOrder := debianCharacterOrder(left, leftPosition)
			rightOrder := debianCharacterOrder(right, rightPosition)
			if leftOrder < rightOrder {
				return -1
			}
			if leftOrder > rightOrder {
				return 1
			}
			if leftPosition < len(left) {
				leftPosition++
			}
			if rightPosition < len(right) {
				rightPosition++
			}
		}

		leftZeroEnd := leftPosition
		for leftZeroEnd < len(left) && left[leftZeroEnd] == '0' {
			leftZeroEnd++
		}
		rightZeroEnd := rightPosition
		for rightZeroEnd < len(right) && right[rightZeroEnd] == '0' {
			rightZeroEnd++
		}

		leftEnd := leftZeroEnd
		for leftEnd < len(left) && isASCIIDigit(left[leftEnd]) {
			leftEnd++
		}
		rightEnd := rightZeroEnd
		for rightEnd < len(right) && isASCIIDigit(right[rightEnd]) {
			rightEnd++
		}
		if leftEnd-leftZeroEnd < rightEnd-rightZeroEnd {
			return -1
		}
		if leftEnd-leftZeroEnd > rightEnd-rightZeroEnd {
			return 1
		}
		if result := compareBytes(left[leftZeroEnd:leftEnd], right[rightZeroEnd:rightEnd]); result != 0 {
			return result
		}

		// 必须越过整个数字字段，包括前导零；否则全零字段会被重复扫描。
		leftPosition = leftEnd
		for leftPosition < len(left) && isASCIIDigit(left[leftPosition]) {
			leftPosition++
		}
		rightPosition = rightEnd
		for rightPosition < len(right) && isASCIIDigit(right[rightPosition]) {
			rightPosition++
		}
	}
	return 0
}

// debianCharacterOrder 返回 dpkg 非数字阶段使用的字符权重。
// 数字和字符串结束的权重均为 0，~ 为 -1，字母保持 ASCII 权重，其他字符排在字母之后。
func debianCharacterOrder(value string, position int) int {
	if position >= len(value) {
		return 0
	}
	character := value[position]
	if isASCIIDigit(character) {
		return 0
	}
	if character == '~' {
		return -1
	}
	if isASCIIAlpha(character) {
		return int(character)
	}
	return int(character) + 256
}
