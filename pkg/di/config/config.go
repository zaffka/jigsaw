package config

import (
	"fmt"
	"time"
)

// CommonServices holds configuration for common services registration.
type CommonServices struct {
	Logger   Logger
	HTTP     HTTPServer
	Database Database
}

// Logger holds configuration for logger registration.
type Logger struct {
	ServiceName      string
	Version          string
	LogLevel         string
	LogFormat        string
	LogFormatDefault string
}

// Validate validates Logger configuration.
func (c Logger) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if c.Version == "" {
		return fmt.Errorf("version is required")
	}
	if c.LogLevel == "" {
		return fmt.Errorf("log level is required")
	}
	return nil
}

// Database holds configuration for database registration.
type Database struct {
	Name            string
	DSN             string
	MaxConns        int32
	MinConns        int32
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

// Validate validates Database configuration.
func (c Database) Validate() error {
	if c.DSN == "" {
		return fmt.Errorf("DSN is required")
	}
	if c.MaxConns <= 0 {
		return fmt.Errorf("max connections must be positive")
	}
	if c.MinConns < 0 {
		return fmt.Errorf("min connections cannot be negative")
	}
	if c.MinConns > c.MaxConns {
		return fmt.Errorf("min connections cannot be greater than max connections")
	}
	if c.ConnMaxLifetime <= 0 {
		return fmt.Errorf("connection max lifetime must be positive")
	}
	if c.PingTimeout <= 0 {
		return fmt.Errorf("ping timeout must be positive")
	}
	return nil
}

// HTTPServer holds configuration for HTTP server.
type HTTPServer struct {
	Addr            string
	WriteTimeout    time.Duration
	ReadTimeout     time.Duration
	ShutdownTimeout time.Duration
	MaxHeaderBytes  int
}

// Validate validates HTTPServer configuration.
func (c HTTPServer) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("address is required")
	}
	if c.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if c.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	if c.MaxHeaderBytes <= 0 {
		return fmt.Errorf("max header bytes must be positive")
	}
	return nil
}

// S3 holds configuration for S3 client registration.
type S3 struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
}

// Validate validates S3 configuration.
func (c S3) Validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("S3 endpoint is required")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("S3 access key is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("S3 secret key is required")
	}
	if c.BucketName == "" {
		return fmt.Errorf("S3 bucket name is required")
	}
	if c.Region == "" {
		return fmt.Errorf("S3 region is required")
	}
	return nil
}
