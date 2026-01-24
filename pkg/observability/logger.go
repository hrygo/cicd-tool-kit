// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

// Package observability provides logging and metrics.
package observability

// Logger is the structured logger interface.
// This will be fully implemented in SPEC-OPS-01.
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	With(fields ...Field) Logger
}

// Field represents a log field.
type Field struct {
	Key   string
	Value any
}

// logger is the default implementation.
type logger struct {
	// TODO: Add underlying logger (zap, etc.)
}

// NewLogger creates a new logger.
func NewLogger(level string) Logger {
	return &logger{}
}

func (l *logger) Debug(msg string, fields ...Field) {
	// TODO: Implement per SPEC-OPS-01
}

func (l *logger) Info(msg string, fields ...Field) {
	// TODO: Implement per SPEC-OPS-01
}

func (l *logger) Warn(msg string, fields ...Field) {
	// TODO: Implement per SPEC-OPS-01
}

func (l *logger) Error(msg string, fields ...Field) {
	// TODO: Implement per SPEC-OPS-01
}

func (l *logger) With(fields ...Field) Logger {
	// TODO: Implement per SPEC-OPS-01
	return l
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Err creates an error field.
func Err(err error) Field {
	return Field{Key: "error", Value: err}
}
