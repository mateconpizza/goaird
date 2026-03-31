package application

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/mateconpizza/goairdrop/internal/hook"
	"github.com/mateconpizza/goairdrop/internal/server/middleware"
)

func genCurl(h *hook.Hook, baseURL string) string {
	url := strings.TrimRight(baseURL, "/") + h.Endpoint

	var b strings.Builder

	b.WriteString("curl -X ")
	b.WriteString(h.Method + " ")
	b.WriteString(url)
	b.WriteString(" \\\n")

	b.WriteString(fmt.Sprintf(`  -H "%s: $HOSTNAME" \`+"\n", middleware.HeaderDevice))
	b.WriteString(fmt.Sprintf(`  -H "%s: $TOKEN" \`+"\n", middleware.HeaderToken))

	switch h.Type {
	case hook.TypeCommand:
		b.WriteString(`  -H "Content-Type: application/json" \` + "\n")

		payload := map[string]string{}

		// infer payload fields from template args
		for _, arg := range h.CommandTemplate.Args {
			extractTemplateVars(arg, payload)
		}

		// add default action if defined
		if len(h.AllowedActions) > 0 {
			payload["action"] = h.AllowedActions[0]
		}

		jsonBody, _ := json.Marshal(payload)

		b.WriteString("  -d '")
		b.Write(jsonBody)
		b.WriteString("'")

	case hook.TypeUpload:
		b.WriteString(`  -H "Content-Type: multipart/form-data" \` + "\n")

		// Default field name heuristic
		field := "file[]"
		if h.Name != "" && strings.Contains(h.Name, "images") {
			field = "files[]"
		}

		b.WriteString(`  -F "`)
		b.WriteString(field)
		b.WriteString(`=@./file"`)

	default:
		b.WriteString("# unsupported hook type")
	}

	return b.String()
}

func extractTemplateVars(s string, out map[string]string) {
	re := regexp.MustCompile(`\{\{\s*payload\.([a-zA-Z0-9_]+)\s*\}\}`)
	matches := re.FindAllStringSubmatch(s, -1)
	for _, m := range matches {
		key := m[1]
		if _, exists := out[key]; !exists {
			out[key] = "string"
		}
	}
}
