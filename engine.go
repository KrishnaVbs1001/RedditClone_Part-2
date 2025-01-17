package main

import (
	"fmt"
	"sync"
	"time"
)

// Data Models
type User struct {
	Username   string
	Password   string
	Karma      int
	CreatedAt  time.Time
	Subreddits map[string]bool
	mu         sync.RWMutex
}

type Comment struct {
	ID        string
	Content   string
	Author    string
	ParentID  string
	CreatedAt time.Time
	Votes     int
	Children  []*Comment
	mu        sync.RWMutex
}

type Post struct {
	ID        string
	Title     string
	Content   string
	Author    string
	Subreddit string
	CreatedAt time.Time
	Votes     int
	Comments  []*Comment
	mu        sync.RWMutex
}

type Subreddit struct {
	Name        string
	Description string
	Creator     string
	CreatedAt   time.Time
	Posts       []*Post
	Members     map[string]bool
	mu          sync.RWMutex
}

type DirectMessage struct {
	ID        string
	From      string
	To        string
	Content   string
	CreatedAt time.Time
	Replies   []*DirectMessage
	mu        sync.RWMutex
}

// RedditEngine represents the main engine
type RedditEngine struct {
	users          map[string]*User
	subreddits     map[string]*Subreddit
	posts          map[string]*Post
	directMessages map[string][]*DirectMessage
	mu             sync.RWMutex
}

// NewRedditEngine creates a new Reddit engine instance
func NewRedditEngine() *RedditEngine {
	return &RedditEngine{
		users:          make(map[string]*User),
		subreddits:     make(map[string]*Subreddit),
		posts:          make(map[string]*Post),
		directMessages: make(map[string][]*DirectMessage),
	}
}

// User Management Methods
func (e *RedditEngine) RegisterUser(username, password string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.users[username]; exists {
		return fmt.Errorf("user already exists")
	}

	e.users[username] = &User{
		Username:   username,
		Password:   password,
		CreatedAt:  time.Now(),
		Subreddits: make(map[string]bool),
	}
	return nil
}

// Subreddit Management Methods
func (e *RedditEngine) CreateSubreddit(name, description, creator string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.subreddits[name]; exists {
		return fmt.Errorf("subreddit already exists")
	}

	e.subreddits[name] = &Subreddit{
		Name:        name,
		Description: description,
		Creator:     creator,
		CreatedAt:   time.Now(),
		Posts:       make([]*Post, 0),
		Members:     make(map[string]bool),
	}
	return nil
}

func (e *RedditEngine) JoinSubreddit(username, subredditName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	user, ok := e.users[username]
	if !ok {
		return fmt.Errorf("user not found")
	}

	subreddit, ok := e.subreddits[subredditName]
	if !ok {
		return fmt.Errorf("subreddit not found")
	}

	user.mu.Lock()
	user.Subreddits[subredditName] = true
	user.mu.Unlock()

	subreddit.mu.Lock()
	subreddit.Members[username] = true
	subreddit.mu.Unlock()

	return nil
}

func (e *RedditEngine) LeaveSubreddit(username, subredditName string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	fmt.Printf("LeaveSubreddit called for user: %s, subreddit: %s\n", username, subredditName)

	// Check if user exists
	user, ok := e.users[username]
	if !ok {
		fmt.Printf("User %s not found\n", username)
		return fmt.Errorf("user not found")
	}

	// Check if subreddit exists
	subreddit, ok := e.subreddits[subredditName]
	if !ok {
		fmt.Printf("Subreddit %s not found\n", subredditName)
		return fmt.Errorf("subreddit not found")
	}

	// Remove user from subreddit's members
	user.mu.Lock()
	delete(user.Subreddits, subredditName)
	user.mu.Unlock()
	//fmt.Printf("User %s removed from subreddit %s in user's subreddits\n", username, subredditName)

	subreddit.mu.Lock()
	delete(subreddit.Members, username)
	subreddit.mu.Unlock()
	//fmt.Printf("User %s removed from subreddit %s members\n", username, subredditName)

	return nil
}

// NEW
func (e *RedditEngine) GetComments(postID string) ([]*Comment, error) {
	e.mu.RLock()
	post, exists := e.posts[postID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("post not found")
	}

	// Return the root comments
	return post.Comments, nil
}

// Post Management Methods
func (e *RedditEngine) CreatePost(title, content, author, subredditName string) (*Post, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	subreddit, ok := e.subreddits[subredditName]
	if !ok {
		return nil, fmt.Errorf("subreddit not found")
	}

	post := &Post{
		ID:        fmt.Sprintf("post_%d", time.Now().UnixNano()),
		Title:     title,
		Content:   content,
		Author:    author,
		Subreddit: subredditName,
		CreatedAt: time.Now(),
		Comments:  make([]*Comment, 0),
	}

	e.posts[post.ID] = post

	subreddit.mu.Lock()
	subreddit.Posts = append(subreddit.Posts, post)
	subreddit.mu.Unlock()

	return post, nil
}

// Comment Methods
func (e *RedditEngine) AddComment(content, author, postID, parentCommentID string) (*Comment, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	post, ok := e.posts[postID]
	if !ok {
		return nil, fmt.Errorf("post not found")
	}

	comment := &Comment{
		ID:        fmt.Sprintf("comment_%d", time.Now().UnixNano()),
		Content:   content,
		Author:    author,
		ParentID:  parentCommentID,
		CreatedAt: time.Now(),
		Children:  make([]*Comment, 0),
	}

	post.mu.Lock()
	defer post.mu.Unlock()

	if parentCommentID == "" {
		post.Comments = append(post.Comments, comment)
	} else {
		parent := findComment(post.Comments, parentCommentID)
		if parent == nil {
			return nil, fmt.Errorf("parent comment not found")
		}
		parent.mu.Lock()
		parent.Children = append(parent.Children, comment)
		parent.mu.Unlock()
	}

	return comment, nil
}

func findComment(comments []*Comment, commentID string) *Comment {
	for _, comment := range comments {
		if comment.ID == commentID {
			return comment
		}
		if found := findComment(comment.Children, commentID); found != nil {
			return found
		}
	}
	return nil
}

// Voting and Karma Methods
func (e *RedditEngine) VotePost(postID string, upvote bool) error {
	e.mu.RLock()
	post, ok := e.posts[postID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("post not found")
	}

	post.mu.Lock()
	if upvote {
		post.Votes += 2
		e.updateKarma(post.Author, 1)
	} else {
		post.Votes--
		e.updateKarma(post.Author, -1)
	}
	post.mu.Unlock()

	return nil
}

func (e *RedditEngine) updateKarma(username string, value int) {
	e.mu.RLock()
	user, ok := e.users[username]
	e.mu.RUnlock()

	if ok {
		user.mu.Lock()
		user.Karma += value
		user.mu.Unlock()
	}
}

// Feed Generation
func (e *RedditEngine) GetUserFeed(username string) ([]*Post, error) {
	e.mu.RLock()
	user, ok := e.users[username]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("user not found")
	}

	var feed []*Post
	user.mu.RLock()
	defer user.mu.RUnlock()

	for subredditName := range user.Subreddits {
		e.mu.RLock()
		subreddit := e.subreddits[subredditName]
		e.mu.RUnlock()

		subreddit.mu.RLock()
		feed = append(feed, subreddit.Posts...)
		subreddit.mu.RUnlock()
	}

	sortPosts(feed)
	return feed, nil
}

// Direct Message Methods
func (e *RedditEngine) SendDirectMessage(from, to, content string) (*DirectMessage, error) {
	e.mu.RLock() // Use RLock instead of Lock for checking users
	_, fromExists := e.users[from]
	_, toExists := e.users[to]
	e.mu.RUnlock()

	if !fromExists {
		return nil, fmt.Errorf("sender not found")
	}
	if !toExists {
		return nil, fmt.Errorf("recipient not found")
	}

	dm := &DirectMessage{
		ID:        fmt.Sprintf("dm_%d", time.Now().UnixNano()),
		From:      from,
		To:        to,
		Content:   content,
		CreatedAt: time.Now(),
		Replies:   make([]*DirectMessage, 0),
	}

	e.mu.Lock()
	if e.directMessages[to] == nil {
		e.directMessages[to] = make([]*DirectMessage, 0)
	}
	e.directMessages[to] = append(e.directMessages[to], dm)
	e.mu.Unlock()

	return dm, nil
}

func (e *RedditEngine) ReplyToDirectMessage(originalMsgID, from, content string) (*DirectMessage, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	var originalDM *DirectMessage
	for _, messages := range e.directMessages {
		for _, dm := range messages {
			if dm.ID == originalMsgID {
				originalDM = dm
				break
			}
		}
	}

	if originalDM == nil {
		return nil, fmt.Errorf("original message not found")
	}

	reply := &DirectMessage{
		ID:        fmt.Sprintf("dm_%d", time.Now().UnixNano()),
		From:      from,
		To:        originalDM.From,
		Content:   content,
		CreatedAt: time.Now(),
	}

	originalDM.mu.Lock()
	originalDM.Replies = append(originalDM.Replies, reply)
	originalDM.mu.Unlock()

	return reply, nil
}

func (e *RedditEngine) GetDirectMessages(username string) ([]*DirectMessage, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if _, ok := e.users[username]; !ok {
		return nil, fmt.Errorf("user not found")
	}

	return e.directMessages[username], nil
}

// Helper Functions
func sortPosts(posts []*Post) {
	n := len(posts)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if posts[j].CreatedAt.Before(posts[j+1].CreatedAt) {
				posts[j], posts[j+1] = posts[j+1], posts[j]
			}
		}
	}
}
