package rmdsl

// Package rmdsl implements a small, versioned DSL for describing reMarkable-like documents.
//
// It supports loading cases from:
// - YAML (static fixtures)
// - JavaScript generators executed in a goja VM (scriptable fixtures)
//
// The primary initial consumer is ticket tooling for programmatic PNG rendering.

