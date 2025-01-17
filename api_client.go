package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type APIClient struct {
	baseURL  string
	username string
	client   *http.Client
}

type RegisterRequest struct {
	Username *string `json:"username"` // Pointer to make it optional
	Password string  `json:"password"`
}

type CreateSubredditRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreatePostRequest struct {
	Title     string `json:"title"`
	Content   string `json:"content"`
	Subreddit string `json:"subreddit"`
}

type CommentRequest struct {
	Content string `json:"content"`
}

type VoteRequest struct {
	Upvote bool `json:"upvote"`
}

type MessageRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

func NewAPIClient(baseURL, username string) *APIClient {
	return &APIClient{
		baseURL:  baseURL,
		username: username,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *APIClient) Register(username, password string) error {
	data := RegisterRequest{
		Password: password,
	}
	return c.post("/api/register", data, nil)
}

func (c *APIClient) CreateSubreddit(name, description string) error {
	data := CreateSubredditRequest{
		Name:        name,
		Description: description,
	}
	return c.post("/api/subreddits", data, nil)
}

func (c *APIClient) JoinSubreddit(name string) error {
	return c.post(fmt.Sprintf("/api/subreddits/%s/join", name), nil, nil)
}

func (c *APIClient) LeaveSubreddit(name string) error {
	return c.post(fmt.Sprintf("/api/subreddits/%s/leave", name), nil, nil)
}

func (c *APIClient) CreatePost(title, content, subreddit string) error {
	data := CreatePostRequest{
		Title:     title,
		Content:   content,
		Subreddit: subreddit,
	}
	return c.post("/api/posts", data, nil)
}

func (c *APIClient) GetPosts() ([]*Post, error) {
	var response SuccessResponse
	err := c.get("/api/posts", &response)
	if err != nil {
		return nil, err
	}

	// Type assertion for the response data
	postsData, ok := response.Data.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Convert the data to []*Post
	posts := make([]*Post, 0, len(postsData))
	for _, postData := range postsData {
		if postMap, ok := postData.(map[string]interface{}); ok {
			post := &Post{
				ID:        postMap["id"].(string),
				Title:     postMap["title"].(string),
				Content:   postMap["content"].(string),
				Author:    postMap["author"].(string),
				Subreddit: postMap["subreddit"].(string),
				Votes:     int(postMap["votes"].(float64)),
			}
			posts = append(posts, post)
		}
	}

	return posts, nil
}

func (c *APIClient) VotePost(postID string, upvote bool) error {
	data := VoteRequest{Upvote: upvote}
	return c.post(fmt.Sprintf("/api/posts/%s/vote", postID), data, nil)
}

// Helper methods for HTTP requests
func (c *APIClient) post(endpoint string, data interface{}, response interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Username", c.username)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf(errResp.Message)
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}
	return nil
}

func (c *APIClient) get(endpoint string, response interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+endpoint, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Username", c.username)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf(errResp.Message)
	}

	return json.NewDecoder(resp.Body).Decode(response)
}
