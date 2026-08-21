// Package typescript implements the 006a LanguageParser for TypeScript
// and JavaScript Node.js stack traces. Two exported LanguageParser
// values, javascriptParser and typescriptParser, share one unexported
// parse engine (engine.go) -- see plan.md's Architecture/approach
// section. This file wires them together; the actual method bodies are
// implemented task by task (T002-T010), not here in T001.
package typescript
