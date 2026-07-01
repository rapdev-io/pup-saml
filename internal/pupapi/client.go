package pupapi

import (
	"encoding/json"
	"fmt"
	"os"
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

// pupEnv returns the current environment with pup-forwarded DD credentials stripped.
// When pup runs an extension it injects DD_ACCESS_TOKEN etc. into the env; if those
// vars are present when we spawn a child `pup api` process, pup uses them directly
// instead of the stored session for --org, causing 401s. Stripping them forces the
// child pup to resolve auth from its own session store.
func pupEnv() []string {
	strip := map[string]bool{
		"DD_ACCESS_TOKEN": true,
		"DD_API_KEY":      true,
		"DD_APP_KEY":      true,
	}
	var env []string
	for _, e := range os.Environ() {
		key := e
		if i := strings.IndexByte(e, '='); i >= 0 {
			key = e[:i]
		}
		if !strip[key] {
			env = append(env, e)
		}
	}
	return env
}

func (c *Client) GetRaw(path string) ([]byte, error) {
	args := []string{"api", path, "--no-agent"}
	if c.org != "" {
		args = append(args, "--org", c.org)
	}
	cmd := exec.Command("pup", args...)
	cmd.Env = pupEnv()
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
