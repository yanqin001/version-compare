package osversion

import "strings"

// alpmVersion 保存 Arch Linux 的 Epoch、pkgver 和可选 pkgrel 视图。
type alpmVersion struct {
	epoch      string
	version    string
	release    string
	hasRelease bool
}

// compareALPM 按 libalpm 规则比较完整 Arch Linux 包版本。
func compareALPM(left, right string) (int, error) {
	leftVersion, err := parseALPM(left)
	if err != nil {
		return 0, err
	}
	rightVersion, err := parseALPM(right)
	if err != nil {
		return 0, err
	}

	return compareParsedALPM(leftVersion, rightVersion), nil
}

// compareParsedALPM 比较两个已完成格式校验的 ALPM 版本。
func compareParsedALPM(leftVersion, rightVersion alpmVersion) int {
	if result := compareNumericStrings(leftVersion.epoch, rightVersion.epoch); result != 0 {
		return result
	}
	if result := compareALPMPart(leftVersion.version, rightVersion.version); result != 0 {
		return result
	}
	// libalpm 将单侧缺失的 pkgrel 视为未约束；只有两侧均提供时才继续比较。
	if leftVersion.hasRelease && rightVersion.hasRelease {
		return compareALPMPart(leftVersion.release, rightVersion.release)
	}
	return 0
}

// parseALPM 解析 [epoch:]pkgver[-pkgrel] 并校验 PKGBUILD 对字段字符的约束。
func parseALPM(input string) (alpmVersion, error) {
	if input == "" {
		return alpmVersion{}, invalidNativeVersion("ALPM", input, 0, "version is empty")
	}

	result := alpmVersion{epoch: "0"}
	remainder := input
	if colon := strings.IndexByte(remainder, ':'); colon >= 0 {
		if colon == 0 {
			return alpmVersion{}, invalidNativeVersion("ALPM", input, 0, "epoch is empty")
		}
		for index := 0; index < colon; index++ {
			if !isASCIIDigit(remainder[index]) {
				return alpmVersion{}, invalidNativeVersion("ALPM", input, index, "epoch must be decimal digits")
			}
		}
		result.epoch = remainder[:colon]
		remainder = remainder[colon+1:]
		if remainder == "" {
			return alpmVersion{}, invalidNativeVersion("ALPM", input, colon+1, "pkgver is empty")
		}
	}

	if separator := strings.LastIndexByte(remainder, '-'); separator >= 0 {
		result.version = remainder[:separator]
		result.release = remainder[separator+1:]
		result.hasRelease = true
	} else {
		result.version = remainder
	}

	versionOffset := len(input) - len(remainder)
	if err := validateALPMVersion(input, result.version, versionOffset); err != nil {
		return alpmVersion{}, err
	}
	if result.hasRelease {
		releaseOffset := versionOffset + len(result.version) + 1
		if err := validateALPMRelease(input, result.release, releaseOffset); err != nil {
			return alpmVersion{}, err
		}
	}
	return result, nil
}

// validateALPMVersion 校验 pkgver；允许的非字母数字 ASCII 字符会作为分隔符，
// 但 ALPM 规范禁止空白、斜杠、冒号、连字符和比较运算符。
func validateALPMVersion(input, version string, offset int) error {
	if version == "" {
		return invalidNativeVersion("ALPM", input, offset, "pkgver is empty")
	}
	for index := 0; index < len(version); index++ {
		character := version[index]
		if isASCIIAlphanumeric(character) {
			continue
		}
		if character < 0x21 || character > 0x7e ||
			character == '/' || character == ':' || character == '-' ||
			character == '<' || character == '>' || character == '=' {
			return invalidNativeVersion("ALPM", input, offset+index, "unsupported character in pkgver")
		}
	}
	return nil
}

// validateALPMRelease 校验正整数及其可选点分 subrelease。
func validateALPMRelease(input, release string, offset int) error {
	if release == "" {
		return invalidNativeVersion("ALPM", input, offset, "pkgrel is empty")
	}
	segmentStart := 0
	separators := 0
	for index := 0; index <= len(release); index++ {
		if index < len(release) && isASCIIDigit(release[index]) {
			continue
		}
		if index == segmentStart {
			return invalidNativeVersion("ALPM", input, offset+index, "pkgrel contains an empty numeric segment")
		}
		if index == len(release) {
			break
		}
		if release[index] != '.' {
			return invalidNativeVersion("ALPM", input, offset+index, "pkgrel must contain decimal segments")
		}
		separators++
		if separators > 1 {
			return invalidNativeVersion("ALPM", input, offset+index, "pkgrel can contain at most two numeric segments")
		}
		segmentStart = index + 1
	}
	return nil
}

// compareALPMPart 实现 libalpm 的 alpha/numeric 分段排序。
// 分隔符本身不区分类别，但同一位置连续分隔符的数量会影响顺序。
func compareALPMPart(left, right string) int {
	if left == right {
		return 0
	}

	leftPosition := 0
	rightPosition := 0
	leftSegmentEnd := 0
	rightSegmentEnd := 0
	for leftPosition < len(left) && rightPosition < len(right) {
		for leftPosition < len(left) && !isASCIIAlphanumeric(left[leftPosition]) {
			leftPosition++
		}
		for rightPosition < len(right) && !isASCIIAlphanumeric(right[rightPosition]) {
			rightPosition++
		}
		if leftPosition == len(left) || rightPosition == len(right) {
			break
		}

		leftSeparators := leftPosition - leftSegmentEnd
		rightSeparators := rightPosition - rightSegmentEnd
		if leftSeparators < rightSeparators {
			return -1
		}
		if leftSeparators > rightSeparators {
			return 1
		}

		leftNumeric := isASCIIDigit(left[leftPosition])
		rightNumeric := isASCIIDigit(right[rightPosition])
		if leftNumeric != rightNumeric {
			if leftNumeric {
				return 1
			}
			return -1
		}

		leftSegmentEnd = leftPosition
		rightSegmentEnd = rightPosition
		if leftNumeric {
			for leftSegmentEnd < len(left) && isASCIIDigit(left[leftSegmentEnd]) {
				leftSegmentEnd++
			}
			for rightSegmentEnd < len(right) && isASCIIDigit(right[rightSegmentEnd]) {
				rightSegmentEnd++
			}
			if result := compareNumericStrings(left[leftPosition:leftSegmentEnd], right[rightPosition:rightSegmentEnd]); result != 0 {
				return result
			}
		} else {
			for leftSegmentEnd < len(left) && isASCIIAlpha(left[leftSegmentEnd]) {
				leftSegmentEnd++
			}
			for rightSegmentEnd < len(right) && isASCIIAlpha(right[rightSegmentEnd]) {
				rightSegmentEnd++
			}
			if result := compareBytes(left[leftPosition:leftSegmentEnd], right[rightPosition:rightSegmentEnd]); result != 0 {
				return result
			}
		}
		leftPosition = leftSegmentEnd
		rightPosition = rightSegmentEnd
	}

	if leftPosition == len(left) && rightPosition == len(right) {
		return 0
	}
	// libalpm 保留了 RPM 4.8.1 的尾部规则：附着字母段早于空串，
	// 而额外数字段或分隔段晚于空串。
	leftEnded := leftPosition == len(left)
	rightCharacterIsAlpha := rightPosition < len(right) && isASCIIAlpha(right[rightPosition])
	leftCharacterIsAlpha := leftPosition < len(left) && isASCIIAlpha(left[leftPosition])
	if leftEnded && !rightCharacterIsAlpha || leftCharacterIsAlpha {
		return -1
	}
	return 1
}
