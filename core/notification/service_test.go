package notification

import (
	"github.com/goastian/astiango-hub/core/entity"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTemplateVariables_WithValidTemplate_ReturnsVariables(t *testing.T) {
	svc := Service{}
	template := "Dear ${user:name}, your task ${task:id} is ${task:status}."
	expected := []entity.NotificationVariable{
		{Category: "user", Name: "name"},
		{Category: "task", Name: "id"},
		{Category: "task", Name: "status"},
	}

	variables := svc.parseTemplateVariables(template)

	// contains all expected variables
	assert.ElementsMatch(t, expected, variables)
}

func TestConvertMarkdownToHTML_SanitizesExecutableContent(t *testing.T) {
	svc := Service{}
	content := svc.convertMarkdownToHtml("# Safe\n\n<script>alert(1)</script><img src=x onerror=alert(1)>\n\n[bad](javascript:alert(1)) [good](https://example.com)")

	assert.Contains(t, content, "<h1>Safe</h1>")
	assert.Contains(t, content, `href="https://example.com"`)
	assert.NotContains(t, strings.ToLower(content), "script")
	assert.NotContains(t, strings.ToLower(content), "onerror")
	assert.NotContains(t, strings.ToLower(content), "javascript:")
	assert.NotContains(t, strings.ToLower(content), "<img")
}

func TestParseTemplateVariables_WithRepeatedVariables_ReturnsUniqueVariables(t *testing.T) {
	svc := Service{}
	template := "Dear ${user:name}, your task ${task:id} is ${task:status}. Again, ${user:name} and ${task:id}."
	expected := []entity.NotificationVariable{
		{Category: "user", Name: "name"},
		{Category: "task", Name: "id"},
		{Category: "task", Name: "status"},
	}

	variables := svc.parseTemplateVariables(template)

	// contains all expected variables
	assert.ElementsMatch(t, expected, variables)
}
