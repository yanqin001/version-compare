// Package rangeexpr 提供严格、无版本排序语义的版本范围表达式解析能力。
package rangeexpr

import (
	"fmt"
	"strings"
)

// Bound 表示区间的一侧边界。
// Version 是边界版本；Inclusive 表示是否包含该版本；Set 表示边界是否存在。
type Bound struct {
	Version   string
	Inclusive bool
	Set       bool
}

// Interval 表示由可选上下界组成的版本区间。
type Interval struct {
	Lower Bound
	Upper Bound
}

// Expression 表示解析后的版本范围表达式。
// Any 仅表示显式通配符；Intervals 中的多个区间按 OR 关系组合。
type Expression struct {
	Any       bool
	Intervals []Interval
}

type comparisonDirection uint8

const (
	directionEqual comparisonDirection = iota
	directionLower
	directionUpper
)

type comparison struct {
	direction comparisonDirection
	bound     Bound
}

type scanner struct {
	input string
	pos   int
}

// Parse 将版本范围字符串解析为结构化表达式。
// 空表达式、缺失版本、非法运算符、残缺区间及歧义组合都会返回错误。
func Parse(input string) (Expression, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Expression{}, fmt.Errorf("range expression is empty")
	}

	branches, err := splitORBranches(input)
	if err != nil {
		return Expression{}, err
	}

	result := Expression{}
	for _, branch := range branches {
		branch = strings.TrimSpace(branch)
		if branch == "*" {
			result.Any = true
			continue
		}

		intervals, err := parseBranch(branch)
		if err != nil {
			return Expression{}, err
		}
		result.Intervals = append(result.Intervals, intervals...)
	}

	if result.Any {
		// 通配符与任何范围做 OR 后仍是通配，返回规范化结果可简化调用方判断。
		result.Intervals = nil
	}
	if !result.Any && len(result.Intervals) == 0 {
		return Expression{}, fmt.Errorf("range expression contains no interval")
	}
	return result, nil
}

// splitORBranches 只在区间外识别 ||，避免把区间内容误拆成独立分支。
// 区间允许四种开闭组合，因此这里只跟踪是否位于区间内，不要求左右括号同类。
func splitORBranches(input string) ([]string, error) {
	branches := make([]string, 0, 2)
	start := 0
	openAt := -1

	for i := 0; i < len(input); i++ {
		switch input[i] {
		case '[', '(':
			if openAt >= 0 {
				return nil, syntaxError(input, i, "nested interval is not allowed")
			}
			openAt = i
		case ']', ')':
			if openAt < 0 {
				return nil, syntaxError(input, i, "unexpected closing bracket")
			}
			openAt = -1
		case '|':
			if openAt >= 0 {
				return nil, syntaxError(input, i, "OR operator is not allowed inside an interval")
			}
			if i+1 >= len(input) || input[i+1] != '|' {
				return nil, syntaxError(input, i, "OR operator must be ||")
			}
			branch := strings.TrimSpace(input[start:i])
			if branch == "" {
				return nil, syntaxError(input, i, "empty OR branch")
			}
			branches = append(branches, branch)
			i++
			start = i + 1
		}
	}

	if openAt >= 0 {
		return nil, syntaxError(input, openAt, "interval is not closed")
	}
	last := strings.TrimSpace(input[start:])
	if last == "" {
		return nil, syntaxError(input, len(input), "empty OR branch")
	}
	return append(branches, last), nil
}

func parseBranch(branch string) ([]Interval, error) {
	if branch == "" {
		return nil, fmt.Errorf("range expression contains an empty branch")
	}
	if isComparisonStart(branch[0]) {
		return parseComparisonBranch(branch)
	}
	return parseValueBranch(branch)
}

// parseComparisonBranch 解析单个比较条件、由上下界组成的双条件，或逗号分隔的等值条件。
// 双条件必须恰好包含一个下界和一个上界，顺序可以互换。
func parseComparisonBranch(input string) ([]Interval, error) {
	s := scanner{input: input}
	first, err := s.parseComparison()
	if err != nil {
		return nil, err
	}

	hadSpace := s.skipSpaces()
	if s.atEnd() {
		return []Interval{comparisonInterval(first)}, nil
	}

	commaSeparated := false
	if s.input[s.pos] == ',' {
		commaSeparated = true
		s.pos++
		s.skipSpaces()
	} else if !hadSpace {
		return nil, syntaxError(input, s.pos, "comparison conditions need a comma or whitespace separator")
	}
	if s.atEnd() {
		return nil, syntaxError(input, s.pos, "missing comparison condition")
	}

	// 多个等值条件以逗号连接时表示精确版本列表，而不是区间交集。
	if first.direction == directionEqual {
		if !commaSeparated {
			return nil, syntaxError(input, s.pos, "exact comparisons must be separated by commas")
		}
		intervals := []Interval{comparisonInterval(first)}
		for {
			next, err := s.parseComparison()
			if err != nil {
				return nil, err
			}
			if next.direction != directionEqual {
				return nil, syntaxError(input, s.pos, "exact comparison list contains a range operator")
			}
			intervals = append(intervals, comparisonInterval(next))

			s.skipSpaces()
			if s.atEnd() {
				return intervals, nil
			}
			if s.input[s.pos] != ',' {
				return nil, syntaxError(input, s.pos, "exact comparisons must be separated by commas")
			}
			s.pos++
			s.skipSpaces()
			if s.atEnd() {
				return nil, syntaxError(input, s.pos, "missing exact comparison")
			}
		}
	}

	second, err := s.parseComparison()
	if err != nil {
		return nil, err
	}
	s.skipSpaces()
	if !s.atEnd() {
		return nil, syntaxError(input, s.pos, "a range can contain only two comparison conditions")
	}
	if second.direction == directionEqual {
		return nil, syntaxError(input, s.pos, "equality cannot be combined with a range condition")
	}
	if first.direction == second.direction {
		return nil, syntaxError(input, s.pos, "a range needs one lower and one upper condition")
	}

	interval := Interval{}
	applyComparison(&interval, first)
	applyComparison(&interval, second)
	return []Interval{interval}, nil
}

func (s *scanner) parseComparison() (comparison, error) {
	s.skipSpaces()
	if s.atEnd() {
		return comparison{}, syntaxError(s.input, s.pos, "missing comparison operator")
	}

	start := s.pos
	var result comparison
	switch s.input[s.pos] {
	case '<':
		result.direction = directionUpper
		s.pos++
		result.bound.Inclusive = s.consume('=')
	case '>':
		result.direction = directionLower
		s.pos++
		result.bound.Inclusive = s.consume('=')
	case '=':
		result.direction = directionEqual
		result.bound.Inclusive = true
		s.pos++
		s.consume('=')
	default:
		return comparison{}, syntaxError(s.input, start, "expected a comparison operator")
	}

	if !s.atEnd() && isComparisonStart(s.input[s.pos]) {
		return comparison{}, syntaxError(s.input, s.pos, "invalid comparison operator")
	}
	s.skipSpaces()
	versionAt := s.pos
	for !s.atEnd() && s.input[s.pos] != ',' && !isSpace(s.input[s.pos]) {
		if isReservedVersionByte(s.input[s.pos]) {
			return comparison{}, syntaxError(s.input, s.pos, "invalid character in comparison version")
		}
		s.pos++
	}
	if s.pos == versionAt {
		return comparison{}, syntaxError(s.input, versionAt, "comparison version is empty")
	}

	result.bound.Version = s.input[versionAt:s.pos]
	result.bound.Set = true
	return result, nil
}

func comparisonInterval(c comparison) Interval {
	interval := Interval{}
	applyComparison(&interval, c)
	return interval
}

func applyComparison(interval *Interval, c comparison) {
	switch c.direction {
	case directionEqual:
		interval.Lower = c.bound
		interval.Upper = c.bound
	case directionLower:
		interval.Lower = c.bound
	case directionUpper:
		interval.Upper = c.bound
	}
}

// parseValueBranch 解析裸精确值、方括号精确值以及区间列表。
// 逗号表示 OR；两个括号表达式也可以直接相邻，以兼容常见漏洞数据格式。
func parseValueBranch(input string) ([]Interval, error) {
	s := scanner{input: input}
	intervals := make([]Interval, 0, 2)

	for {
		s.skipSpaces()
		if s.atEnd() {
			if len(intervals) == 0 {
				return nil, syntaxError(input, s.pos, "missing value")
			}
			return intervals, nil
		}

		var (
			interval Interval
			bracket  bool
			err      error
		)
		if s.input[s.pos] == '[' || s.input[s.pos] == '(' {
			interval, err = s.parseBracketInterval()
			bracket = true
		} else {
			interval, err = s.parseBareExact()
		}
		if err != nil {
			return nil, err
		}
		intervals = append(intervals, interval)

		s.skipSpaces()
		if s.atEnd() {
			return intervals, nil
		}
		if s.input[s.pos] == ',' {
			s.pos++
			s.skipSpaces()
			if s.atEnd() {
				return nil, syntaxError(input, s.pos, "missing value after comma")
			}
			continue
		}

		// 省略分隔符只允许发生在两个括号表达式之间，避免把 a[b] 等歧义文本静默拆开。
		if bracket && (s.input[s.pos] == '[' || s.input[s.pos] == '(') {
			continue
		}
		return nil, syntaxError(input, s.pos, "values need a comma separator")
	}
}

func (s *scanner) parseBracketInterval() (Interval, error) {
	openAt := s.pos
	open := s.input[s.pos]
	s.pos++
	contentAt := s.pos

	for !s.atEnd() && s.input[s.pos] != ']' && s.input[s.pos] != ')' {
		if s.input[s.pos] == '[' || s.input[s.pos] == '(' {
			return Interval{}, syntaxError(s.input, s.pos, "nested interval is not allowed")
		}
		if s.input[s.pos] == '|' || s.input[s.pos] == '<' || s.input[s.pos] == '>' || s.input[s.pos] == '=' || s.input[s.pos] == '*' {
			return Interval{}, syntaxError(s.input, s.pos, "invalid character inside interval")
		}
		s.pos++
	}
	if s.atEnd() {
		return Interval{}, syntaxError(s.input, openAt, "interval is not closed")
	}

	close := s.input[s.pos]
	content := s.input[contentAt:s.pos]
	s.pos++
	commaCount := strings.Count(content, ",")
	if commaCount == 0 {
		if open != '[' || close != ']' {
			return Interval{}, syntaxError(s.input, openAt, "a parenthesized interval needs two endpoints")
		}
		version := strings.TrimSpace(content)
		if err := validateVersion(version); err != nil {
			return Interval{}, syntaxError(s.input, contentAt, err.Error())
		}
		bound := Bound{Version: version, Inclusive: true, Set: true}
		return Interval{Lower: bound, Upper: bound}, nil
	}
	if commaCount != 1 {
		return Interval{}, syntaxError(s.input, contentAt, "an interval must contain exactly one comma")
	}

	commaAt := strings.IndexByte(content, ',')
	lowerText := strings.TrimSpace(content[:commaAt])
	upperText := strings.TrimSpace(content[commaAt+1:])
	interval := Interval{}
	if lowerText != "" {
		if err := validateVersion(lowerText); err != nil {
			return Interval{}, syntaxError(s.input, contentAt, err.Error())
		}
		interval.Lower = Bound{Version: lowerText, Inclusive: open == '[', Set: true}
	}
	if upperText != "" {
		if err := validateVersion(upperText); err != nil {
			return Interval{}, syntaxError(s.input, contentAt+commaAt+1, err.Error())
		}
		interval.Upper = Bound{Version: upperText, Inclusive: close == ']', Set: true}
	}
	return interval, nil
}

func (s *scanner) parseBareExact() (Interval, error) {
	start := s.pos
	for !s.atEnd() && s.input[s.pos] != ',' && !isSpace(s.input[s.pos]) {
		if isReservedVersionByte(s.input[s.pos]) {
			return Interval{}, syntaxError(s.input, s.pos, "invalid character in exact version")
		}
		s.pos++
	}
	if s.pos == start {
		return Interval{}, syntaxError(s.input, start, "exact version is empty")
	}
	version := s.input[start:s.pos]
	bound := Bound{Version: version, Inclusive: true, Set: true}
	return Interval{Lower: bound, Upper: bound}, nil
}

func validateVersion(version string) error {
	if version == "" {
		return fmt.Errorf("version is empty")
	}
	for i := 0; i < len(version); i++ {
		if isSpace(version[i]) || version[i] == ',' || isReservedVersionByte(version[i]) {
			return fmt.Errorf("version contains a reserved character")
		}
	}
	return nil
}

func (s *scanner) skipSpaces() bool {
	start := s.pos
	for !s.atEnd() && isSpace(s.input[s.pos]) {
		s.pos++
	}
	return s.pos != start
}

func (s *scanner) consume(want byte) bool {
	if s.atEnd() || s.input[s.pos] != want {
		return false
	}
	s.pos++
	return true
}

func (s *scanner) atEnd() bool {
	return s.pos >= len(s.input)
}

func isComparisonStart(b byte) bool {
	return b == '<' || b == '>' || b == '='
}

func isReservedVersionByte(b byte) bool {
	return b == '[' || b == ']' || b == '(' || b == ')' || b == '<' || b == '>' || b == '=' || b == '|' || b == '*'
}

func isSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}

func syntaxError(input string, pos int, message string) error {
	return fmt.Errorf("invalid range expression %q at byte %d: %s", input, pos, message)
}
