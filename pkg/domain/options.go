package domain

import "time"

// VariableStyle defines variable naming styles
type VariableStyle int

const (
	StyleCamelCase VariableStyle = iota
	StyleSnakeCase
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
		return "unknown"
	}
}

// MatchRule defines how fields are matched between types
type MatchRule int

const (
	MatchByName MatchRule = iota
	MatchByType
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
		return "unknown"
	}
}

// ConversionType defines the type of conversion to perform
type ConversionType int

const (
	ConversionDirect ConversionType = iota
	ConversionCast
	ConversionMethod
	ConversionCustom
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
		return "unknown"
	}
}

// ErrorHandlingMethod defines how errors are handled in generated code
type ErrorHandlingMethod int

const (
	ErrorHandlingNone ErrorHandlingMethod = iota
	ErrorHandlingReturn
	ErrorHandlingPanic
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
		return "unknown"
	}
}

// InterfaceOptions contains options that apply to an entire interface
type InterfaceOptions struct {
	Style                VariableStyle     `json:"style"`
	MatchRule            MatchRule         `json:"match_rule"`
	CaseSensitive        bool              `json:"case_sensitive"`
	UseGetter            bool              `json:"use_getter"`
	UseStringer          bool              `json:"use_stringer"`
	UseTypecast          bool              `json:"use_typecast"`
	ReceiverName         string            `json:"receiver_name"`
	AllowReverse         bool              `json:"allow_reverse"`
	SkipFields           []string          `json:"skip_fields"`
	FieldMappings        map[string]string `json:"field_mappings"`
	TypeConverters       map[string]string `json:"type_converters"`
	LiteralAssignments   map[string]string `json:"literal_assignments"`
	PreprocessFunction   string            `json:"preprocess_function"`
	PostprocessFunction  string            `json:"postprocess_function"`
}

// MethodOptions contains options that apply to a specific method
type MethodOptions struct {
	Style                VariableStyle     `json:"style"`
	MatchRule            MatchRule         `json:"match_rule"`
	CaseSensitive        bool              `json:"case_sensitive"`
	UseGetter            bool              `json:"use_getter"`
	UseStringer          bool              `json:"use_stringer"`
	UseTypecast          bool              `json:"use_typecast"`
	AllowReverse         bool              `json:"allow_reverse"`
	SkipFields           []string          `json:"skip_fields"`
	FieldMappings        map[string]string `json:"field_mappings"`
	TypeConverters       map[string]string `json:"type_converters"`
	LiteralAssignments   map[string]string `json:"literal_assignments"`
	PreprocessFunction   string            `json:"preprocess_function"`
	PostprocessFunction  string            `json:"postprocess_function"`
	CustomValidation     string            `json:"custom_validation"`
	ConcurrencyLevel     int               `json:"concurrency_level"`
	TimeoutDuration      time.Duration     `json:"timeout_duration"`
}

// ChannelDirection represents the direction of a channel
type ChannelDirection int

const (
	ChannelBidirectional ChannelDirection = iota
	ChannelSendOnly
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
		return "unknown"
	}
}