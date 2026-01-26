// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package doc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// Generator generates documentation from code and specs.
// Implements SPEC-LIB-03: Doc Generator
type Generator struct {
	config *Config
}

// Config defines generator configuration.
type Config struct {
	OutputDir   string
	TemplateDir string
	Format      string // markdown, html, json
	IncludeSrc  bool
}

// NewGenerator creates a new documentation generator.
func NewGenerator(config *Config) *Generator {
	if config == nil {
		config = &Config{
			OutputDir: "./docs",
			Format:   "markdown",
		}
	}
	return &Generator{config: config}
}

// Generate generates documentation for a package.
func (g *Generator) Generate(pkgPath string) error {
	spec, err := g.analyzePackage(pkgPath)
	if err != nil {
		return err
	}

	return g.writeDoc(spec)
}

// PackageSpec represents a package documentation spec.
type PackageSpec struct {
	Name        string
	Description string
	Types       []TypeDoc
	Functions   []FuncDoc
	Constants   []ConstDoc
	Examples    []Example
}

// TypeDoc documents a type.
type TypeDoc struct {
	Name    string
	Comment string
	Fields  []FieldDoc
	Methods []FuncDoc
}

// FuncDoc documents a function.
type FuncDoc struct {
	Name       string
	Comment    string
	Signature  string
	Params     []ParamDoc
	Returns    []ParamDoc
	Examples   []string
}

// FieldDoc documents a struct field.
type FieldDoc struct {
	Name    string
	Type    string
	Comment string
}

// ParamDoc documents a parameter.
type ParamDoc struct {
	Name string
	Type string
}

// ConstDoc documents a constant.
type ConstDoc struct {
	Name    string
	Value   string
	Type    string
	Comment string
}

// Example documents an example.
type Example struct {
	Code     string
	Expected string
}

// analyzePackage analyzes a Go package.
func (g *Generator) analyzePackage(pkgPath string) (*PackageSpec, error) {
	// Placeholder implementation
	// In production, this would use go/ast or go/parser
	return &PackageSpec{
		Name:        filepath.Base(pkgPath),
		Description: "Package documentation",
	}, nil
}

// writeDoc writes documentation to output.
func (g *Generator) writeDoc(spec *PackageSpec) error {
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return err
	}

	outputPath := filepath.Join(g.config.OutputDir, spec.Name+".md")
	
	var content strings.Builder
	content.WriteString("# " + spec.Name + "\n\n")
	content.WriteString(spec.Description + "\n\n")

	if err := os.WriteFile(outputPath, []byte(content.String()), 0644); err != nil {
		return err
	}

	return nil
}

// GenerateFromTemplate generates docs from a template.
func (g *Generator) GenerateFromTemplate(spec *PackageSpec, tmplPath string) (string, error) {
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// MarkdownRenderer renders markdown documentation.
type MarkdownRenderer struct {
	builder strings.Builder
}

// NewMarkdownRenderer creates a new markdown renderer.
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// Render renders the spec to markdown.
func (r *MarkdownRenderer) Render(spec *PackageSpec) string {
	r.builder.Reset()
	r.writeHeader(spec.Name, 1)
	r.writeRaw(spec.Description)
	r.writeNewline()
	
	for _, t := range spec.Types {
		r.writeType(t)
	}
	
	for _, f := range spec.Functions {
		r.writeFunction(f)
	}
	
	return r.builder.String()
}

func (r *MarkdownRenderer) writeHeader(text string, level int) {
	r.builder.WriteString(strings.Repeat("#", level))
	r.builder.WriteString(" ")
	r.builder.WriteString(text)
	r.builder.WriteString("\n\n")
}

func (r *MarkdownRenderer) writeRaw(text string) {
	r.builder.WriteString(text)
	r.builder.WriteString("\n")
}

func (r *MarkdownRenderer) writeNewline() {
	r.builder.WriteString("\n")
}

func (r *MarkdownRenderer) writeType(t TypeDoc) {
	r.writeHeader(t.Name+" (type)", 2)
	if t.Comment != "" {
		r.writeRaw(t.Comment)
		r.writeNewline()
	}
	
	if len(t.Fields) > 0 {
		r.writeRaw("| Field | Type | Description |")
		r.writeRaw("|-------|------|-------------|")
		for _, f := range t.Fields {
			r.builder.WriteString("| ")
			r.builder.WriteString(f.Name)
			r.builder.WriteString(" | ")
			r.builder.WriteString(f.Type)
			r.builder.WriteString(" | ")
			r.builder.WriteString(f.Comment)
			r.builder.WriteString(" |\n")
		}
		r.writeNewline()
	}
}

func (r *MarkdownRenderer) writeFunction(f FuncDoc) {
	r.writeHeader(f.Name, 3)
	r.writeRaw("```go")
	r.writeRaw(f.Signature)
	r.writeRaw("```")
	r.writeNewline()
	
	if f.Comment != "" {
		r.writeRaw(f.Comment)
		r.writeNewline()
	}
}

// APIDoc represents API documentation.
type APIDoc struct {
	Endpoint   string
	Method     string
	Params     map[string]string
	Responses  map[int]string
	Example    string
}

// GenerateAPIDoc generates API documentation.
func (g *Generator) GenerateAPIDoc(api *APIDoc) (string, error) {
	var buf strings.Builder
	
	buf.WriteString(fmt.Sprintf("## %s %s\n\n", api.Method, api.Endpoint))
	
	if len(api.Params) > 0 {
		buf.WriteString("### Parameters\n\n")
		for k, v := range api.Params {
			buf.WriteString(fmt.Sprintf("- `%s`: %s\n", k, v))
		}
		buf.WriteString("\n")
	}
	
	if len(api.Responses) > 0 {
		buf.WriteString("### Responses\n\n")
		for code, desc := range api.Responses {
			buf.WriteString(fmt.Sprintf("- **%d**: %s\n", code, desc))
		}
		buf.WriteString("\n")
	}
	
	if api.Example != "" {
		buf.WriteString("### Example\n\n")
		buf.WriteString("```")
		buf.WriteString(api.Example)
		buf.WriteString("```\n\n")
	}
	
	return buf.String(), nil
}
