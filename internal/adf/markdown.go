package adf

import (
	"regexp"
	"strings"
)

var (
	headingRe    = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	bulletRe     = regexp.MustCompile(`^[-*]\s+`)
	orderedRe    = regexp.MustCompile(`^\d+\.\s+`)
	tableRowRe   = regexp.MustCompile(`^\|.*\|$`)
	ruleRe       = regexp.MustCompile(`^(-{3,}|\*{3,}|_{3,})$`)
	separatorRe  = regexp.MustCompile(`^:?-+:?$`)

)

// MarkdownToADF converts markdown-formatted text to an ADF document.
func MarkdownToADF(text string) map[string]interface{} {
	lines := strings.Split(text, "\n")
	var content []ADFNode
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			i++
			continue
		}

		trimmed := strings.TrimSpace(line)

		// Code block (```)
		if strings.HasPrefix(trimmed, "```") {
			language := strings.TrimSpace(trimmed[3:])
			var codeLines []string
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "```") {
				codeLines = append(codeLines, lines[i])
				i++
			}
			content = append(content, CodeBlock(strings.Join(codeLines, "\n"), language))
			i++ // skip closing ```
			continue
		}

		// Heading (# through ######)
		if m := headingRe.FindStringSubmatch(line); m != nil {
			level := len(m[1])
			headingText := m[2]
			content = append(content, Heading(level, TextNode(headingText)))
			i++
			continue
		}

		// Bullet list (- or *)
		if bulletRe.MatchString(line) {
			var items []ADFNode
			for i < len(lines) && bulletRe.MatchString(lines[i]) {
				itemText := bulletRe.ReplaceAllString(lines[i], "")
				inlineNodes := parseInlineMarkdown(itemText)
				items = append(items, ListItem(Paragraph(inlineNodes...)))
				i++
			}
			content = append(content, BulletList(items...))
			continue
		}

		// Ordered list (1. 2. etc)
		if orderedRe.MatchString(line) {
			var items []ADFNode
			for i < len(lines) && orderedRe.MatchString(lines[i]) {
				itemText := orderedRe.ReplaceAllString(lines[i], "")
				inlineNodes := parseInlineMarkdown(itemText)
				items = append(items, ListItem(Paragraph(inlineNodes...)))
				i++
			}
			content = append(content, OrderedList(items...))
			continue
		}

		// Markdown table
		if tableRowRe.MatchString(trimmed) {
			var tableLines []string
			for i < len(lines) && tableRowRe.MatchString(strings.TrimSpace(lines[i])) {
				tableLines = append(tableLines, lines[i])
				i++
			}
			if len(tableLines) >= 2 {
				content = append(content, parseTable(tableLines))
			}
			continue
		}

		// Horizontal rule
		if ruleRe.MatchString(trimmed) {
			content = append(content, Rule())
			i++
			continue
		}

		// Regular paragraph — collect consecutive non-special lines
		var paraLines []string
		for i < len(lines) {
			current := lines[i]
			currentTrimmed := strings.TrimSpace(current)

			if currentTrimmed == "" ||
				headingRe.MatchString(current) ||
				bulletRe.MatchString(current) ||
				orderedRe.MatchString(current) ||
				strings.HasPrefix(currentTrimmed, "```") ||
				tableRowRe.MatchString(currentTrimmed) ||
				ruleRe.MatchString(currentTrimmed) {
				break
			}
			paraLines = append(paraLines, current)
			i++
		}

		if len(paraLines) > 0 {
			paraText := strings.Join(paraLines, "\n")
			inlineNodes := parseInlineMarkdown(paraText)
			content = append(content, Paragraph(inlineNodes...))
		}
	}

	return Doc(content...)
}

// parseInlineMarkdown parses inline formatting (bold, italic, code, links)
// and returns a slice of ADFNodes.
// Uses a manual scanner since Go's regexp doesn't support lookbehind.
func parseInlineMarkdown(text string) []ADFNode {
	var nodes []ADFNode
	i := 0
	buf := ""

	flushBuf := func() {
		if buf != "" {
			nodes = append(nodes, textWithBreaks(buf)...)
			buf = ""
		}
	}

	for i < len(text) {
		ch := text[i]

		// ** bold **
		if ch == '*' && i+1 < len(text) && text[i+1] == '*' {
			if end := strings.Index(text[i+2:], "**"); end > 0 {
				flushBuf()
				nodes = append(nodes, Bold(text[i+2:i+2+end]))
				i = i + 2 + end + 2
				continue
			}
		}

		// __ bold __
		if ch == '_' && i+1 < len(text) && text[i+1] == '_' {
			if end := strings.Index(text[i+2:], "__"); end > 0 {
				flushBuf()
				nodes = append(nodes, Bold(text[i+2:i+2+end]))
				i = i + 2 + end + 2
				continue
			}
		}

		// `code`
		if ch == '`' {
			if end := strings.Index(text[i+1:], "`"); end > 0 {
				flushBuf()
				nodes = append(nodes, Code(text[i+1:i+1+end]))
				i = i + 1 + end + 1
				continue
			}
		}

		// [text](url)
		if ch == '[' {
			closeBracket := strings.Index(text[i:], "](")
			if closeBracket > 1 {
				linkText := text[i+1 : i+closeBracket]
				rest := text[i+closeBracket+2:]
				closeParen := strings.Index(rest, ")")
				if closeParen > 0 {
					linkURL := rest[:closeParen]
					flushBuf()
					nodes = append(nodes, Link(linkText, linkURL))
					i = i + closeBracket + 2 + closeParen + 1
					continue
				}
			}
		}

		// *italic* (single *, not preceded by *)
		if ch == '*' && (i == 0 || text[i-1] != '*') && i+1 < len(text) && text[i+1] != '*' {
			// Find closing single * (not **)
			end := -1
			for j := i + 2; j < len(text); j++ {
				if text[j] == '*' && (j+1 >= len(text) || text[j+1] != '*') {
					end = j
					break
				}
			}
			if end > i+1 {
				flushBuf()
				nodes = append(nodes, Italic(text[i+1:end]))
				i = end + 1
				continue
			}
		}

		// _italic_ (single _, not preceded by _)
		if ch == '_' && (i == 0 || text[i-1] != '_') && i+1 < len(text) && text[i+1] != '_' {
			end := -1
			for j := i + 2; j < len(text); j++ {
				if text[j] == '_' && (j+1 >= len(text) || text[j+1] != '_') {
					end = j
					break
				}
			}
			if end > i+1 {
				flushBuf()
				nodes = append(nodes, Italic(text[i+1:end]))
				i = end + 1
				continue
			}
		}

		buf += string(ch)
		i++
	}

	flushBuf()

	if len(nodes) == 0 {
		nodes = textWithBreaks(text)
	}

	return nodes
}

// textWithBreaks converts text with newlines to text nodes with hardBreaks.
func textWithBreaks(text string) []ADFNode {
	var nodes []ADFNode
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		if part != "" {
			nodes = append(nodes, TextNode(part))
		}
		if i < len(parts)-1 {
			nodes = append(nodes, HardBreak())
		}
	}
	return nodes
}

// parseTable parses markdown table lines into an ADF table node.
func parseTable(tableLines []string) ADFNode {
	parseRow := func(line string) []string {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|") {
			line = line[1:]
		}
		if strings.HasSuffix(line, "|") {
			line = line[:len(line)-1]
		}
		cells := strings.Split(line, "|")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		return cells
	}

	isSeparator := func(line string) bool {
		cells := parseRow(line)
		for _, cell := range cells {
			cell = strings.TrimSpace(cell)
			if cell == "" {
				continue
			}
			if !separatorRe.MatchString(cell) {
				return false
			}
		}
		return true
	}

	// Parse header row
	headers := parseRow(tableLines[0])

	// Find where data rows start (skip separator)
	dataStart := 1
	if len(tableLines) > 1 && isSeparator(tableLines[1]) {
		dataStart = 2
	}

	// Build header row
	var headerCells []ADFNode
	for _, h := range headers {
		inlineNodes := parseInlineMarkdown(h)
		headerCells = append(headerCells, TableHeader(Paragraph(inlineNodes...)))
	}
	headerRow := TableRow(headerCells...)

	// Build data rows
	var dataRows []ADFNode
	for _, line := range tableLines[dataStart:] {
		if isSeparator(line) {
			continue
		}
		cells := parseRow(line)
		// Pad to match header count
		for len(cells) < len(headers) {
			cells = append(cells, "")
		}
		cells = cells[:len(headers)]

		var tableCells []ADFNode
		for _, cell := range cells {
			inlineNodes := parseInlineMarkdown(cell)
			tableCells = append(tableCells, TableCell(Paragraph(inlineNodes...)))
		}
		dataRows = append(dataRows, TableRow(tableCells...))
	}

	allRows := append([]ADFNode{headerRow}, dataRows...)
	return Table(allRows...)
}
