package pupapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Client makes authenticated requests to the Datadog API using credentials
// forwarded by pup via environment variables.
type Client struct {
	baseURL string
	http    *http.Client
	token   string
	apiKey  string
	appKey  string
}

func New() *Client {
	site := os.Getenv("DD_SITE")
	if site == "" {
		site = "datadoghq.com"
	}
	return &Client{
		baseURL: fmt.Sprintf("https://api.%s", site),
		http:    &http.Client{},
		token:   os.Getenv("DD_ACCESS_TOKEN"),
		apiKey:  os.Getenv("DD_API_KEY"),
		appKey:  os.Getenv("DD_APP_KEY"),
	}
}

func (c *Client) GetRaw(path string) ([]byte, int, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/"+path, nil)
	if err != nil {
		return nil, 0, err
	}
	// v1 endpoints require API key + App key; they reject OAuth bearer tokens.
	// v2 endpoints accept either, but prefer bearer when available.
	if strings.HasPrefix(path, "v1/") || c.token == "" {
		req.Header.Set("DD-API-KEY", c.apiKey)
		req.Header.Set("DD-APPLICATION-KEY", c.appKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

func (c *Client) Get(path string, v any) error {
	body, status, err := c.GetRaw(path)
	if err != nil {
		return err
	}
	if status == 401 {
		if strings.HasPrefix(path, "v1/") {
			return fmt.Errorf("HTTP 401 for %s: v1 endpoints require DD_API_KEY + DD_APP_KEY (not available in this session)", path)
		}
		return fmt.Errorf("HTTP 401 for %s: OAuth token lacks required scope (user_access_read)", path)
	}
	if status == 403 {
		return fmt.Errorf("HTTP 403 for %s: insufficient permissions", path)
	}
	if status >= 400 {
		return fmt.Errorf("HTTP %d for %s", status, path)
	}
	return json.Unmarshal(body, v)
}

// Paginate fetches all pages of a v2 list endpoint and appends raw data items.
// The endpoint must accept page[size] and page[number] query params.
func (c *Client) Paginate(endpoint string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	pageNum := 0
	for {
		sep := "?"
		if strings.Contains(endpoint, "?") {
			sep = "&"
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
