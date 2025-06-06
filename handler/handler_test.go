package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/andygrunwald/go-gerrit"
)

// MockGerritClient implements GerritClient interface for testing
type MockGerritClient struct {
	GetChangeFunc func(ctx context.Context, changeID string, opt *gerrit.ChangeOptions) (*gerrit.ChangeInfo, *gerrit.Response, error)
	GetPatchFunc  func(ctx context.Context, changeID, revisionID string, opt *gerrit.PatchOptions) (*string, *gerrit.Response, error)
}

func (m *MockGerritClient) GetChange(ctx context.Context, changeID string, opt *gerrit.ChangeOptions) (*gerrit.ChangeInfo, *gerrit.Response, error) {
	if m.GetChangeFunc != nil {
		return m.GetChangeFunc(ctx, changeID, opt)
	}
	return nil, nil, nil
}

func (m *MockGerritClient) GetPatch(ctx context.Context, changeID, revisionID string, opt *gerrit.PatchOptions) (*string, *gerrit.Response, error) {
	if m.GetPatchFunc != nil {
		return m.GetPatchFunc(ctx, changeID, revisionID, opt)
	}
	return nil, nil, nil
}

func TestNewHandler(t *testing.T) {
	// Test that we can create a handler with a mock client
	mockClient := &MockGerritClient{}
	handler := NewHandler(mockClient)

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}

	if handler.client != mockClient {
		t.Fatal("Expected handler to use the provided client")
	}
}

func TestMockGerritClient_GetChange(t *testing.T) {
	// Test that our mock can be configured to return specific values
	expectedChange := &gerrit.ChangeInfo{
		CurrentRevision: "abc123",
		ID:              "test-change",
	}

	mockClient := &MockGerritClient{
		GetChangeFunc: func(ctx context.Context, changeID string, opt *gerrit.ChangeOptions) (*gerrit.ChangeInfo, *gerrit.Response, error) {
			if changeID == "12345" {
				return expectedChange, nil, nil
			}
			return nil, nil, errors.New("change not found")
		},
	}

	// Test successful case
	change, _, err := mockClient.GetChange(context.Background(), "12345", nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if change.CurrentRevision != "abc123" {
		t.Fatalf("Expected CurrentRevision 'abc123', got: %s", change.CurrentRevision)
	}

	// Test error case
	_, _, err = mockClient.GetChange(context.Background(), "99999", nil)
	if err == nil {
		t.Fatal("Expected error for non-existent change")
	}
}

func TestMockGerritClient_GetPatch(t *testing.T) {
	// Test that our mock can return patch content
	expectedPatch := "diff --git a/file.go b/file.go\n+added line"

	mockClient := &MockGerritClient{
		GetPatchFunc: func(ctx context.Context, changeID, revisionID string, opt *gerrit.PatchOptions) (*string, *gerrit.Response, error) {
			if changeID == "12345" && revisionID == "abc123" {
				return &expectedPatch, nil, nil
			}
			return nil, nil, errors.New("patch not found")
		},
	}

	// Test successful case
	patch, _, err := mockClient.GetPatch(context.Background(), "12345", "abc123", nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if *patch != expectedPatch {
		t.Fatalf("Expected patch content '%s', got: %s", expectedPatch, *patch)
	}

	// Test error case
	_, _, err = mockClient.GetPatch(context.Background(), "99999", "xyz789", nil)
	if err == nil {
		t.Fatal("Expected error for non-existent patch")
	}
}

func TestGerritClientAdapter(t *testing.T) {
	// Test that the adapter can be created (we can't test actual Gerrit calls without a real client)
	// This test just verifies the adapter structure works
	adapter := NewGerritClientAdapter(nil)
	if adapter == nil {
		t.Fatal("Expected adapter to be created, got nil")
	}
}

func TestExtractChangeID(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedID  string
		expectError bool
	}{
		{
			name:        "modern Gerrit URL with project path",
			url:         "https://gerrit-review.googlesource.com/c/project/+/12345",
			expectedID:  "12345",
			expectError: false,
		},
		{
			name:        "modern Gerrit URL with trailing slash",
			url:         "https://gerrit.example.com/c/project/+/67890/",
			expectedID:  "67890",
			expectError: false,
		},
		{
			name:        "modern Gerrit URL with nested project path",
			url:         "https://gerrit.example.com/c/some/nested/project/+/54321",
			expectedID:  "54321",
			expectError: false,
		},
		{
			name:        "complex modern URL with query parameters",
			url:         "https://gerrit-review.googlesource.com/c/chromium/src/+/4567890?usp=review-tab",
			expectedID:  "4567890",
			expectError: false,
		},
		{
			name:        "legacy Gerrit URL format",
			url:         "https://gerrit.example.com/#/c/98765/",
			expectedID:  "98765",
			expectError: false,
		},
		{
			name:        "legacy Gerrit URL without trailing slash",
			url:         "https://gerrit.example.com/#/c/11111",
			expectedID:  "11111",
			expectError: false,
		},
		{
			name:        "URL with number at end (fallback)",
			url:         "https://example.com/some/path/22222",
			expectedID:  "22222",
			expectError: false,
		},
		{
			name:        "URL with number at end and trailing slash",
			url:         "https://example.com/some/path/33333/",
			expectedID:  "33333",
			expectError: false,
		},
		{
			name:        "URL with multiple numbers, should pick the last one",
			url:         "https://example.com/123/path/456/change/78901",
			expectedID:  "78901",
			expectError: false,
		},
		{
			name:        "URL with no numbers",
			url:         "https://example.com/no/numbers/here",
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "empty URL",
			url:         "",
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "URL with only non-numeric segments",
			url:         "https://gerrit.example.com/c/project/+/branch-name",
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "URL with mixed alphanumeric but ends with number",
			url:         "https://example.com/abc123def/456",
			expectedID:  "456",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractChangeID(tt.url)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for URL %q, but got none", tt.url)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for URL %q: %v", tt.url, err)
				return
			}

			if result != tt.expectedID {
				t.Errorf("for URL %q, expected change ID %q, got %q", tt.url, tt.expectedID, result)
			}
		})
	}
}
