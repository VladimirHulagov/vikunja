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
	"bytes"
	templatehtml "html/template"
	"strings"

	"github.com/yuin/goldmark"
)

func ConvertMarkdownToHTML(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return input
	}
	if strings.HasPrefix(trimmed, "<") {
		return input
	}

	md := []byte(templatehtml.HTMLEscapeString(input))
	var buf bytes.Buffer
	if err := goldmark.Convert(md, &buf); err != nil {
		return input
	}
	//#nosec G203 -- the html is escaped above before conversion
	return buf.String()
}
