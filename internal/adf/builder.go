package adf

// ADFNode represents a node in the ADF document tree.
type ADFNode struct {
	Type    string                 `json:"type"`
	Attrs   map[string]interface{} `json:"attrs,omitempty"`
	Content []ADFNode              `json:"content,omitempty"`
	Text    string                 `json:"text,omitempty"`
	Marks   []Mark                 `json:"marks,omitempty"`
}

// Mark represents an inline formatting mark.
type Mark struct {
	Type  string                 `json:"type"`
	Attrs map[string]interface{} `json:"attrs,omitempty"`
}

// Doc creates an ADF document wrapper.
func Doc(content ...ADFNode) map[string]interface{} {
	nodes := make([]interface{}, len(content))
	for i, n := range content {
		nodes[i] = nodeToMap(n)
	}
	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": nodes,
	}
}

func nodeToMap(n ADFNode) map[string]interface{} {
	m := map[string]interface{}{
		"type": n.Type,
	}

	if len(n.Attrs) > 0 {
		m["attrs"] = n.Attrs
	}

	if n.Text != "" {
		m["text"] = n.Text
	}

	if len(n.Marks) > 0 {
		marks := make([]interface{}, len(n.Marks))
		for i, mark := range n.Marks {
			mm := map[string]interface{}{"type": mark.Type}
			if len(mark.Attrs) > 0 {
				mm["attrs"] = mark.Attrs
			}
			marks[i] = mm
		}
		m["marks"] = marks
	}

	if len(n.Content) > 0 {
		children := make([]interface{}, len(n.Content))
		for i, child := range n.Content {
			children[i] = nodeToMap(child)
		}
		m["content"] = children
	}

	return m
}

// --- Text nodes with marks ---

func TextNode(text string, marks ...Mark) ADFNode {
	return ADFNode{Type: "text", Text: text, Marks: marks}
}

func Bold(text string) ADFNode {
	return TextNode(text, Mark{Type: "strong"})
}

func Italic(text string) ADFNode {
	return TextNode(text, Mark{Type: "em"})
}

func Code(text string) ADFNode {
	return TextNode(text, Mark{Type: "code"})
}

func Strike(text string) ADFNode {
	return TextNode(text, Mark{Type: "strike"})
}

func Link(text, href string) ADFNode {
	return TextNode(text, Mark{Type: "link", Attrs: map[string]interface{}{"href": href}})
}

func HardBreak() ADFNode {
	return ADFNode{Type: "hardBreak"}
}

// --- Block nodes ---

func Paragraph(content ...ADFNode) ADFNode {
	return ADFNode{Type: "paragraph", Content: content}
}

func Heading(level int, content ...ADFNode) ADFNode {
	if level < 1 {
		level = 1
	}
	if level > 6 {
		level = 6
	}
	return ADFNode{
		Type:    "heading",
		Attrs:   map[string]interface{}{"level": level},
		Content: content,
	}
}

func BulletList(items ...ADFNode) ADFNode {
	return ADFNode{Type: "bulletList", Content: items}
}

func OrderedList(items ...ADFNode) ADFNode {
	return ADFNode{Type: "orderedList", Content: items}
}

func ListItem(content ...ADFNode) ADFNode {
	return ADFNode{Type: "listItem", Content: content}
}

func CodeBlock(text string, language string) ADFNode {
	node := ADFNode{
		Type:    "codeBlock",
		Content: []ADFNode{{Type: "text", Text: text}},
	}
	if language != "" {
		node.Attrs = map[string]interface{}{"language": language}
	}
	return node
}

func Rule() ADFNode {
	return ADFNode{Type: "rule"}
}

func Blockquote(content ...ADFNode) ADFNode {
	return ADFNode{Type: "blockquote", Content: content}
}

// --- Table nodes ---

func Table(rows ...ADFNode) ADFNode {
	return ADFNode{Type: "table", Content: rows}
}

func TableRow(cells ...ADFNode) ADFNode {
	return ADFNode{Type: "tableRow", Content: cells}
}

func TableHeader(content ...ADFNode) ADFNode {
	return ADFNode{Type: "tableHeader", Content: content}
}

func TableCell(content ...ADFNode) ADFNode {
	return ADFNode{Type: "tableCell", Content: content}
}
