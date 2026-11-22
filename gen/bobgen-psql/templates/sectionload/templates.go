// Package templates embeds the code-generation templates for sectionload.
//
// Layout follows bob's template FS rules:
//   - root:  *.bob.go.tpl   → generated once per package (singleton)
//   - table: table/*.go.tpl → generated once per table, concatenated into
//     one file per table in lexical order (00_, 01_, ...)
package templates

import "embed"

//go:embed *.tpl table/*.tpl
var FS embed.FS
