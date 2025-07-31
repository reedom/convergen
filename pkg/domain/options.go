package domain

import "time"

// VariableStyle defines variable naming styles.
type VariableStyle int

const (
	// StyleCamelCase uses camelCase naming convention.
	StyleCamelCase VariableStyle = iota
	// StyleSnakeCase uses snake_case naming convention.
	StyleSnakeCase
	// StylePascalCase uses PascalCase naming convention.
	StylePascalCase
)

func (v VariableStyle) String() string {
	switch v {
	case StyleCamelCase:
		return "camelCase"
	case StyleSnakeCase:
		return "snake_case"
	case StylePascalCase:
		return "PascalCase"
	default:
		return UnknownValue
	}
}

// MatchRule defines how fields are matched between types.
type MatchRule int

const (
	// MatchByName matches fields by their names.
	MatchByName MatchRule = iota
	// MatchByType matches fields by their types.
	MatchByType
	// MatchByTag matches fields by their struct tags.
	MatchByTag
)

func (m MatchRule) String() string {
	switch m {
	case MatchByName:
		return "name"
	case MatchByType:
		return "type"
	case MatchByTag:
		return "tag"
	default:
		return UnknownValue
	}
}

// ConversionType defines the type of conversion to perform.
type ConversionType int

const (
	// ConversionDirect performs direct field assignment.
	ConversionDirect ConversionType = iota
	// ConversionCast performs type casting conversion.
	ConversionCast
	// ConversionMethod uses method calls for conversion.
	ConversionMethod
	// ConversionCustom uses custom converter functions.
	ConversionCustom
	// ConversionLiteral assigns literal values.
	ConversionLiteral
)

func (c ConversionType) String() string {
	switch c {
	case ConversionDirect:
		return "direct"
	case ConversionCast:
		return "cast"
	case ConversionMethod:
		return "method"
	case ConversionCustom:
		return "custom"
	case ConversionLiteral:
		return "literal"
	default:
		return UnknownValue
	}
}

// ErrorHandlingMethod defines how errors are handled in generated code.
type ErrorHandlingMethod int

const (
	// ErrorHandlingNone ignores errors during conversion.
	ErrorHandlingNone ErrorHandlingMethod = iota
	// ErrorHandlingReturn returns errors to the caller.
	ErrorHandlingReturn
	// ErrorHandlingPanic panics on conversion errors.
	ErrorHandlingPanic
	// ErrorHandlingLog logs errors and continues.
	ErrorHandlingLog
)

func (e ErrorHandlingMethod) String() string {
	switch e {
	case ErrorHandlingNone:
		return "none"
	case ErrorHandlingReturn:
		return "return"
	case ErrorHandlingPanic:
		return "panic"
	case ErrorHandlingLog:
		return "log"
	default:
		return UnknownValue
	}
}

// InterfaceOptions contains options that apply to an entire interface.
type InterfaceOptions struct {
	Style               VariableStyle     `json:"style"`
	MatchRule           MatchRule         `json:"match_rule"`
	CaseSensitive       bool              `json:"case_sensitive"`
	UseGetter           bool              `json:"use_getter"`
	UseStringer         bool              `json:"use_stringer"`
	UseTypecast         bool              `json:"use_typecast"`
	ReceiverName        string            `json:"receiver_name"`
	AllowReverse        bool              `json:"allow_reverse"`
	SkipFields          []string          `json:"skip_fields"`
	FieldMappings       map[string]string `json:"field_mappings"`
	TypeConverters      map[string]string `json:"type_converters"`
	LiteralAssignments  map[string]string `json:"literal_assignments"`
	PreprocessFunction  string            `json:"preprocess_function"`
	PostprocessFunction string            `json:"postprocess_function"`
}

// MethodOptions contains options that apply to a specific method.
type MethodOptions struct {
	Style               VariableStyle     `json:"style"`
	MatchRule           MatchRule         `json:"match_rule"`
	CaseSensitive       bool              `json:"case_sensitive"`
	UseGetter           bool              `json:"use_getter"`
	UseStringer         bool              `json:"use_stringer"`
	UseTypecast         bool              `json:"use_typecast"`
	AllowReverse        bool              `json:"allow_reverse"`
	SkipFields          []string          `json:"skip_fields"`
	FieldMappings       map[string]string `json:"field_mappings"`
	TypeConverters      map[string]string `json:"type_converters"`
	LiteralAssignments  map[string]string `json:"literal_assignments"`
	PreprocessFunction  string            `json:"preprocess_function"`
	PostprocessFunction string            `json:"postprocess_function"`
	CustomValidation    string            `json:"custom_validation"`
	ConcurrencyLevel    int               `json:"concurrency_level"`
	TimeoutDuration     time.Duration     `json:"timeout_duration"`
}

// ChannelDirection represents the direction of a channel.
type ChannelDirection int

const (
	// ChannelBidirectional represents a bidirectional channel.
	ChannelBidirectional ChannelDirection = iota
	// ChannelSendOnly represents a send-only channel.
	ChannelSendOnly
	// ChannelReceiveOnly represents a receive-only channel.
	ChannelReceiveOnly
)

func (c ChannelDirection) String() string {
	switch c {
	case ChannelBidirectional:
		return "bidirectional"
	case ChannelSendOnly:
		return "send-only"
	case ChannelReceiveOnly:
		return "receive-only"
	default:
		return UnknownValue
	}
}
