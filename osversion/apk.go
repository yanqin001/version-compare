package osversion

// apkTokenKind 的顺序属于 APK 比较规则的一部分：当前缀相同且 token 类型不同时，
// 较晚的 token 类型通常更旧，预发布 suffix 另行处理。
type apkTokenKind uint8

const (
	apkTokenInitialDigit apkTokenKind = iota
	apkTokenDigit
	apkTokenLetter
	apkTokenSuffix
	apkTokenSuffixNumber
	apkTokenCommitHash
	apkTokenRevision
	apkTokenEnd
	apkTokenInvalid
)

const (
	apkSuffixAlpha = iota
	apkSuffixBeta
	apkSuffixPre
	apkSuffixRC
	apkSuffixNone
	apkSuffixCVS
	apkSuffixSVN
	apkSuffixGit
	apkSuffixHg
	apkSuffixPatch
)

// apkToken 是一个不复制原字符串的 APK 版本 token。
type apkToken struct {
	kind   apkTokenKind
	value  string
	suffix int
	offset int
}

// apkScanner 保存 APK 有限状态扫描器的位置和上一 token 类型。
type apkScanner struct {
	input    string
	position int
	previous apkTokenKind
	started  bool
}

// compareAPK 按 apk-tools 的 pkgver 规则比较两个 Alpine 版本。
func compareAPK(left, right string) (int, error) {
	if err := validateAPK(left); err != nil {
		return 0, err
	}
	if err := validateAPK(right); err != nil {
		return 0, err
	}
	return compareValidatedAPK(left, right), nil
}

// compareValidatedAPK 比较两个已经完整扫描并校验过的 APK 版本。
func compareValidatedAPK(left, right string) int {
	leftScanner := apkScanner{input: left}
	rightScanner := apkScanner{input: right}
	for {
		leftToken := leftScanner.next()
		rightToken := rightScanner.next()
		if leftToken.kind == rightToken.kind {
			if leftToken.kind == apkTokenEnd {
				return 0
			}
			if result := compareAPKToken(leftToken, rightToken); result != 0 {
				return result
			}
			continue
		}

		// _alpha、_beta、_pre、_rc 是唯一在字符串结束之前排序的扩展 token。
		if leftToken.kind == apkTokenSuffix && leftToken.suffix < apkSuffixNone {
			return -1
		}
		if rightToken.kind == apkTokenSuffix && rightToken.suffix < apkSuffixNone {
			return 1
		}
		if leftToken.kind > rightToken.kind {
			return -1
		}
		return 1
	}
}

// validateAPK 完整扫描输入，确保比较提前得出结果时也不会遗漏非法尾部。
func validateAPK(input string) error {
	if input == "" {
		return invalidNativeVersion("APK", input, 0, "version is empty")
	}

	scanner := apkScanner{input: input}
	for {
		token := scanner.next()
		switch token.kind {
		case apkTokenEnd:
			return nil
		case apkTokenInvalid:
			return invalidNativeVersion("APK", input, token.offset, "token does not follow APK version grammar")
		}
	}
}

// next 按 digit{.digit}...{letter}{_suffix{number}}...{~hash}{-rnumber} 推进一个 token。
func (scanner *apkScanner) next() apkToken {
	if !scanner.started {
		scanner.started = true
		start := scanner.position
		if start >= len(scanner.input) || !isASCIIDigit(scanner.input[start]) {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		for scanner.position < len(scanner.input) && isASCIIDigit(scanner.input[scanner.position]) {
			scanner.position++
		}
		scanner.previous = apkTokenInitialDigit
		return apkToken{kind: apkTokenInitialDigit, value: scanner.input[start:scanner.position], offset: start}
	}

	if scanner.position == len(scanner.input) {
		return apkToken{kind: apkTokenEnd, offset: scanner.position}
	}

	start := scanner.position
	character := scanner.input[start]
	switch {
	case character >= 'a' && character <= 'z':
		if scanner.previous > apkTokenDigit {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.position++
		scanner.previous = apkTokenLetter
		return apkToken{kind: apkTokenLetter, value: scanner.input[start:scanner.position], offset: start}
	case character == '.':
		if scanner.previous > apkTokenDigit {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.position++
		if scanner.position == len(scanner.input) || !isASCIIDigit(scanner.input[scanner.position]) {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		numberStart := scanner.position
		for scanner.position < len(scanner.input) && isASCIIDigit(scanner.input[scanner.position]) {
			scanner.position++
		}
		scanner.previous = apkTokenDigit
		return apkToken{kind: apkTokenDigit, value: scanner.input[numberStart:scanner.position], offset: start}
	case isASCIIDigit(character):
		kind := apkTokenInvalid
		switch scanner.previous {
		case apkTokenInitialDigit, apkTokenDigit:
			kind = apkTokenDigit
		case apkTokenSuffix:
			kind = apkTokenSuffixNumber
		}
		if kind == apkTokenInvalid {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		for scanner.position < len(scanner.input) && isASCIIDigit(scanner.input[scanner.position]) {
			scanner.position++
		}
		scanner.previous = kind
		return apkToken{kind: kind, value: scanner.input[start:scanner.position], offset: start}
	case character == '_':
		if scanner.previous > apkTokenSuffixNumber {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.position++
		suffixStart := scanner.position
		for scanner.position < len(scanner.input) && scanner.input[scanner.position] >= 'a' && scanner.input[scanner.position] <= 'z' {
			scanner.position++
		}
		suffix, ok := parseAPKSuffix(scanner.input[suffixStart:scanner.position])
		if !ok {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.previous = apkTokenSuffix
		return apkToken{kind: apkTokenSuffix, value: scanner.input[suffixStart:scanner.position], suffix: suffix, offset: start}
	case character == '~':
		if scanner.previous >= apkTokenCommitHash {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.position++
		hashStart := scanner.position
		for scanner.position < len(scanner.input) && isAPKHexDigit(scanner.input[scanner.position]) {
			scanner.position++
		}
		if hashStart == scanner.position {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.previous = apkTokenCommitHash
		return apkToken{kind: apkTokenCommitHash, value: scanner.input[hashStart:scanner.position], offset: start}
	case character == '-':
		if scanner.previous >= apkTokenRevision || start+2 > len(scanner.input) || scanner.input[start:start+2] != "-r" {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.position += 2
		numberStart := scanner.position
		for scanner.position < len(scanner.input) && isASCIIDigit(scanner.input[scanner.position]) {
			scanner.position++
		}
		if numberStart == scanner.position {
			return apkToken{kind: apkTokenInvalid, offset: start}
		}
		scanner.previous = apkTokenRevision
		return apkToken{kind: apkTokenRevision, value: scanner.input[numberStart:scanner.position], offset: start}
	default:
		return apkToken{kind: apkTokenInvalid, offset: start}
	}
}

// compareAPKToken 比较类型相同的 token。
func compareAPKToken(left, right apkToken) int {
	switch left.kind {
	case apkTokenInitialDigit, apkTokenSuffixNumber, apkTokenRevision:
		return compareNumericStrings(left.value, right.value)
	case apkTokenDigit:
		// APK 对任意一侧以 0 开始的非首数字段使用原始字节序，这是与 RPM 的关键差异。
		if left.value[0] == '0' || right.value[0] == '0' {
			return compareBytes(left.value, right.value)
		}
		return compareNumericStrings(left.value, right.value)
	case apkTokenLetter, apkTokenCommitHash:
		return compareBytes(left.value, right.value)
	case apkTokenSuffix:
		if left.suffix < right.suffix {
			return -1
		}
		if left.suffix > right.suffix {
			return 1
		}
	}
	return 0
}

// parseAPKSuffix 返回 apk-tools 规定的 suffix 顺序。
func parseAPKSuffix(suffix string) (int, bool) {
	switch suffix {
	case "alpha":
		return apkSuffixAlpha, true
	case "beta":
		return apkSuffixBeta, true
	case "pre":
		return apkSuffixPre, true
	case "rc":
		return apkSuffixRC, true
	case "cvs":
		return apkSuffixCVS, true
	case "svn":
		return apkSuffixSVN, true
	case "git":
		return apkSuffixGit, true
	case "hg":
		return apkSuffixHg, true
	case "p":
		return apkSuffixPatch, true
	default:
		return 0, false
	}
}

func isAPKHexDigit(character byte) bool {
	return isASCIIDigit(character) || character >= 'a' && character <= 'f'
}
