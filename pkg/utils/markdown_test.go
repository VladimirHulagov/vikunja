// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertMarkdownToHTML(t *testing.T) {
	t.Run("empty string", func(t *testing.T) {
		result := ConvertMarkdownToHTML("")
		assert.Empty(t, result)
	})
	t.Run("whitespace only", func(t *testing.T) {
		result := ConvertMarkdownToHTML("   ")
		assert.Equal(t, "   ", result)
	})
	t.Run("plain text", func(t *testing.T) {
		result := ConvertMarkdownToHTML("hello world")
		assert.Equal(t, "<p>hello world</p>\n", result)
	})
	t.Run("markdown bold", func(t *testing.T) {
		result := ConvertMarkdownToHTML("**bold**")
		assert.Equal(t, "<p><strong>bold</strong></p>\n", result)
	})
	t.Run("markdown heading", func(t *testing.T) {
		result := ConvertMarkdownToHTML("# Title")
		assert.Contains(t, result, "<h1>")
		assert.Contains(t, result, "Title")
	})
	t.Run("markdown link", func(t *testing.T) {
		result := ConvertMarkdownToHTML("[link](https://example.com)")
		assert.Contains(t, result, "<a href")
		assert.Contains(t, result, "link")
	})
	t.Run("markdown list", func(t *testing.T) {
		result := ConvertMarkdownToHTML("- item 1\n- item 2")
		assert.Contains(t, result, "<ul>")
		assert.Contains(t, result, "<li>")
	})
	t.Run("html passthrough", func(t *testing.T) {
		input := "<p>already html</p>"
		result := ConvertMarkdownToHTML(input)
		assert.Equal(t, input, result)
	})
	t.Run("html with leading whitespace", func(t *testing.T) {
		input := "  <p>already html</p>"
		result := ConvertMarkdownToHTML(input)
		assert.Equal(t, input, result)
	})
	t.Run("html special chars are escaped in markdown", func(t *testing.T) {
		result := ConvertMarkdownToHTML("use <script>alert('xss')</script>")
		assert.NotContains(t, result, "<script>")
		assert.Contains(t, result, "&lt;script&gt;")
	})
	t.Run("markdown code block", func(t *testing.T) {
		result := ConvertMarkdownToHTML("```\ncode\n```")
		assert.Contains(t, result, "<code>")
	})
	t.Run("markdown inline code", func(t *testing.T) {
		result := ConvertMarkdownToHTML("`code`")
		assert.Contains(t, result, "<code>")
	})
}
