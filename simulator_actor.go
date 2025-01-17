package main

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
)

type SimulatorActor struct {
	enginePID *actor.PID
	userCount int
	posts     map[string]string
	wg        sync.WaitGroup
	startTime time.Time
}

func NewSimulatorActor(enginePID *actor.PID, userCount int) actor.Actor {
	return &SimulatorActor{
		enginePID: enginePID,
		userCount: userCount,
		posts:     make(map[string]string),
	}
}

func (state *SimulatorActor) GenerateZipfDistribution(alpha float64) []int {
	distribution := make([]int, state.userCount)
	for i := 0; i < state.userCount; i++ {
		rank := float64(i + 1)
		value := int(math.Ceil(float64(state.userCount) / math.Pow(rank, alpha)))
		distribution[i] = value
	}
	return distribution
}

func generateRandomName(prefix string, length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return prefix + string(result)
}

func (state *SimulatorActor) simulateUserActivity(context actor.Context, username string, activity int) {
	defer state.wg.Done()

	// Register user
	context.Request(state.enginePID, &RegisterUserMessage{
		Username: username,
		Password: "password",
	})
	fmt.Printf("User registered: %s\n", username)

	// Join subreddits
	numSubreddits := 2 + rand.Intn(3)
	for i := 0; i < numSubreddits; i++ {
		subredditName := fmt.Sprintf("r_%d", rand.Intn(20))
		context.Request(state.enginePID, &JoinSubredditMessage{
			Username:  username,
			Subreddit: subredditName,
		})
		fmt.Printf("User %s joined subreddit %s\n", username, subredditName)
		time.Sleep(time.Millisecond * 50)
	}

	// Simulate leaving sub-reddits
	numToLeave := rand.Intn(3) // Randomly leave up to 2 sub-reddits
	for i := 0; i < numToLeave; i++ {
		subredditName := fmt.Sprintf("r_%d", rand.Intn(20)) // Example subreddit naming convention
		context.Request(state.enginePID, &LeaveSubredditMessage{
			Username:  username,
			Subreddit: subredditName,
		})
		fmt.Printf("User %s requested to leave subreddit %s\n", username, subredditName)
		time.Sleep(time.Millisecond * 50) // Avoid flooding the engine
	}

	// Create posts
	postCount := 1 + int(float64(activity)/float64(state.userCount)*5)
	fmt.Printf("User %s will create %d posts\n", username, postCount)

	for i := 0; i < postCount; i++ {
		subredditName := fmt.Sprintf("r_%d", rand.Intn(20))

		// Create post
		future := context.RequestFuture(state.enginePID, &CreatePostMessage{
			Title:     fmt.Sprintf("Post %d by %s", i, username),
			Content:   fmt.Sprintf("Content for post %d", i),
			Author:    username,
			Subreddit: subredditName,
		}, 5*time.Second)

		result, err := future.Result()
		if err != nil {
			fmt.Printf("Error creating post: %v\n", err)
			continue
		}

		postResponse, ok := result.(*struct {
			Post *Post
			Err  error
		})
		if !ok {
			fmt.Printf("Invalid response type for post creation\n")
			continue
		}

		if postResponse.Err != nil {
			fmt.Printf("Error in post creation response: %v\n", postResponse.Err)
			continue
		}

		postID := postResponse.Post.ID
		fmt.Printf("Created post %s in subreddit %s\n", postID, subredditName)

		// Add votes
		numVotes := 5 + rand.Intn(10)
		for v := 0; v < numVotes; v++ {
			isUpvote := rand.Float64() < 0.7 // 70% chance of upvote
			voteMsg := &VotePostMessage{
				PostID: postID,
				Upvote: isUpvote,
			}
			future := context.RequestFuture(state.enginePID, voteMsg, time.Second)
			if _, err := future.Result(); err != nil {
				fmt.Printf("Error voting: %v\n", err)
			} else {
				if isUpvote {
					fmt.Printf("Upvoted post %s\n", postID)
				} else {
					fmt.Printf("Downvoted post %s\n", postID)
				}
			}
			time.Sleep(time.Millisecond * 10)
		}

		// Add comments
		for c := 0; c < 2+rand.Intn(5); c++ {
			commentMsg := &AddCommentMessage{
				Content:         fmt.Sprintf("Comment %d on post %s", c, postID),
				Author:          fmt.Sprintf("user_%d", rand.Intn(state.userCount)),
				PostID:          postID,
				ParentCommentID: "",
			}
			future := context.RequestFuture(state.enginePID, commentMsg, time.Second)
			if _, err := future.Result(); err != nil {
				fmt.Printf("Error commenting: %v\n", err)
			}
		}

		time.Sleep(time.Millisecond * 100)
	}

	// Send DMs
	dmCount := rand.Intn(3)
	for i := 0; i < dmCount; i++ {
		recipient := fmt.Sprintf("user_%d", rand.Intn(state.userCount))
		dmMsg := &SendDMMessage{
			From:    username,
			To:      recipient,
			Content: fmt.Sprintf("Message %d from %s to %s", i, username, recipient),
		}
		future := context.RequestFuture(state.enginePID, dmMsg, time.Second)
		if _, err := future.Result(); err != nil {
			fmt.Printf("Error sending DM: %v\n", err)
		}
		time.Sleep(time.Millisecond * 50)
	}
}

func (state *SimulatorActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *actor.Started:
		fmt.Println("Starting simulation...")
		state.startTime = time.Now()
		distribution := state.GenerateZipfDistribution(1.3)

		// Create subreddits first and wait for them to be created
		subreddits := make([]string, 20)
		for i := 0; i < 20; i++ {
			subredditName := fmt.Sprintf("r_%d", i) // Use consistent naming
			subreddits[i] = subredditName
			future := context.RequestFuture(state.enginePID, &CreateSubredditMessage{
				Name:        subredditName,
				Description: fmt.Sprintf("A community for %s", subredditName),
				Creator:     "admin",
			}, 5*time.Second)

			_, err := future.Result()
			if err != nil {
				fmt.Printf("Error creating subreddit %s: %v\n", subredditName, err)
			} else {
				fmt.Printf("Created subreddit: %s\n", subredditName)
			}
			time.Sleep(time.Millisecond * 10)
		}

		// Start user simulations after subreddits are created
		state.wg.Add(state.userCount)
		for i := 0; i < state.userCount; i++ {
			username := fmt.Sprintf("user_%d", i)
			activity := distribution[i]
			go state.simulateUserActivity(context, username, activity)
			time.Sleep(time.Millisecond * 5)
		}

		fmt.Println("Simulation started successfully")

		// Wait for completion in a goroutine
		go func() {
			state.wg.Wait()
			duration := time.Since(state.startTime)
			fmt.Printf("\nSimulation completed in %v\n", duration)
			context.Request(state.enginePID, &GetStatsMessage{})
		}()

	case *StatsResponse:
		fmt.Printf("\nFinal Statistics:\n")
		fmt.Printf("Total Users: %d\n", msg.Users)
		fmt.Printf("Total Subreddits: %d\n", msg.Subreddits)
		fmt.Printf("Total Posts: %d\n", msg.Posts)
		fmt.Printf("Total Comments: %d\n", msg.Comments)
		fmt.Printf("Total Direct Messages: %d\n", msg.DirectMessages)
		fmt.Printf("Total Upvotes: %d\n", msg.TotalUpvotes)
		fmt.Printf("Total Downvotes: %d\n", msg.TotalDownvotes)
		fmt.Printf("\nTop 10 Users by Karma:\n")
		for i, user := range msg.TopUsers {
			fmt.Printf("%d. %s: %d karma\n", i+1, user.Username, user.Karma)
		}
		context.Stop(context.Self())

	case *actor.Stopped:
		fmt.Println("Simulator stopped cleanly")
	}
}
