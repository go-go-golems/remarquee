// Package rmdoc provides parsing utilities for reMarkable `.rmdoc` archives.
//
// A `.rmdoc` file is a zip archive containing:
// - `<docUUID>.content`  (JSON, legacy or cPages schema)
// - `<docUUID>.metadata` (JSON)
// - `<docUUID>.pagedata` (text, one template name per line)
// - `<docUUID>.pdf`      (optional payload for PDF documents)
// - `<docUUID>/<pageID>.rm` (annotation files; pageID can be numeric, uuid, etc.)
//
// This package focuses on:
// - opening the zip container
// - detecting the `.content` schema (legacy vs cPages)
// - building a deterministic page plan (`[]PageRef`) from `.content` (+ `.pagedata`)
package rmdoc
