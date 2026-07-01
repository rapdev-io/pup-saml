package pupapi

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps `pup api` for authenticated Datadog API calls.
// Auth is fully delegated to pup — no credentials needed in the extension binary.
type Client struct {
	org string
}

func New(org string) *Client {
	return &Client{org: org}
}

func (c *Client) GetRaw(path string) ([]byte, error) {
	args := []string{"api", path, "--no-agent"}
	if c.org != "" {
		args = append(args, "--org", c.org)
	}
	cmd := exec.Command("pup", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s", strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func (c *Client) Get(path string, v any) error {
	out, err := c.GetRaw(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(out, v)
}

// Paginate fetches all pages of a v2 list endpoint via repeated pup api calls.
func (c *Client) Paginate(endpoint string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	pageNum := 0
	for {
		sep := "&"
		if !strings.Contains(endpoint, "?") {
			sep = "?"
		}
		path := fmt.Sprintf("%s%spage[size]=100&page[number]=%d", endpoint, sep, pageNum)
		var page struct {
			Data []json.RawMessage `json:"data"`
			Meta struct {
				Page struct {
					TotalCount int `json:"total_count"`
				} `json:"page"`
			} `json:"meta"`
		}
		if err := c.Get(path, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if len(all) >= page.Meta.Page.TotalCount || len(page.Data) == 0 {
			break
		}
		pageNum++
	}
	return all, nil
}
