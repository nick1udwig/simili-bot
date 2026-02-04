package steps

import (
	"testing"

	"github.com/similigh/simili-bot/internal/core/pipeline"
)

func TestExtractSourceRepo(t *testing.T) {
	handler := &CommandHandler{}

	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "With Bold Markers",
			body:     "## 🤖 Simili Triage Report\n\nTransferred from **similigh/event-integrator-core** (90% confidence)",
			expected: "similigh/event-integrator-core",
		},
		{
			name:     "Without Bold Markers",
			body:     "Transferred from similigh/event-integrator-cli for some reason",
			expected: "similigh/event-integrator-cli",
		},
		{
			name:     "No Transfer Info",
			body:     "Just a normal comment",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.extractSourceRepo(tt.body)
			if result != tt.expected {
				t.Errorf("extractSourceRepo() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommandHandler_Run_IgnoreNonComment(t *testing.T) {
	handler := &CommandHandler{gh: nil} // No GH needed for early exit
	ctx := &pipeline.Context{
		Issue: &pipeline.Issue{
			EventType: "issues",
		},
	}

	if err := handler.Run(ctx); err != nil {
		t.Errorf("Run() error = %v", err)
	}
	// Should do nothing
}
