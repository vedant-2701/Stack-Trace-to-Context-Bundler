// Package render turns a contract.Bundle into the clipboard-ready output
// formats consumers of this tool paste into third-party AI chats.
package render

import "github.com/vedant-2701/stack-trace-bundler/internal/contract"

// Markdown renders the bundle as a single, self-contained Markdown document.
func Markdown(_ contract.Bundle) string {
	return ""
}
