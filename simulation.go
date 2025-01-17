package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type Simulator struct {
	baseURL string
	clients []*APIClient
	posts   []string // Store post IDs
}

func NewSimulator(baseURL string, numUsers int) *Simulator {
	return &Simulator{
		baseURL: baseURL,
		clients: make([]*APIClient, numUsers),
		posts:   make([]string, 0),
	}
}

func RunSimulation(baseURL string, numUsers int) {
	rand.Seed(time.Now().UnixNano())
	simulator := NewSimulator(baseURL, numUsers)
	simulator.Run()
}

func (s *Simulator) Run() {
	// 1. Register users
	log.Println("Registering users...")
	var wg sync.WaitGroup
	for i := range s.clients {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			client := NewAPIClient(s.baseURL, fmt.Sprintf("user%d", i+1))
			s.clients[i] = client

			if err := client.Register(fmt.Sprintf("user%d", i+1), "password"); err != nil {
				log.Printf("Failed to register user%d: %v", i+1, err)
				return
			}
			log.Printf("Registered user%d", i+1)
		}(i)
	}
	wg.Wait()

	// 2. Create subreddits
	log.Println("\nCreating subreddits...")
	subreddits := []string{"technology", "science", "gaming", "movies", "DOSP", "CISE", "UF", "USA", "books"}
	for i, subreddit := range subreddits {
		client := s.clients[i%len(s.clients)]
		err := client.CreateSubreddit(subreddit, fmt.Sprintf("A subreddit about %s", subreddit))
		if err != nil {
			log.Printf("Failed to create subreddit %s: %v", subreddit, err)
			continue
		}
		log.Printf("Created subreddit: %s", subreddit)
		time.Sleep(time.Millisecond * 100)
	}

	// 3. Users join random subreddits
	log.Println("\nUsers joining subreddits...")
	for _, client := range s.clients {
		numToJoin := 2 + rand.Intn(3) // Join 2-4 subreddits
		for i := 0; i < numToJoin; i++ {
			subreddit := subreddits[rand.Intn(len(subreddits))]
			if err := client.JoinSubreddit(subreddit); err != nil {
				log.Printf("Failed to join subreddit %s: %v", subreddit, err)
				continue
			}
			log.Printf("%s joined %s", client.username, subreddit)
			time.Sleep(time.Millisecond * 100)
		}
	}

	// 4. Create posts
	log.Println("\nCreating posts...")
	for _, client := range s.clients {
		numPosts := 2 + rand.Intn(5) // Create 2-7 posts
		for i := 0; i < numPosts; i++ {
			subreddit := subreddits[rand.Intn(len(subreddits))]
			title := fmt.Sprintf("Post %d by %s", i+1, client.username)
			content := fmt.Sprintf("This is test content for post %d", i+1)

			if err := client.CreatePost(title, content, subreddit); err != nil {
				log.Printf("Failed to create post: %v", err)
				continue
			}
			log.Printf("%s created post in %s", client.username, subreddit)
			time.Sleep(time.Millisecond * 100)
		}
	}

	// 5. Get all posts for voting and commenting
	log.Println("\nGathering posts for interaction...")
	for _, client := range s.clients {
		posts, err := client.GetPosts()
		if err != nil {
			log.Printf("Failed to get posts: %v", err)
			continue
		}
		for _, post := range posts {
			s.posts = append(s.posts, post.ID)
		}
	}

	// 6. Simulate voting
	log.Println("\nSimulating voting...")
	for _, client := range s.clients {
		numVotes := 5 + rand.Intn(5) // Vote on 5-10 posts
		for i := 0; i < numVotes && len(s.posts) > 0; i++ {
			postID := s.posts[rand.Intn(len(s.posts))]
			upvote := rand.Float32() > 0.3 // 70% chance of upvote
			if err := client.VotePost(postID, upvote); err != nil {
				log.Printf("Failed to vote: %v", err)
				continue
			}
			if upvote {
				log.Printf("%s upvoted post %s", client.username, postID)
			} else {
				log.Printf("%s downvoted post %s", client.username, postID)
			}
			time.Sleep(time.Millisecond * 100)
		}
	}

	// 6.5 Simulate commenting
	log.Println("\nSimulating comments on posts...")
	commentTemplates := []string{
		"Great post! Really enjoyed reading this.",
		"Interesting perspective on this topic.",
		"I disagree with some points here.",
		"Thanks for sharing this information!",
		"Could you elaborate more on this?",
		"This reminds me of something similar...",
		"Very well written and explained.",
		"Not sure I agree, but interesting viewpoint.",
		"This needs more discussion.",
		"Looking forward to more posts like this!",
	}

	for _, client := range s.clients {
		numComments := 3 + rand.Intn(5) // Each user makes 3-8 comments
		for i := 0; i < numComments && len(s.posts) > 0; i++ {

			postID := s.posts[rand.Intn(len(s.posts))]
			commentText := commentTemplates[rand.Intn(len(commentTemplates))]

			commentData := map[string]string{
				"content": fmt.Sprintf("%s", commentText),
			}

			jsonData, err := json.Marshal(commentData)
			if err != nil {
				log.Printf("Failed to marshal comment: %v", err)
				continue
			}

			req, err := http.NewRequest(
				"POST",
				fmt.Sprintf("%s/api/posts/%s/comments", s.baseURL, postID),
				bytes.NewBuffer(jsonData),
			)
			if err != nil {
				log.Printf("Failed to create request: %v", err)
				continue
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Username", client.username)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("Failed to add comment: %v", err)
				continue
			}
			resp.Body.Close()

			log.Printf("%s commented on post %s: %s", client.username, postID, commentText)
			time.Sleep(time.Millisecond * 100)

			if rand.Float32() < 0.25 {
				replyText := commentTemplates[rand.Intn(len(commentTemplates))]
				replyReq := CommentRequest{
					Content: fmt.Sprintf("Reply: %s [Reply by %s]", replyText, client.username),
				}

				resp, err := http.Post(
					fmt.Sprintf("%s/api/posts/%s/comments", s.baseURL, postID),
					"application/json",
					bytes.NewBuffer([]byte(fmt.Sprintf(`{"content":"%s"}`, replyReq.Content))),
				)

				if err != nil {
					log.Printf("Failed to add reply: %v", err)
					continue
				}
				resp.Body.Close()

				log.Printf("%s replied to a comment on post %s: %s", client.username, postID, replyText)
				time.Sleep(time.Millisecond * 100)
			}
		}
	}

	// 7. Some users leave subreddits
	log.Println("\nSimulating users leaving subreddits...")
	for _, client := range s.clients {
		if rand.Float32() > 0.7 { // 30% chance to leave a subreddit
			subreddit := subreddits[rand.Intn(len(subreddits))]
			if err := client.LeaveSubreddit(subreddit); err != nil {
				log.Printf("Failed to leave subreddit: %v", err)
				continue
			}
			log.Printf("%s left %s", client.username, subreddit)
			time.Sleep(time.Millisecond * 100)
		}
	}

	// Print final statistics
	log.Println("\nSimulation completed. Getting final user stats...")
	time.Sleep(time.Second) // Wait for any pending operations

	resp, err := http.Get(fmt.Sprintf("%s/api/users", s.baseURL))
	if err != nil {
		log.Printf("Failed to get final stats: %v", err)
		return
	}
	defer resp.Body.Close()

	var result SuccessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode stats: %v", err)
		return
	}

	log.Printf("\nFinal Statistics:")
	prettyPrint, _ := json.MarshalIndent(result.Data, "", "    ")
	// Add this to the final statistics section
	log.Println("\nGetting posts with comments...")
	for _, postID := range s.posts {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/posts/%s/comments", s.baseURL, postID), nil)
		if err != nil {
			log.Printf("Failed to create request: %v", err)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Failed to get comments for post %s: %v", postID, err)
			continue
		}

		var result struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Data    []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
				Author  string `json:"author"`
			} `json:"data"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if err != nil {
			log.Printf("Failed to decode comments for post %s: %v", postID, err)
			continue
		}

		if len(result.Data) > 0 {
			log.Printf("Post %s has %d comments:", postID, len(result.Data))
			for _, comment := range result.Data {
				log.Printf("  - %s: %s", comment.Author, comment.Content)
			}
		} else {
			log.Printf("Post %s has no comments", postID)
		}
	}
	log.Println("****** FINAL STATS OF ALL USERS ******")
	fmt.Println(string(prettyPrint))
}
