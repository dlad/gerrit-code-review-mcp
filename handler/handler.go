package handler

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/andygrunwald/go-gerrit"
	"github.com/mark3labs/mcp-go/mcp"
)

// GerritClient defines the interface for Gerrit operations needed by the handler
type GerritClient interface {
	GetChange(ctx context.Context, changeID string, opt *gerrit.ChangeOptions) (*gerrit.ChangeInfo, *gerrit.Response, error)
	GetPatch(ctx context.Context, changeID, revisionID string, opt *gerrit.PatchOptions) (*string, *gerrit.Response, error)
}

// GerritClientAdapter adapts the go-gerrit client to implement GerritClient interface
type GerritClientAdapter struct {
	client *gerrit.Client
}

// NewGerritClientAdapter creates a new adapter for the go-gerrit client
func NewGerritClientAdapter(client *gerrit.Client) *GerritClientAdapter {
	return &GerritClientAdapter{client: client}
}

// GetChange implements GerritClient interface
func (a *GerritClientAdapter) GetChange(ctx context.Context, changeID string, opt *gerrit.ChangeOptions) (*gerrit.ChangeInfo, *gerrit.Response, error) {
	return a.client.Changes.GetChange(ctx, changeID, opt)
}

// GetPatch implements GerritClient interface
func (a *GerritClientAdapter) GetPatch(ctx context.Context, changeID, revisionID string, opt *gerrit.PatchOptions) (*string, *gerrit.Response, error) {
	return a.client.Changes.GetPatch(ctx, changeID, revisionID, opt)
}

type Handler struct {
	client GerritClient
}

func NewHandler(client GerritClient) *Handler {
	h := Handler{
		client: client,
	}
	return &h
}

// extractChangeID extracts the change ID from a Gerrit change URL
func extractChangeID(url string) (string, error) {
	// Handle different Gerrit URL formats:
	// https://gerrit-review.googlesource.com/c/project/+/12345
	// https://gerrit.example.com/c/project/+/12345/
	// https://gerrit.example.com/#/c/12345/

	// Pattern for modern Gerrit URLs: /c/project/+/changeID (handles nested paths and query params)
	re1 := regexp.MustCompile(`/c/(?:[^/]+/)*\+/(\d+)(?:[?&#]|$|/)`)
	if matches := re1.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1], nil
	}

	// Pattern for legacy Gerrit URLs: #/c/changeID
	re2 := regexp.MustCompile(`#/c/(\d+)`)
	if matches := re2.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1], nil
	}

	// If URL doesn't match patterns, try to extract just the number at the end
	parts := strings.Split(strings.TrimSuffix(url, "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if matched, _ := regexp.MatchString(`^\d+$`, parts[i]); matched {
			return parts[i], nil
		}
	}

	return "", fmt.Errorf("could not extract change ID from URL: %s", url)
}

// GetGerritChangePatch fetches the patch for the latest patchset for a gerrit change
func (h *Handler) GetGerritChangePatch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	changeURL, err := request.RequireString("change_url")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Extract change ID from URL
	changeID, err := extractChangeID(changeURL)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to parse change URL: %v", err)), nil
	}

	// Fetch change details with revisions
	opt := &gerrit.ChangeOptions{
		AdditionalFields: []string{"CURRENT_REVISION", "CURRENT_COMMIT"},
	}
	change, _, err := h.client.GetChange(ctx, changeID, opt)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get change %s: %v", changeID, err)), nil
	}

	// Get the current revision ID
	if change.CurrentRevision == "" {
		return mcp.NewToolResultError("no current revision found for change"), nil
	}

	// Get the patch for the current revision
	patch, _, err := h.client.GetPatch(ctx, changeID, change.CurrentRevision, &gerrit.PatchOptions{})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get patch for change %s: %v", changeID, err)), nil
	}

	if patch == nil {
		return mcp.NewToolResultError("received nil patch content"), nil
	}

	p := *patch

	// limit size of patch
	n := 32000
	r := []rune(p)
	if len(r) > n {
		p = fmt.Sprintf("WARNING: This patch has been truncated as it is very big:\n%s", string(r[:n]))
	}

	return mcp.NewToolResultText(string(p)), nil
}
