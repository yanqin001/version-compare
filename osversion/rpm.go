package osversion

import "strings"

// rpmVersion 保存 RPM 的 Epoch、Version、Release 视图，字段直接引用原输入以避免复制。
type rpmVersion struct {
	epoch      string
	version    string
	release    string
	hasRelease bool
}

// compareRPM 按 RPM 的完整 EVR 顺序比较两个版本。
func compareRPM(left, right string) (int, error) {
	leftVersion, err := parseRPM(left)
	if err != nil {
		return 0, err
	}
	rightVersion, err := parseRPM(right)
	if err != nil {
		return 0, err
	}

	return compareParsedRPM(leftVersion, rightVersion), nil
}

// compareParsedRPM 比较两个已完成格式校验的 RPM EVR。
func compareParsedRPM(leftVersion, rightVersion rpmVersion) int {
	if result := compareNumericStrings(leftVersion.epoch, rightVersion.epoch); result != 0 {
		return result
	}
	if result := compareRPMPart(leftVersion.version, rightVersion.version); result != 0 {
		return result
	}
	if leftVersion.hasRelease != rightVersion.hasRelease {
		if leftVersion.hasRelease {
			return 1
		}
		return -1
	}
	if !leftVersion.hasRelease {
		return 0
	}
	return compareRPMPart(leftVersion.release, rightVersion.release)
}

// parseRPM 解析 [epoch:]version[-release]，同时拒绝 RPM 构建规则不允许的字符。
func parseRPM(input string) (rpmVersion, error) {
	if input == "" {
		return rpmVersion{}, invalidNativeVersion("RPM", input, 0, "version is empty")
	}

	result := rpmVersion{epoch: "0"}
	remainder := input
	if colon := strings.IndexByte(remainder, ':'); colon >= 0 {
		if colon == 0 {
			return rpmVersion{}, invalidNativeVersion("RPM", input, colon, "epoch is empty")
		}
		for index := 0; index < colon; index++ {
			if !isASCIIDigit(remainder[index]) {
				return rpmVersion{}, invalidNativeVersion("RPM", input, index, "epoch must be decimal digits")
			}
		}
		result.epoch = remainder[:colon]
		remainder = remainder[colon+1:]
		if remainder == "" {
			return rpmVersion{}, invalidNativeVersion("RPM", input, colon+1, "version is empty")
		}
	}

	if separator := strings.IndexByte(remainder, '-'); separator >= 0 {
		result.version = remainder[:separator]
		result.release = remainder[separator+1:]
		result.hasRelease = true
	} else {
		result.version = remainder
	}

	versionOffset := len(input) - len(remainder)
	if err := validateRPMPart(input, result.version, versionOffset, "version"); err != nil {
		return rpmVersion{}, err
	}
	if result.hasRelease {
		releaseOffset := versionOffset + len(result.version) + 1
		if err := validateRPMPart(input, result.release, releaseOffset, "release"); err != nil {
			return rpmVersion{}, err
		}
	}
	return result, nil
}

// validateRPMPart 校验单个 Version 或 Release；连字符由 EVR 解析层处理，不能出现在字段内部。
func validateRPMPart(input, part string, offset int, field string) error {
	if part == "" {
		return invalidNativeVersion("RPM", input, offset, field+" is empty")
	}

	hasAlphanumeric := false
	for index := 0; index < len(part); index++ {
		character := part[index]
		if isASCIIAlphanumeric(character) {
			hasAlphanumeric = true
			continue
		}
		switch character {
		case '.', '_', '+', '~', '^':
			continue
		default:
			return invalidNativeVersion("RPM", input, offset+index, "unsupported character in "+field)
		}
	}
	if !hasAlphanumeric {
		return invalidNativeVersion("RPM", input, offset, field+" must contain an alphanumeric segment")
	}
	return nil
}

// compareRPMPart 比较一个 Version 或 Release 字段。
// 普通分隔符被忽略，~ 表示前置版本，^ 表示正式版之后且下一个普通段之前的快照版本。
func compareRPMPart(left, right string) int {
	leftPosition := 0
	rightPosition := 0
	for {
		for leftPosition < len(left) && !isASCIIAlphanumeric(left[leftPosition]) && left[leftPosition] != '~' && left[leftPosition] != '^' {
			leftPosition++
		}
		for rightPosition < len(right) && !isASCIIAlphanumeric(right[rightPosition]) && right[rightPosition] != '~' && right[rightPosition] != '^' {
			rightPosition++
		}

		leftTilde := leftPosition < len(left) && left[leftPosition] == '~'
		rightTilde := rightPosition < len(right) && right[rightPosition] == '~'
		if leftTilde || rightTilde {
			if leftTilde && rightTilde {
				leftPosition++
				rightPosition++
				continue
			}
			if leftTilde {
				return -1
			}
			return 1
		}

		leftCaret := leftPosition < len(left) && left[leftPosition] == '^'
		rightCaret := rightPosition < len(right) && right[rightPosition] == '^'
		if leftCaret || rightCaret {
			if leftCaret && rightCaret {
				leftPosition++
				rightPosition++
				continue
			}
			// ^ 高于已经结束的基准版本，但低于同位置仍有普通字段的下一个版本。
			if leftCaret {
				if rightPosition == len(right) {
					return 1
				}
				return -1
			}
			if leftPosition == len(left) {
				return -1
			}
			return 1
		}

		if leftPosition == len(left) || rightPosition == len(right) {
			break
		}

		leftNumeric := isASCIIDigit(left[leftPosition])
		rightNumeric := isASCIIDigit(right[rightPosition])
		if leftNumeric != rightNumeric {
			if leftNumeric {
				return 1
			}
			return -1
		}

		leftEnd := leftPosition
		rightEnd := rightPosition
		if leftNumeric {
			for leftEnd < len(left) && isASCIIDigit(left[leftEnd]) {
				leftEnd++
			}
			for rightEnd < len(right) && isASCIIDigit(right[rightEnd]) {
				rightEnd++
			}
			if result := compareNumericStrings(left[leftPosition:leftEnd], right[rightPosition:rightEnd]); result != 0 {
				return result
			}
		} else {
			for leftEnd < len(left) && isASCIIAlpha(left[leftEnd]) {
				leftEnd++
			}
			for rightEnd < len(right) && isASCIIAlpha(right[rightEnd]) {
				rightEnd++
			}
			if result := compareBytes(left[leftPosition:leftEnd], right[rightPosition:rightEnd]); result != 0 {
				return result
			}
		}
		leftPosition = leftEnd
		rightPosition = rightEnd
	}

	if leftPosition < len(left) {
		return 1
	}
	if rightPosition < len(right) {
		return -1
	}
	return 0
}
