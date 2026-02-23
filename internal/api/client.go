package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mauricejumelet/jira-cli/internal/config"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	authHeader string
}

func NewClient(cfg *config.Config) *Client {
	credentials := cfg.Email + ":" + cfg.APIToken
	encoded := base64.StdEncoding.EncodeToString([]byte(credentials))

	return &Client{
		httpClient: &http.Client{},
		baseURL:    cfg.BaseURL,
		authHeader: "Basic " + encoded,
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) doRequest(method, endpoint string, body io.Reader) ([]byte, error) {
	reqURL := c.baseURL + endpoint

	req, err := http.NewRequest(method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode == 204 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

func parseError(statusCode int, body []byte) error {
	var errResp struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		var parts []string
		parts = append(parts, errResp.ErrorMessages...)
		for field, msg := range errResp.Errors {
			parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
		}
		if len(parts) > 0 {
			return fmt.Errorf("API error (%d): %s", statusCode, strings.Join(parts, "; "))
		}
	}
	return fmt.Errorf("API error (%d): %s", statusCode, string(body))
}

// --- Types ---

type Issue struct {
	ID     string            `json:"id"`
	Key    string            `json:"key"`
	Self   string            `json:"self"`
	Fields IssueFields       `json:"fields"`
}

type IssueFields struct {
	Summary     string                 `json:"summary"`
	Description map[string]interface{} `json:"description,omitempty"`
	Status      *Status                `json:"status,omitempty"`
	Priority    *Priority              `json:"priority,omitempty"`
	IssueType   *IssueType             `json:"issuetype,omitempty"`
	Assignee    *User                  `json:"assignee,omitempty"`
	Reporter    *User                  `json:"reporter,omitempty"`
	Created     string                 `json:"created,omitempty"`
	Updated     string                 `json:"updated,omitempty"`
	Labels      []string               `json:"labels,omitempty"`
	Project     *Project               `json:"project,omitempty"`
	Comment     *CommentPage           `json:"comment,omitempty"`
}

type Status struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Priority struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IssueType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress,omitempty"`
	Active      bool   `json:"active,omitempty"`
}

type Project struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
	Self string `json:"self,omitempty"`
}

type Comment struct {
	ID      string                 `json:"id"`
	Author  *User                  `json:"author,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
	Created string                 `json:"created,omitempty"`
	Updated string                 `json:"updated,omitempty"`
}

type CommentPage struct {
	Comments   []Comment `json:"comments"`
	MaxResults int       `json:"maxResults"`
	Total      int       `json:"total"`
	StartAt    int       `json:"startAt"`
}

type Transition struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	To   *Status `json:"to,omitempty"`
}

type Attachment struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType,omitempty"`
	Created  string `json:"created,omitempty"`
	Author   *User  `json:"author,omitempty"`
	Content  string `json:"content,omitempty"`
}

type SearchResult struct {
	Issues        []Issue `json:"issues"`
	Total         int     `json:"total"`
	MaxResults    int     `json:"maxResults"`
	StartAt       int     `json:"startAt"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
}

// --- Issue Operations ---

func (c *Client) GetIssue(issueKey string, fields []string, expand []string) (*Issue, error) {
	params := url.Values{}
	if len(fields) > 0 {
		params.Set("fields", strings.Join(fields, ","))
	}
	if len(expand) > 0 {
		params.Set("expand", strings.Join(expand, ","))
	}

	endpoint := "/rest/api/3/issue/" + issueKey
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &issue, nil
}

func (c *Client) CreateIssue(projectKey, summary, issueType string, description map[string]interface{}, priority, assignee string, labels []string) (*Issue, error) {
	fields := map[string]interface{}{
		"project":   map[string]string{"key": projectKey},
		"summary":   summary,
		"issuetype": map[string]string{"name": issueType},
	}

	if description != nil {
		fields["description"] = description
	}
	if priority != "" {
		fields["priority"] = map[string]string{"name": priority}
	}
	if assignee != "" {
		fields["assignee"] = map[string]string{"accountId": assignee}
	}
	if len(labels) > 0 {
		fields["labels"] = labels
	}

	payload := map[string]interface{}{"fields": fields}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	body, err := c.doRequest("POST", "/rest/api/3/issue", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := json.Unmarshal(body, &issue); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &issue, nil
}

func (c *Client) UpdateIssue(issueKey string, fields map[string]interface{}) error {
	payload := map[string]interface{}{"fields": fields}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, err = c.doRequest("PUT", "/rest/api/3/issue/"+issueKey, bytes.NewReader(jsonBody))
	return err
}

func (c *Client) DeleteIssue(issueKey string, deleteSubtasks bool) error {
	endpoint := fmt.Sprintf("/rest/api/3/issue/%s?deleteSubtasks=%t", issueKey, deleteSubtasks)
	_, err := c.doRequest("DELETE", endpoint, nil)
	return err
}

func (c *Client) GetTransitions(issueKey string) ([]Transition, error) {
	body, err := c.doRequest("GET", fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Transitions []Transition `json:"transitions"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return resp.Transitions, nil
}

func (c *Client) TransitionIssue(issueKey, transitionID string) error {
	payload := map[string]interface{}{
		"transition": map[string]string{"id": transitionID},
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	_, err = c.doRequest("POST", fmt.Sprintf("/rest/api/3/issue/%s/transitions", issueKey), bytes.NewReader(jsonBody))
	return err
}

func (c *Client) SearchIssues(jql string, fields []string, maxResults int, nextPageToken string) (*SearchResult, error) {
	data := map[string]interface{}{
		"jql":        jql,
		"maxResults": maxResults,
	}

	if len(fields) > 0 {
		data["fields"] = fields
	} else {
		data["fields"] = []string{"summary", "status", "priority", "assignee", "created", "updated", "issuetype"}
	}

	if nextPageToken != "" {
		data["nextPageToken"] = nextPageToken
	}

	jsonBody, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	body, err := c.doRequest("POST", "/rest/api/3/search/jql", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &result, nil
}

// --- Comment Operations ---

func (c *Client) GetComments(issueKey string, maxResults int) (*CommentPage, error) {
	params := url.Values{}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	endpoint := fmt.Sprintf("/rest/api/3/issue/%s/comment?%s", issueKey, params.Encode())
	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var page CommentPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &page, nil
}

func (c *Client) AddComment(issueKey string, adfBody map[string]interface{}) (*Comment, error) {
	payload := map[string]interface{}{
		"body": adfBody,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	body, err := c.doRequest("POST", fmt.Sprintf("/rest/api/3/issue/%s/comment", issueKey), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &comment, nil
}

func (c *Client) UpdateComment(issueKey, commentID string, adfBody map[string]interface{}) (*Comment, error) {
	payload := map[string]interface{}{
		"body": adfBody,
	}
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	body, err := c.doRequest("PUT", fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := json.Unmarshal(body, &comment); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &comment, nil
}

func (c *Client) DeleteComment(issueKey, commentID string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/rest/api/3/issue/%s/comment/%s", issueKey, commentID), nil)
	return err
}

// --- Attachment Operations ---

func (c *Client) AddAttachment(issueKey, filePath string, filename string) ([]Attachment, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	displayName := filename
	if displayName == "" {
		displayName = filepath.Base(filePath)
	}

	part, err := writer.CreateFormFile("file", displayName)
	if err != nil {
		return nil, fmt.Errorf("creating form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copying file data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing multipart writer: %w", err)
	}

	reqURL := c.baseURL + fmt.Sprintf("/rest/api/3/issue/%s/attachments", issueKey)
	req, err := http.NewRequest("POST", reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Atlassian-Token", "no-check")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseError(resp.StatusCode, respBody)
	}

	var attachments []Attachment
	if err := json.Unmarshal(respBody, &attachments); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return attachments, nil
}

// --- Project Operations ---

func (c *Client) GetProjects(maxResults int) ([]Project, error) {
	params := url.Values{}
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	body, err := c.doRequest("GET", "/rest/api/3/project/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []Project `json:"values"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return resp.Values, nil
}

func (c *Client) GetProject(projectKey string) (map[string]interface{}, error) {
	body, err := c.doRequest("GET", "/rest/api/3/project/"+projectKey, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result, nil
}

func (c *Client) GetStatuses(projectKey string) ([]map[string]interface{}, error) {
	var endpoint string
	if projectKey != "" {
		endpoint = fmt.Sprintf("/rest/api/3/project/%s/statuses", projectKey)
	} else {
		endpoint = "/rest/api/3/status"
	}

	body, err := c.doRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result, nil
}

// --- User Operations ---

func (c *Client) GetMyself() (*User, error) {
	body, err := c.doRequest("GET", "/rest/api/3/myself", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &user, nil
}

func (c *Client) SearchUsers(query string, maxResults int) ([]User, error) {
	params := url.Values{}
	params.Set("query", query)
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	body, err := c.doRequest("GET", "/rest/api/3/user/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return users, nil
}

func (c *Client) GetAssignableUsers(projectKey string, maxResults int) ([]User, error) {
	params := url.Values{}
	params.Set("project", projectKey)
	if maxResults > 0 {
		params.Set("maxResults", fmt.Sprintf("%d", maxResults))
	}

	body, err := c.doRequest("GET", "/rest/api/3/user/assignable/search?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return users, nil
}

// --- Metadata Operations ---

func (c *Client) GetIssueTypes() ([]IssueType, error) {
	body, err := c.doRequest("GET", "/rest/api/3/issuetype", nil)
	if err != nil {
		return nil, err
	}

	var types []IssueType
	if err := json.Unmarshal(body, &types); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return types, nil
}

func (c *Client) GetPriorities() ([]Priority, error) {
	body, err := c.doRequest("GET", "/rest/api/3/priority", nil)
	if err != nil {
		return nil, err
	}

	var priorities []Priority
	if err := json.Unmarshal(body, &priorities); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return priorities, nil
}
