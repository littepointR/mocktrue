package serial

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

type serialGraphFilterOptions struct {
	Mode          string
	Expression    string
	CaseSensitive bool
	WholeWord     bool
}

type serialGraphFilterCandidate struct {
	Len         int
	ByteLength  int
	ByteCount   int
	Text        string
	Hex         string
	Message     string
	Level       string
	Source      string
	GraphID     string
	NodeID      string
	NodeType    string
	Direction   string
	PayloadText string
	PayloadHex  string
	Details     string
	Action      string
	Category    string
}

func matchSerialGraphFilter(candidate serialGraphFilterCandidate, options serialGraphFilterOptions) (bool, error) {
	expression := strings.TrimSpace(options.Expression)
	if expression == "" {
		return true, nil
	}

	switch strings.ToLower(strings.TrimSpace(options.Mode)) {
	case "", "plain":
		return matchSerialGraphPlainFilter(candidate, expression, options), nil
	case "regex":
		return matchSerialGraphRegexFilter(candidate, expression, options)
	case "expression":
		return matchSerialGraphExpressionFilter(candidate, expression, options)
	default:
		return false, fmt.Errorf("unknown filter mode: %s", options.Mode)
	}
}

func matchSerialGraphPlainFilter(candidate serialGraphFilterCandidate, expression string, options serialGraphFilterOptions) bool {
	haystack := candidate.filterHaystack()
	needle := expression
	if !options.CaseSensitive {
		haystack = strings.ToLower(haystack)
		needle = strings.ToLower(needle)
	}
	if !options.WholeWord {
		return strings.Contains(haystack, needle)
	}
	pattern := `\b` + regexp.QuoteMeta(needle) + `\b`
	return regexp.MustCompile(pattern).MatchString(haystack)
}

func matchSerialGraphRegexFilter(candidate serialGraphFilterCandidate, expression string, options serialGraphFilterOptions) (bool, error) {
	pattern := expression
	if options.WholeWord {
		pattern = `\b(?:` + pattern + `)\b`
	}
	if !options.CaseSensitive {
		pattern = `(?i:` + pattern + `)`
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("invalid regex: %w", err)
	}
	return re.MatchString(candidate.filterHaystack()), nil
}

func matchSerialGraphExpressionFilter(candidate serialGraphFilterCandidate, expression string, options serialGraphFilterOptions) (bool, error) {
	tokens, err := tokenizeSerialGraphFilterExpression(expression)
	if err != nil {
		return false, err
	}
	parser := serialGraphFilterParser{tokens: tokens, candidate: candidate, options: options}
	matched, err := parser.parse()
	if err != nil {
		return false, err
	}
	return matched, nil
}

func (c serialGraphFilterCandidate) filterHaystack() string {
	values := []string{
		c.Text,
		c.Hex,
		c.Message,
		c.Level,
		c.Source,
		c.GraphID,
		c.NodeID,
		c.NodeType,
		c.Direction,
		c.PayloadText,
		c.PayloadHex,
		c.Details,
		c.Action,
		c.Category,
		strconv.Itoa(c.length()),
	}
	return strings.Join(values, "\n")
}

func (c serialGraphFilterCandidate) length() int {
	if c.Len != 0 {
		return c.Len
	}
	if c.ByteLength != 0 {
		return c.ByteLength
	}
	return c.ByteCount
}

func (c serialGraphFilterCandidate) numericField(name string) (int, bool) {
	switch strings.ToLower(name) {
	case "len", "bytelength", "bytecount":
		return c.length(), true
	default:
		return 0, false
	}
}

func (c serialGraphFilterCandidate) stringField(name string) (string, bool) {
	switch strings.ToLower(name) {
	case "text":
		if c.Text != "" {
			return c.Text, true
		}
		return c.PayloadText, true
	case "hex":
		if c.Hex != "" {
			return c.Hex, true
		}
		return c.PayloadHex, true
	case "message":
		return c.Message, true
	case "level":
		return c.Level, true
	case "source":
		return c.Source, true
	case "graphid":
		return c.GraphID, true
	case "nodeid":
		return c.NodeID, true
	case "nodetype":
		return c.NodeType, true
	case "direction":
		return c.Direction, true
	case "payloadtext":
		return c.PayloadText, true
	case "payloadhex":
		return c.PayloadHex, true
	case "details":
		return c.Details, true
	case "action":
		return c.Action, true
	case "category":
		return c.Category, true
	default:
		return "", false
	}
}

type serialGraphFilterTokenKind int

const (
	serialGraphFilterTokenEOF serialGraphFilterTokenKind = iota
	serialGraphFilterTokenIdent
	serialGraphFilterTokenNumber
	serialGraphFilterTokenString
	serialGraphFilterTokenOperator
	serialGraphFilterTokenLParen
	serialGraphFilterTokenRParen
)

type serialGraphFilterToken struct {
	kind  serialGraphFilterTokenKind
	value string
}

func tokenizeSerialGraphFilterExpression(input string) ([]serialGraphFilterToken, error) {
	tokens := make([]serialGraphFilterToken, 0, len(input)/2+1)
	for index := 0; index < len(input); {
		r := rune(input[index])
		if unicode.IsSpace(r) {
			index++
			continue
		}
		if input[index] == '(' {
			tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenLParen, value: "("})
			index++
			continue
		}
		if input[index] == ')' {
			tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenRParen, value: ")"})
			index++
			continue
		}
		if input[index] == '"' {
			value, next, err := scanSerialGraphFilterString(input, index)
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenString, value: value})
			index = next
			continue
		}
		if input[index] >= '0' && input[index] <= '9' {
			start := index
			for index < len(input) && input[index] >= '0' && input[index] <= '9' {
				index++
			}
			tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenNumber, value: input[start:index]})
			continue
		}
		if isSerialGraphFilterIdentifierStart(input[index]) {
			start := index
			for index < len(input) && isSerialGraphFilterIdentifierPart(input[index]) {
				index++
			}
			tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenIdent, value: input[start:index]})
			continue
		}
		if strings.ContainsRune("=!<>", rune(input[index])) {
			if index+1 < len(input) {
				two := input[index : index+2]
				if two == "==" || two == "!=" || two == ">=" || two == "<=" {
					tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenOperator, value: two})
					index += 2
					continue
				}
			}
			if input[index] == '>' || input[index] == '<' {
				tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenOperator, value: input[index : index+1]})
				index++
				continue
			}
		}
		return nil, fmt.Errorf("unexpected token %q", input[index:index+1])
	}
	tokens = append(tokens, serialGraphFilterToken{kind: serialGraphFilterTokenEOF})
	return tokens, nil
}

func scanSerialGraphFilterString(input string, start int) (string, int, error) {
	var builder strings.Builder
	for index := start + 1; index < len(input); index++ {
		ch := input[index]
		if ch == '"' {
			return builder.String(), index + 1, nil
		}
		if ch == '\\' {
			if index+1 >= len(input) {
				return "", 0, fmt.Errorf("unterminated string literal")
			}
			index++
			switch input[index] {
			case '"', '\\':
				builder.WriteByte(input[index])
			case 'n':
				builder.WriteByte('\n')
			case 'r':
				builder.WriteByte('\r')
			case 't':
				builder.WriteByte('\t')
			default:
				builder.WriteByte(input[index])
			}
			continue
		}
		builder.WriteByte(ch)
	}
	return "", 0, fmt.Errorf("unterminated string literal")
}

func isSerialGraphFilterIdentifierStart(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_'
}

func isSerialGraphFilterIdentifierPart(ch byte) bool {
	return isSerialGraphFilterIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

type serialGraphFilterParser struct {
	tokens    []serialGraphFilterToken
	position  int
	candidate serialGraphFilterCandidate
	options   serialGraphFilterOptions
}

func (p *serialGraphFilterParser) parse() (bool, error) {
	matched, err := p.parseOr()
	if err != nil {
		return false, err
	}
	if p.peek().kind != serialGraphFilterTokenEOF {
		return false, fmt.Errorf("unexpected token %q", p.peek().value)
	}
	return matched, nil
}

func (p *serialGraphFilterParser) parseOr() (bool, error) {
	left, err := p.parseAnd()
	if err != nil {
		return false, err
	}
	for p.matchKeyword("or") {
		right, err := p.parseAnd()
		if err != nil {
			return false, err
		}
		left = left || right
	}
	return left, nil
}

func (p *serialGraphFilterParser) parseAnd() (bool, error) {
	left, err := p.parseUnary()
	if err != nil {
		return false, err
	}
	for p.matchKeyword("and") {
		right, err := p.parseUnary()
		if err != nil {
			return false, err
		}
		left = left && right
	}
	return left, nil
}

func (p *serialGraphFilterParser) parseUnary() (bool, error) {
	if p.matchKeyword("not") {
		matched, err := p.parseUnary()
		if err != nil {
			return false, err
		}
		return !matched, nil
	}
	return p.parsePrimary()
}

func (p *serialGraphFilterParser) parsePrimary() (bool, error) {
	if p.matchKind(serialGraphFilterTokenLParen) {
		matched, err := p.parseOr()
		if err != nil {
			return false, err
		}
		if !p.matchKind(serialGraphFilterTokenRParen) {
			return false, fmt.Errorf("expected closing parenthesis")
		}
		return matched, nil
	}
	return p.parseComparison()
}

func (p *serialGraphFilterParser) parseComparison() (bool, error) {
	fieldToken := p.peek()
	if fieldToken.kind != serialGraphFilterTokenIdent {
		return false, fmt.Errorf("expected filter field")
	}
	p.position++
	field := fieldToken.value

	if value, ok := p.candidate.numericField(field); ok {
		operator := p.peek()
		if operator.kind != serialGraphFilterTokenOperator || !isSerialGraphNumericOperator(operator.value) {
			return false, fmt.Errorf("expected numeric comparison after %s", field)
		}
		p.position++
		numberToken := p.peek()
		if numberToken.kind != serialGraphFilterTokenNumber {
			return false, fmt.Errorf("expected numeric literal after %s", operator.value)
		}
		p.position++
		right, err := strconv.Atoi(numberToken.value)
		if err != nil {
			return false, fmt.Errorf("invalid numeric literal %q", numberToken.value)
		}
		return compareSerialGraphFilterNumber(value, operator.value, right), nil
	}

	value, ok := p.candidate.stringField(field)
	if !ok {
		return false, fmt.Errorf("unknown filter field: %s", field)
	}
	if p.matchKeyword("contains") {
		literal, err := p.expectStringLiteral("after contains")
		if err != nil {
			return false, err
		}
		return compareSerialGraphFilterString(field, value, "contains", literal, p.options), nil
	}
	operator := p.peek()
	if operator.kind != serialGraphFilterTokenOperator || (operator.value != "==" && operator.value != "!=") {
		return false, fmt.Errorf("expected string comparison after %s", field)
	}
	p.position++
	literal, err := p.expectStringLiteral("after " + operator.value)
	if err != nil {
		return false, err
	}
	return compareSerialGraphFilterString(field, value, operator.value, literal, p.options), nil
}

func (p *serialGraphFilterParser) expectStringLiteral(context string) (string, error) {
	token := p.peek()
	if token.kind != serialGraphFilterTokenString {
		return "", fmt.Errorf("expected string literal %s", context)
	}
	p.position++
	return token.value, nil
}

func (p *serialGraphFilterParser) peek() serialGraphFilterToken {
	if p.position >= len(p.tokens) {
		return serialGraphFilterToken{kind: serialGraphFilterTokenEOF}
	}
	return p.tokens[p.position]
}

func (p *serialGraphFilterParser) matchKind(kind serialGraphFilterTokenKind) bool {
	if p.peek().kind != kind {
		return false
	}
	p.position++
	return true
}

func (p *serialGraphFilterParser) matchKeyword(keyword string) bool {
	token := p.peek()
	if token.kind != serialGraphFilterTokenIdent || !strings.EqualFold(token.value, keyword) {
		return false
	}
	p.position++
	return true
}

func isSerialGraphNumericOperator(operator string) bool {
	switch operator {
	case "==", "!=", ">", ">=", "<", "<=":
		return true
	default:
		return false
	}
}

func compareSerialGraphFilterNumber(left int, operator string, right int) bool {
	switch operator {
	case "==":
		return left == right
	case "!=":
		return left != right
	case ">":
		return left > right
	case ">=":
		return left >= right
	case "<":
		return left < right
	case "<=":
		return left <= right
	default:
		return false
	}
}

func compareSerialGraphFilterString(field string, left string, operator string, right string, options serialGraphFilterOptions) bool {
	if strings.EqualFold(field, "hex") || strings.EqualFold(field, "payloadhex") {
		left = normalizeSerialGraphFilterHex(left)
		right = normalizeSerialGraphFilterHex(right)
	} else if !options.CaseSensitive {
		left = strings.ToLower(left)
		right = strings.ToLower(right)
	}
	switch operator {
	case "contains":
		return strings.Contains(left, right)
	case "==":
		return left == right
	case "!=":
		return left != right
	default:
		return false
	}
}

func normalizeSerialGraphFilterHex(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if unicode.IsSpace(r) {
			continue
		}
		builder.WriteRune(unicode.ToLower(r))
	}
	return builder.String()
}
