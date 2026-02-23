package adf

// ExtractText extracts plain text from an ADF document.
func ExtractText(adf map[string]interface{}) string {
	if adf == nil {
		return ""
	}

	var parts []string
	extractContent(adf, &parts)
	return joinParts(parts)
}

func extractContent(node map[string]interface{}, parts *[]string) {
	if nodeType, ok := node["type"].(string); ok && nodeType == "text" {
		if text, ok := node["text"].(string); ok {
			*parts = append(*parts, text)
		}
	}

	if content, ok := node["content"].([]interface{}); ok {
		for _, child := range content {
			if childMap, ok := child.(map[string]interface{}); ok {
				extractContent(childMap, parts)
			}
		}
	}
}

func joinParts(parts []string) string {
	result := ""
	for _, p := range parts {
		result += p
	}
	return result
}
