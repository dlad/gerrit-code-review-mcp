package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/andygrunwald/go-gerrit"
	"github.com/lad/gerrit-code-review-mcp/handler"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	ctx := context.Background()

	baseURL := os.Getenv("GERRIT_BASE_URL")
	username := os.Getenv("GERRIT_USERNAME")
	password := os.Getenv("GERRIT_PASSWORD")

	if baseURL == "" {
		log.Fatal("GERRIT_BASE_URL environment variable is required")
	}

	client, err := gerrit.NewClient(ctx, baseURL, nil)
	if err != nil {
		log.Fatalf("Failed to create Gerrit client: %v", err)
	}

	if len(username) > 0 {
		err = setAuth(ctx, client, username, password)
		if err != nil {
			log.Fatalf("Could not authenticate against gerrit with user %s: %v", username, err)
		}
		log.Println("Gerrit client successfully authenticated and ready")
	}

	gerritAdapter := handler.NewGerritClientAdapter(client)
	h := handler.NewHandler(gerritAdapter)
	h.GetGerritChangePatch(ctx, mcp.CallToolRequest{})

	s := server.NewMCPServer(
		"Gerrit Code Review",
		"0.0.0",
		server.WithRecovery(),
	)

	getGerritChangeTool := mcp.NewTool("get-gerrit-change",
		mcp.WithDescription("Get Gerrit change"),
		mcp.WithString("change_url",
			mcp.Required(),
			mcp.Description("URL of Gerrit change"),
		),
	)

	s.AddTool(getGerritChangeTool, h.GetGerritChangePatch)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// checkAuth is used to check if the current credentials are valid.
// If the response is 401 Unauthorized then the error will be discarded.
// Copied from https://github.com/andygrunwald/go-gerrit/blob/650ad12c8718fc7b18463001cb54ec8593ea5045/gerrit.go#L193
func checkAuth(ctx context.Context, client *gerrit.Client) (bool, error) {
	_, response, err := client.Accounts.GetAccount(ctx, "self")
	switch err {
	case gerrit.ErrWWWAuthenticateHeaderMissing:
		return false, nil
	case gerrit.ErrWWWAuthenticateHeaderNotDigest:
		return false, nil
	default:
		// Response could be nil if the connection outright failed
		// or some other error occurred before we got a response.
		if response == nil && err != nil {
			return false, err
		}

		if err != nil && response.StatusCode == http.StatusUnauthorized {
			err = nil
		}
		return response.StatusCode == http.StatusOK, err
	}
}

// setAuth is used to set the appropriate Gerrit authentication method.
// Copied from https://github.com/andygrunwald/go-gerrit/blob/650ad12c8718fc7b18463001cb54ec8593ea5045/gerrit.go#L165
func setAuth(ctx context.Context, c *gerrit.Client, username, password string) error {
	// Digest auth (first since that's the default auth type)
	c.Authentication.SetDigestAuth(username, password)
	if success, err := checkAuth(ctx, c); success || err != nil {
		return err
	}

	// Basic auth
	c.Authentication.SetBasicAuth(username, password)
	if success, err := checkAuth(ctx, c); success || err != nil {
		return err
	}

	// Cookie auth
	c.Authentication.SetCookieAuth(username, password)
	if success, err := checkAuth(ctx, c); success || err != nil {
		return err
	}

	// Reset auth in case the consumer needs to do something special.
	c.Authentication.ResetAuth()
	return gerrit.ErrAuthenticationFailed
}
