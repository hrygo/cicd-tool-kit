// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");

package cache

import (
	"crypto/sha256"
	"encoding/hex"
)

// KeyGenerator generates cache keys.
// This will be fully implemented in SPEC-PERF-01.
type KeyGenerator struct {
	// TODO: Add key generation configuration
	prefix string
}

// NewKeyGenerator creates a new key generator.
func NewKeyGenerator() *KeyGenerator {
	return &KeyGenerator{
		prefix: "cicd",
	}
}

// Generate generates a cache key from inputs.
func (kg *KeyGenerator) Generate(inputs ...string) string {
	// TODO: Implement per SPEC-PERF-01
	// Use SHA256 hash of inputs
	h := sha256.New()
	for _, input := range inputs {
		h.Write([]byte(input))
	}
	return kg.prefix + ":" + hex.EncodeToString(h.Sum(nil))
}

// GenerateForSkill generates a key for skill execution.
func (kg *KeyGenerator) GenerateForSkill(skillName, diff string) string {
	// TODO: Implement per SPEC-PERF-01
	return kg.Generate(skillName, diff)
}

// CacheError represents a cache error.
type CacheError struct {
	Code string
}

func (e *CacheError) Error() string {
	return e.Code
}
