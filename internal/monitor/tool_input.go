package monitor

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ExtractToolInput extracts a human-readable summary from tool input JSON.
// Works for both Claude (PascalCase) and OpenCode (lowercase) tool names.
func ExtractToolInput(toolName string, inputJSON json.RawMessage) string {
	if inputJSON == nil {
		return ""
	}

	var input map[string]json.RawMessage
	if err := json.Unmarshal(inputJSON, &input); err != nil {
		return ""
	}

	switch strings.ToLower(toolName) {
	case "read":
		return jsonString(input["file_path"])
	case "write":
		return jsonString(input["file_path"])
	case "edit":
		return jsonString(input["file_path"])
	case "multiedit":
		return jsonString(input["filePath"])
	case "bash":
		cmd := jsonString(input["command"])
		if len(cmd) > 100 {
			cmd = cmd[:100] + "..."
		}
		return cmd
	case "grep":
		return jsonString(input["pattern"])
	case "glob":
		return jsonString(input["pattern"])
	case "list":
		return jsonString(input["path"])
	case "task":
		return jsonString(input["description"])
	case "webfetch":
		return jsonString(input["url"])
	case "websearch", "codesearch":
		return jsonString(input["query"])
	case "question":
		return extractQuestionSummary(input["questions"])
	case "todoread":
		return ""
	case "todowrite":
		return extractTodoSummary(input["todos"])
	case "batch":
		return extractBatchSummary(input["tool_calls"])
	case "apply_patch":
		patch := jsonString(input["patchText"])
		if len(patch) > 80 {
			patch = patch[:80] + "..."
		}
		return patch
	case "askuserquestion":
		return "interactive"
	case "exitplanmode", "plan_exit":
		return "plan"
	case "skill":
		return jsonString(input["skill"])
	default:
		return ""
	}
}

// extractQuestionSummary extracts the first question text from a questions array.
func extractQuestionSummary(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var questions []struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal(raw, &questions); err != nil {
		return ""
	}
	if len(questions) == 0 {
		return ""
	}
	q := questions[0].Question
	if len(q) > 80 {
		q = q[:80] + "..."
	}
	if len(questions) > 1 {
		q += fmt.Sprintf(" (+%d more)", len(questions)-1)
	}
	return q
}

// extractTodoSummary summarizes a todos array.
func extractTodoSummary(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var todos []json.RawMessage
	if err := json.Unmarshal(raw, &todos); err != nil {
		return ""
	}
	return fmt.Sprintf("%d items", len(todos))
}

// extractBatchSummary summarizes a batch tool_calls array.
func extractBatchSummary(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var calls []struct {
		Tool string `json:"tool"`
	}
	if err := json.Unmarshal(raw, &calls); err != nil {
		return ""
	}
	names := make([]string, 0, len(calls))
	for _, c := range calls {
		if c.Tool != "" {
			names = append(names, c.Tool)
		}
	}
	if len(names) == 0 {
		return fmt.Sprintf("%d calls", len(calls))
	}
	return strings.Join(names, ", ")
}
