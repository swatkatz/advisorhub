package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/swatkatz/advisorhub/backend/internal/client"
)

// StubEnhancer writes deterministic strings for testing.
type StubEnhancer struct{}

func (e *StubEnhancer) Enhance(_ context.Context, alert *Alert) error {
	alert.Summary = fmt.Sprintf("enhanced:%s", alert.ID)
	if NeedsDraft(alert.Category) {
		dm := fmt.Sprintf("draft:%s", alert.ID)
		alert.DraftMessage = &dm
	}
	return nil
}

// ClaudeEnhancer calls the Anthropic API for natural language enhancement.
type ClaudeEnhancer struct {
	Clients client.ClientRepository
	Notes   client.AdvisorNoteRepository
	APIKey  string
}

func (e *ClaudeEnhancer) Enhance(ctx context.Context, alert *Alert) error {
	c, err := e.Clients.GetClient(ctx, alert.ClientID)
	if err != nil {
		return fmt.Errorf("getting client for enhancement: %w", err)
	}

	notes, _ := e.Notes.GetNotes(ctx, c.ID, c.AdvisorID)

	prompt := buildPrompt(alert, c, notes)

	summary, draftMsg, err := e.callClaude(ctx, prompt)
	if err != nil {
		return fmt.Errorf("calling Claude API: %w", err)
	}

	alert.Summary = summary
	if draftMsg != "" {
		alert.DraftMessage = &draftMsg
	}
	return nil
}

func buildPrompt(a *Alert, c *client.Client, notes []client.AdvisorNote) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are an AI assistant for a financial advisor. Generate a brief, actionable summary for the following alert.\n\n")
	fmt.Fprintf(&b, "Client: %s\nAlert Category: %s\nSeverity: %s\nAlert Data: %s\n\n",
		c.Name, a.Category, a.Severity, string(a.Payload))

	if len(notes) > 0 {
		b.WriteString("Recent advisor notes:\n")
		limit := len(notes)
		if limit > 3 {
			limit = 3
		}
		for _, n := range notes[:limit] {
			fmt.Fprintf(&b, "- %s: %s\n", n.Date.Format("2006-01-02"), n.Text)
		}
		b.WriteString("\n")
	}

	b.WriteString("Provide a 1-2 sentence summary for the advisor.")
	if NeedsDraft(a.Category) {
		b.WriteString(" Also provide a short, professional draft message the advisor could send to the client.")
	}
	b.WriteString("\nRespond in JSON: {\"summary\": \"...\", \"draft_message\": \"...\"}")
	return b.String()
}

func (e *ClaudeEnhancer) callClaude(ctx context.Context, prompt string) (summary, draftMsg string, err error) {
	reqBody := map[string]any{
		"model":      "claude-sonnet-4-20250514",
		"max_tokens": 500,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", e.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", "", fmt.Errorf("parsing API response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return "", "", fmt.Errorf("empty API response")
	}

	// Strip markdown code fences (```json ... ```) that Claude may wrap around the response.
	text := strings.TrimSpace(apiResp.Content[0].Text)
	if strings.HasPrefix(text, "```") {
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = text[idx+1:]
		}
		text = strings.TrimSuffix(strings.TrimSpace(text), "```")
		text = strings.TrimSpace(text)
	}

	var result struct {
		Summary      string `json:"summary"`
		DraftMessage string `json:"draft_message"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		// Fallback: use the raw text as summary
		return apiResp.Content[0].Text, "", nil
	}

	return result.Summary, result.DraftMessage, nil
}
