// Author: Kaviru Hapuarachchi
// GitHub: https://github.com/kavirubc
// Created: 2026-02-04

package steps

import (
	"fmt"
	"log"
	"strings"

	"github.com/similigh/simili-bot/internal/core/pipeline"
	"github.com/similigh/simili-bot/internal/integrations/github"
)

// CommandHandler processes bot commands like /undo.
type CommandHandler struct {
	gh *github.Client
}

// NewCommandHandler creates a new command handler step.
func NewCommandHandler(deps *pipeline.Dependencies) *CommandHandler {
	return &CommandHandler{
		gh: deps.GitHub,
	}
}

// Name returns the step name.
func (s *CommandHandler) Name() string {
	return "command_handler"
}

// Run checks for commands in issue comments.
func (s *CommandHandler) Run(ctx *pipeline.Context) error {
	// Handle specific commands in comment events
	if ctx.Issue.EventType == "issue_comment" && ctx.Issue.CommentBody != "" {
		command := strings.TrimSpace(strings.ToLower(ctx.Issue.CommentBody))
		if strings.HasPrefix(command, "/") {
			log.Printf("[command_handler] Processing command: %s", command)
			switch {
			case strings.HasPrefix(command, "/undo"):
				return s.handleUndo(ctx)
			default:
				log.Printf("[command_handler] Unknown command: %s", command)
			}
		}
		return nil
	}

	// For standard issue events, check history for undo commands to prevent loops
	if ctx.Issue.EventType == "issues" {
		return s.analyzeHistoryForLoops(ctx)
	}

	return nil
}

// analyzeHistoryForLoops checks history for undo commands and previous transfers to preventing loops
func (s *CommandHandler) analyzeHistoryForLoops(ctx *pipeline.Context) error {
	if s.gh == nil {
		return nil
	}

	// Fetch recent comments
	comments, _, err := s.gh.ListComments(ctx.Ctx, ctx.Issue.Org, ctx.Issue.Repo, ctx.Issue.Number, nil)
	if err != nil {
		log.Printf("[command_handler] Failed to list comments: %v", err)
		return nil // Non-fatal
	}

	var blockedTargets []string

	// Analyze comments
	for _, c := range comments {
		body := strings.TrimSpace(c.GetBody())

		// Check for /undo command
		if strings.EqualFold(body, "/undo") {
			log.Printf("[command_handler] Found /undo in history. Blocking auto-transfer.")
			ctx.Metadata["transfer_blocked"] = true
		}

		// Check for previous transfers (hot-potato loop prevention)
		if strings.Contains(body, "🤖 Simili Triage Report") && strings.Contains(body, "Transferred from") {
			sourceRepo := s.extractSourceRepo(body)
			if sourceRepo != "" {
				log.Printf("[command_handler] Found previous transfer from %s. Blocking return transfer.", sourceRepo)
				blockedTargets = append(blockedTargets, sourceRepo)
			}
		}
	}

	if len(blockedTargets) > 0 {
		ctx.Metadata["blocked_targets"] = blockedTargets
	}

	return nil
}

// handleUndo reverses a previous transfer.
func (s *CommandHandler) handleUndo(ctx *pipeline.Context) error {
	if s.gh == nil {
		return fmt.Errorf("GitHub client required for undo command")
	}

	log.Printf("[command_handler] Handling /undo for issue #%d", ctx.Issue.Number)

	// To undo, we need to find where this issue came from.
	// We look for Simili-Bot's triage report which contains the source info.
	comments, _, err := s.gh.ListComments(ctx.Ctx, ctx.Issue.Org, ctx.Issue.Repo, ctx.Issue.Number, nil)
	if err != nil {
		return fmt.Errorf("failed to list comments for undo: %w", err)
	}

	var sourceRepo string
	for _, c := range comments {
		body := c.GetBody()
		if strings.Contains(body, "🤖 Simili Triage Report") && strings.Contains(body, "Transferred from") {
			// Extract source repo from text like "Transferred from **similigh/event-integrator-core**"
			sourceRepo = s.extractSourceRepo(body)
			if sourceRepo != "" {
				break
			}
		}
	}

	if sourceRepo == "" {
		log.Printf("[command_handler] Could not determine source repository for /undo")
		return nil
	}

	log.Printf("[command_handler] Reversing transfer back to %s", sourceRepo)
	ctx.TransferTarget = sourceRepo
	ctx.Metadata["reverse_transfer"] = true
	ctx.Metadata["comment"] = fmt.Sprintf("🔄 **Undoing transfer.** Moving issue back to `%s` as requested by @%s.", sourceRepo, ctx.Issue.CommentAuthor)

	return nil
}

// extractSourceRepo pulls the repo name out of the triage report body.
func (s *CommandHandler) extractSourceRepo(body string) string {
	// Simple marker-based extraction
	marker := "Transferred from **"
	start := strings.Index(body, marker)
	if start == -1 {
		// Try without bolding just in case
		marker = "Transferred from "
		start = strings.Index(body, marker)
		if start == -1 {
			return ""
		}
	}
	start += len(marker)

	end := strings.Index(body[start:], "**")
	if end == -1 {
		// If no closing bolding, try looking for space or newline
		end = strings.IndexAny(body[start:], " \n")
		if end == -1 {
			end = len(body[start:])
		}
	}

	return strings.TrimSpace(body[start : start+end])
}
