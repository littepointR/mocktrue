package protocol

// Kind identifies the parser implementation type.
type Kind string

const (
	KindVisual   Kind = "visual"
	KindScript   Kind = "script"
	KindTemplate Kind = "template"
)

// Field represents a parsed field within a frame.
type Field struct {
	Name    string
	Offset  int
	Width   int
	Type    string // "u8"|"u16"|"u32"|"i8"|"i16"|"i32"|"f32"|"bytes"|"ascii"
	Value   any
	Display string // formatted display value
}

// ParseResult is the output of a parser invocation.
type ParseResult struct {
	OK       bool
	Fields   []Field
	Errors   []string
	Consumed int // bytes consumed from input
	NeedMore int // additional bytes needed (0 if frame complete)
}

// ParserConfig is the serializable configuration for a parser.
type ParserConfig struct {
	Kind     Kind       `json:"kind"`
	Visual   *FrameSpec `json:"visual,omitempty"`
	Script   string     `json:"script,omitempty"`
	Template string     `json:"template,omitempty"`
}

// Parser is the interface all protocol parsers implement.
type Parser interface {
	Name() string
	Kind() Kind
	Config() ParserConfig
	Parse(data []byte) (ParseResult, error)
}
