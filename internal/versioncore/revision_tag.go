package versioncore

// stripRevisionTagPrefix 仅剥离完整 r/R 数字发布标签的包装，不从任意分支或路径中截取数字。
// 必须在不透明版本识别前执行，让 r36 也能与 r36-p1 使用相同主版本；p1 保留为后缀，
// 不改写为小版本，也不推断其高于正式版。适用于普通及通用 OS 解析，不用于原生包管理器。
func stripRevisionTagPrefix(value string) string {
	if len(value) < 2 || (value[0] != 'r' && value[0] != 'R') || !isDigit(value[1]) {
		return value
	}
	position := 1
	for {
		for position < len(value) && isDigit(value[position]) {
			position++
		}
		if position+1 < len(value) && (value[position] == '.' || value[position] == '_') && isDigit(value[position+1]) {
			position++
			continue
		}
		break
	}
	if position == len(value) {
		return value[1:]
	}
	if position+1 >= len(value) || (value[position] != '-' && value[position] != '+') {
		return value
	}
	// 后缀首字符必须是字母或数字；剩余只允许普通版本标点，不把范围语法或路径当作标签。
	for index := position + 1; index < len(value); index++ {
		character := value[index]
		if isDigit(character) || isLetter(character) || (character >= 'A' && character <= 'Z') {
			continue
		}
		if index > position+1 && (character == '.' || character == '_' || character == '-' || character == '+') {
			continue
		}
		return value
	}
	return value[1:]
}
