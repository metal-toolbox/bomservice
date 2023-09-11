package model

// AppKind defines the types of application this can be.
type AppKind string

// LogLevel is the logging level string.
type LogLevel string

const (
	AppName string = "hollow-bomservice"

	// AppKindServer identifies a hollow bomservice.
	AppKindServer AppKind = "hollow-bomservice-server"

	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
	LogLevelTrace LogLevel = "trace"
)
