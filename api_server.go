package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
)

type APIServer struct {
	engine      *RedditEngine
	router      *mux.Router
	userCounter uint64
}

// Response structures
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewAPIServer(engine *RedditEngine) *APIServer {
	server := &APIServer{
		engine: engine,
		router: mux.NewRouter(),
	}
	server.setupRoutes()
	return server
}

func (s *APIServer) setupRoutes() {
	// Auth routes
	s.router.HandleFunc("/api/register", s.handleRegister).Methods("POST")

	// Subreddit routes
	s.router.HandleFunc("/api/subreddits", s.handleCreateSubreddit).Methods("POST")
	s.router.HandleFunc("/api/subreddits/{name}/join", s.handleJoinSubreddit).Methods("POST")
	s.router.HandleFunc("/api/subreddits/{name}/leave", s.handleLeaveSubreddit).Methods("POST")

	// Post routes
	s.router.HandleFunc("/api/posts", s.handleCreatePost).Methods("POST")
	s.router.HandleFunc("/api/posts", s.handleGetPosts).Methods("GET")
	s.router.HandleFunc("/api/posts/{id}/vote", s.handleVotePost).Methods("POST")
	s.router.HandleFunc("/api/posts/{id}/comments", s.handleAddComment).Methods("POST")

	// Message routes
	s.router.HandleFunc("/api/messages", s.handleSendMessage).Methods("POST")
	s.router.HandleFunc("/api/messages", s.handleGetMessages).Methods("GET")
	s.router.HandleFunc("/api/users", s.handleGetUsers).Methods("GET")

	s.router.HandleFunc("/api/posts/{id}/comments", s.handleGetComments).Methods("GET")
	s.router.HandleFunc("/api/stats", s.handleGetStats).Methods("GET")

}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *APIServer) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	var username string
	if req.Username == nil {
		// Auto-generate username
		userNum := atomic.AddUint64(&s.userCounter, 1)
		username = fmt.Sprintf("user%d", userNum)
	} else {
		// Use provided username
		username = *req.Username
		// Check if username is empty
		if strings.TrimSpace(username) == "" {
			writeJSON(w, ErrorResponse{
				Status:  "error",
				Message: "Username cannot be empty",
			})
			return
		}
	}

	err := s.engine.RegisterUser(username, req.Password)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to register user: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("%s registered successfully", username),
		Data: map[string]string{
			"username": username,
		},
	})
}

func (s *APIServer) handleCreateSubreddit(w http.ResponseWriter, r *http.Request) {
	var req CreateSubredditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	username := r.Header.Get("Username")
	err := s.engine.CreateSubreddit(req.Name, req.Description, username)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to create subreddit '%s': %v", req.Name, err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("Subreddit '%s' created by %s", req.Name, username),
		Data: map[string]string{
			"name":        req.Name,
			"creator":     username,
			"description": req.Description,
		},
	})
}

func (s *APIServer) handleJoinSubreddit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subredditName := vars["name"]
	username := r.Header.Get("Username")

	err := s.engine.JoinSubreddit(username, subredditName)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to join subreddit: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("%s joined subreddit '%s'", username, subredditName),
	})
}

func (s *APIServer) handleLeaveSubreddit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subredditName := vars["name"]
	username := r.Header.Get("Username")

	err := s.engine.LeaveSubreddit(username, subredditName)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to leave subreddit: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("%s left subreddit '%s'", username, subredditName),
	})
}

func (s *APIServer) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	var req CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	username := r.Header.Get("Username")
	post, err := s.engine.CreatePost(req.Title, req.Content, username, req.Subreddit)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to create post: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("Post created by %s in %s", username, req.Subreddit),
		Data:    post,
	})
}

func (s *APIServer) handleGetPosts(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("Username")
	posts, err := s.engine.GetUserFeed(username)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to get posts: %v", err),
		})
		return
	}

	type PostResponse struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Author    string    `json:"author"`
		Content   string    `json:"content"`
		Subreddit string    `json:"subreddit"`
		Votes     int       `json:"votes"`
		CreatedAt time.Time `json:"created_at"`
	}

	prettifiedPosts := make([]PostResponse, 0)
	for _, post := range posts {
		prettifiedPosts = append(prettifiedPosts, PostResponse{
			ID:        post.ID,
			Title:     post.Title,
			Author:    post.Author,
			Content:   post.Content,
			Subreddit: post.Subreddit,
			Votes:     post.Votes,
			CreatedAt: post.CreatedAt,
		})
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("Retrieved %d posts for %s", len(posts), username),
		Data:    prettifiedPosts,
	})
}

func (s *APIServer) handleVotePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]
	username := r.Header.Get("Username")

	var req VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	err := s.engine.VotePost(postID, req.Upvote)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to vote on post: %v", err),
		})
		return
	}

	voteType := "upvoted"
	if !req.Upvote {
		voteType = "downvoted"
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("%s %s post %s", username, voteType, postID),
	})
}

func (s *APIServer) handleAddComment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]
	username := r.Header.Get("Username")

	var req CommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	comment, err := s.engine.AddComment(req.Content, username, postID, "")
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to add comment: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("%s commented on post %s", username, postID),
		Data:    comment,
	})
}

func (s *APIServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	var req MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Invalid request format",
		})
		return
	}

	username := r.Header.Get("Username")
	msg, err := s.engine.SendDirectMessage(username, req.To, req.Content)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to send message: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("%s sent a message to %s", username, req.To),
		Data:    msg,
	})
}

func (s *APIServer) handleGetMessages(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("Username")
	messages, err := s.engine.GetDirectMessages(username)
	if err != nil {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: fmt.Sprintf("Failed to get messages: %v", err),
		})
		return
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("Retrieved %d messages for %s", len(messages), username),
		Data:    messages,
	})
}

func (s *APIServer) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *APIServer) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	type UserInfo struct {
		Username   string    `json:"username"`
		CreatedAt  time.Time `json:"created_at"`
		Karma      int       `json:"karma"`
		Subreddits int       `json:"subreddits"`
	}

	userList := make([]UserInfo, 0)
	for username, user := range s.engine.users {
		user.mu.RLock()
		userInfo := UserInfo{
			Username:   username,
			CreatedAt:  user.CreatedAt,
			Karma:      user.Karma,
			Subreddits: len(user.Subreddits),
		}
		user.mu.RUnlock()
		userList = append(userList, userInfo)
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("Retrieved %d users", len(userList)),
		Data:    userList,
	})
}

func (s *APIServer) handleGetComments(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID := vars["id"]

	post, exists := s.engine.posts[postID]
	if !exists {
		writeJSON(w, ErrorResponse{
			Status:  "error",
			Message: "Post not found",
		})
		return
	}

	type CommentResponse struct {
		ID        string            `json:"id"`
		Content   string            `json:"content"`
		Author    string            `json:"author"`
		CreatedAt time.Time         `json:"created_at"`
		Votes     int               `json:"votes"`
		Children  []CommentResponse `json:"children"`
	}

	// Convert comments to response format
	var convertComment func(*Comment) CommentResponse
	convertComment = func(c *Comment) CommentResponse {
		children := make([]CommentResponse, 0)
		for _, child := range c.Children {
			children = append(children, convertComment(child))
		}

		return CommentResponse{
			ID:        c.ID,
			Content:   c.Content,
			Author:    c.Author,
			CreatedAt: c.CreatedAt,
			Votes:     c.Votes,
			Children:  children,
		}
	}

	comments := make([]CommentResponse, 0)
	for _, comment := range post.Comments {
		comments = append(comments, convertComment(comment))
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: fmt.Sprintf("Retrieved %d comments for post %s", len(comments), postID),
		Data:    comments,
	})
}

func (s *APIServer) handleGetStats(w http.ResponseWriter, r *http.Request) {
	type UserKarma struct {
		Username string `json:"username"`
		Karma    int    `json:"karma"`
	}
	log.Println("------Top 5 users based on karma------")

	type StatsResponse struct {
		TotalUsers      int         `json:"total_users"`
		TotalSubreddits int         `json:"total_subreddits"`
		TotalPosts      int         `json:"total_posts"`
		TotalComments   int         `json:"total_comments"`
		DirectMessages  int         `json:"total_direct_messages"`
		TotalUpvotes    int         `json:"total_upvotes"`
		TotalDownvotes  int         `json:"total_downvotes"`
		TopUsers        []UserKarma `json:"top_users"`
	}

	// Calculate total comments
	totalComments := 0
	totalUpvotes := 0
	totalDownvotes := 0
	for _, post := range s.engine.posts {
		post.mu.RLock()
		if post.Votes > 0 {
			totalUpvotes += post.Votes
		} else {
			totalDownvotes += -post.Votes
		}
		totalComments += countComments(post.Comments)
		post.mu.RUnlock()
	}

	// Get top users by karma
	type userKarmaPair struct {
		username string
		karma    int
	}
	userKarmas := make([]userKarmaPair, 0)

	for username, user := range s.engine.users {
		user.mu.RLock()
		karma := user.Karma
		user.mu.RUnlock()
		userKarmas = append(userKarmas, userKarmaPair{username, karma})
	}

	// Sort users by karma
	sort.Slice(userKarmas, func(i, j int) bool {
		return userKarmas[i].karma > userKarmas[j].karma
	})

	// Get top 5 users
	topUsers := make([]UserKarma, 0)
	for i := 0; i < len(userKarmas) && i < 5; i++ {
		topUsers = append(topUsers, UserKarma{
			Username: userKarmas[i].username,
			Karma:    userKarmas[i].karma,
		})
	}

	// Count total direct messages
	totalDMs := 0
	for _, messages := range s.engine.directMessages {
		totalDMs += len(messages)
	}

	stats := StatsResponse{
		TotalUsers:      len(s.engine.users),
		TotalSubreddits: len(s.engine.subreddits),
		TotalPosts:      len(s.engine.posts),
		TotalComments:   totalComments,
		DirectMessages:  totalDMs,
		TotalUpvotes:    totalUpvotes,
		TotalDownvotes:  totalDownvotes,
		TopUsers:        topUsers,
	}

	writeJSON(w, SuccessResponse{
		Status:  "success",
		Message: "Statistics retrieved successfully",
		Data:    stats,
	})
}
