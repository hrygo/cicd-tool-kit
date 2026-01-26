// Copyright 2026 CICD AI Toolkit. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package security

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// RBAC provides role-based access control.
// Implements SPEC-SEC-03: RBAC
type RBAC struct {
	mu           sync.RWMutex
	roles        map[string]*Role
	policies     map[string][]*Policy
	roleHierarchy map[string][]string // parent -> children
	defaultRole  string
}

// Policy represents an access control policy.
type Policy struct {
	ID          string
	Resource    string
	Actions     []string
	Effect      PolicyEffect
	Conditions  []Condition
}

// PolicyEffect defines whether to allow or deny.
type PolicyEffect string

const (
	EffectAllow PolicyEffect = "allow"
	EffectDeny  PolicyEffect = "deny"
)

// Condition represents a condition for policy evaluation.
type Condition struct {
	Type     string
	Key      string
	Operator string
	Value    any
}

// Role represents a user role with permissions.
type Role struct {
	Name        string
	Description string
	Permissions []string
	Inherits    []string
	Metadata    map[string]string
}

// Subject represents an entity that can be authorized.
type Subject struct {
	ID       string
	Roles    []string
	Attributes map[string]string
}

// NewRBAC creates a new RBAC instance.
func NewRBAC() *RBAC {
	rbac := &RBAC{
		roles:        make(map[string]*Role),
		policies:     make(map[string][]*Policy),
		roleHierarchy: make(map[string][]string),
		defaultRole:  "viewer",
	}

	// Initialize with default roles
	rbac.initDefaultRoles()

	return rbac
}

// initDefaultRoles initializes standard roles.
func (r *RBAC) initDefaultRoles() {
	// Viewer: read-only access
	r.AddRole(&Role{
		Name:        "viewer",
		Description: "Read-only access",
		Permissions: []string{
			"read:config",
			"read:output",
			"read:logs",
		},
	})

	// Operator: can run operations
	r.AddRole(&Role{
		Name:        "operator",
		Description: "Can execute operations",
		Inherits:    []string{"viewer"},
		Permissions: []string{
			"execute:run",
			"execute:skill",
			"write:cache",
		},
	})

	// Developer: can modify code
	r.AddRole(&Role{
		Name:        "developer",
		Description: "Can modify code and configuration",
		Inherits:    []string{"operator"},
		Permissions: []string{
			"write:code",
			"write:config",
			"delete:cache",
			"manage:skills",
		},
	})

	// Admin: full access
	r.AddRole(&Role{
		Name:        "admin",
		Description: "Full administrative access",
		Inherits:    []string{"developer"},
		Permissions: []string{
			"admin:*",
			"manage:users",
			"manage:roles",
			"manage:policies",
		},
	})

	// Build hierarchy
	r.roleHierarchy = map[string][]string{
		"viewer":    {"operator"},
		"operator":  {"developer"},
		"developer": {"admin"},
	}
}

// AddRole adds a role to the RBAC system.
func (r *RBAC) AddRole(role *Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if role.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	r.roles[role.Name] = role

	// Update hierarchy
	for _, parent := range role.Inherits {
		r.roleHierarchy[parent] = append(r.roleHierarchy[parent], role.Name)
	}

	return nil
}

// GetRole retrieves a role by name.
func (r *RBAC) GetRole(name string) (*Role, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	role, ok := r.roles[name]
	return role, ok
}

// Check checks if a subject has a specific permission.
func (r *RBAC) Check(ctx context.Context, subjectID, permission string) bool {
	subject, ok := r.getSubject(subjectID)
	if !ok {
		// Use default role
		return r.checkRole(r.defaultRole, permission)
	}

	return r.CheckSubject(ctx, subject, permission)
}

// CheckSubject checks if a subject has a permission.
func (r *RBAC) CheckSubject(ctx context.Context, subject *Subject, permission string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check explicit denies first
	for _, roleName := range subject.Roles {
		if r.checkPolicyEffect(ctx, subject, roleName, permission, EffectDeny) {
			return false
		}
	}

	// Check allows
	for _, roleName := range subject.Roles {
		if r.checkRoleWithInheritance(roleName, permission) {
			return true
		}
	}

	return false
}

// checkRole checks if a role has a permission (including inherited).
func (r *RBAC) checkRoleWithInheritance(roleName, permission string) bool {
	// Check current role
	if r.checkRole(roleName, permission) {
		return true
	}

	// Check parent roles
	role, ok := r.roles[roleName]
	if !ok {
		return false
	}

	for _, parent := range role.Inherits {
		if r.checkRoleWithInheritance(parent, permission) {
			return true
		}
	}

	return false
}

// checkRole checks if a specific role has a permission.
func (r *RBAC) checkRole(roleName, permission string) bool {
	role, ok := r.roles[roleName]
	if !ok {
		return false
	}

	for _, perm := range role.Permissions {
		if r.matchPermission(perm, permission) {
			return true
		}
	}

	return false
}

// matchPermission checks if a permission pattern matches a required permission.
// Supports wildcard matching: "admin:*" matches all admin permissions.
func (r *RBAC) matchPermission(pattern, required string) bool {
	if pattern == "*" {
		return true
	}

	if strings.HasSuffix(pattern, ":*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(required, prefix)
	}

	return pattern == required
}

// checkPolicyEffect checks if policies evaluate to a specific effect.
func (r *RBAC) checkPolicyEffect(ctx context.Context, subject *Subject, roleName, permission string, effect PolicyEffect) bool {
	policies, ok := r.policies[roleName]
	if !ok {
		return false
	}

	for _, policy := range policies {
		if policy.Effect != effect {
			continue
		}

		// Check resource/action match
		if !r.matchPermission(policy.Resource, permission) {
			continue
		}

		// Evaluate conditions
		if r.evaluateConditions(ctx, subject, policy.Conditions) {
			return true
		}
	}

	return false
}

// evaluateConditions evaluates policy conditions.
func (r *RBAC) evaluateConditions(ctx context.Context, subject *Subject, conditions []Condition) bool {
	if len(conditions) == 0 {
		return true
	}

	for _, cond := range conditions {
		if !r.evaluateCondition(subject, cond) {
			return false
		}
	}

	return true
}

// evaluateCondition evaluates a single condition.
func (r *RBAC) evaluateCondition(subject *Subject, cond Condition) bool {
	attrValue, ok := subject.Attributes[cond.Key]
	if !ok {
		return false
	}

	switch cond.Operator {
	case "=", "eq":
		return fmt.Sprintf("%v", attrValue) == fmt.Sprintf("%v", cond.Value)
	case "!=", "ne":
		return fmt.Sprintf("%v", attrValue) != fmt.Sprintf("%v", cond.Value)
	case "in":
		if values, ok := cond.Value.([]string); ok {
			for _, v := range values {
				if attrValue == v {
					return true
				}
			}
			return false
		}
		return false
	default:
		return false
	}
}

// AddPolicy adds a policy to a role.
func (r *RBAC) AddPolicy(roleName string, policy *Policy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.roles[roleName]; !ok {
		return fmt.Errorf("role not found: %s", roleName)
	}

	r.policies[roleName] = append(r.policies[roleName], policy)
	return nil
}

// AddPolicy adds a permission policy (compatibility method).
func (r *RBAC) AddPolicySimple(role, permission string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if rObj, ok := r.roles[role]; ok {
		rObj.Permissions = append(rObj.Permissions, permission)
	}
}

// getSubject retrieves a subject from storage.
func (r *RBAC) getSubject(id string) (*Subject, bool) {
	// In production, this would query a database
	// For now, return a default subject
	if id == "" {
		return nil, false
	}

	return &Subject{
		ID:     id,
		Roles:  []string{"viewer"},
		Attributes: make(map[string]string),
	}, true
}

// SetDefaultRole sets the default role for unauthenticated users.
func (r *RBAC) SetDefaultRole(role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.roles[role]; !ok {
		return fmt.Errorf("role not found: %s", role)
	}

	r.defaultRole = role
	return nil
}

// Enforce enforces RBAC check, returning an error if denied.
func (r *RBAC) Enforce(ctx context.Context, subjectID, permission string) error {
	if !r.Check(ctx, subjectID, permission) {
		return &AccessDeniedError{
			SubjectID:  subjectID,
			Permission: permission,
		}
	}
	return nil
}

// EnforceSubject enforces RBAC check for a subject.
func (r *RBAC) EnforceSubject(ctx context.Context, subject *Subject, permission string) error {
	if !r.CheckSubject(ctx, subject, permission) {
		return &AccessDeniedError{
			SubjectID:  subject.ID,
			Permission: permission,
		}
	}
	return nil
}

// AccessDeniedError represents an access denied error.
type AccessDeniedError struct {
	SubjectID  string
	Permission string
}

func (e *AccessDeniedError) Error() string {
	return fmt.Sprintf("access denied: subject %s does not have permission %s", e.SubjectID, e.Permission)
}

// PermissionBuilder helps build permission strings.
type PermissionBuilder struct {
	resource string
	action   string
}

// NewPermissionBuilder creates a new permission builder.
func NewPermissionBuilder() *PermissionBuilder {
	return &PermissionBuilder{}
}

// Resource sets the resource.
func (b *PermissionBuilder) Resource(resource string) *PermissionBuilder {
	b.resource = resource
	return b
}

// Action sets the action.
func (b *PermissionBuilder) Action(action string) *PermissionBuilder {
	b.action = action
	return b
}

// Build builds the permission string.
func (b *PermissionBuilder) Build() string {
	return b.resource + ":" + b.action
}

// Parse parses a permission string.
func Parse(permission string) (resource, action string) {
	parts := strings.SplitN(permission, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "*"
}
