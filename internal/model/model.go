package model

// AppKind defines the types of application this can be.
type AppKind string

// LogLevel is the logging level string.
type LogLevel string

const (
	AppName string = "bomservice"

	// AppKindServer identifies a bomservice.
	AppKindServer AppKind = "bomservice-server"

	LogLevelInfo  LogLevel = "info"
	LogLevelDebug LogLevel = "debug"
	LogLevelTrace LogLevel = "trace"
)
