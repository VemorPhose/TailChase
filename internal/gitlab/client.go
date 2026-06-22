package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Job struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	WebURL string `json:"web_url"`
}

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func TokenFromEnv() string {
	return strings.TrimSpace(os.Getenv("GITLAB_TOKEN"))
}

func NewClient(baseURL string, token string) Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://gitlab.com"
	}
	return Client{
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		Token:   strings.TrimSpace(token),
	}
}

func (c Client) ListPipelineJobs(ctx context.Context, project string, pipelineID int64, failedOnly bool) ([]Job, error) {
	query := url.Values{}
	query.Set("per_page", "100")
	if failedOnly {
		query.Add("scope[]", "failed")
	}
	endpoint := c.apiURL("/projects/" + url.PathEscape(project) + "/pipelines/" + strconv.FormatInt(pipelineID, 10) + "/jobs")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	c.addAuth(req)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("gitlab list pipeline jobs returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var jobs []Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (c Client) GetJobTrace(ctx context.Context, project string, jobID int64) (string, error) {
	endpoint := c.apiURL("/projects/" + url.PathEscape(project) + "/jobs/" + strconv.FormatInt(jobID, 10) + "/trace")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	c.addAuth(req)

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("gitlab job trace returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c Client) apiURL(path string) string {
	return strings.TrimRight(c.BaseURL, "/") + "/api/v4" + path
}

func (c Client) addAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("PRIVATE-TOKEN", c.Token)
	}
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}
