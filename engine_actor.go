package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/asynkron/protoactor-go/actor"
)

type RedditEngineActor struct {
	engine *RedditEngine
}

type GetCommentsMessage struct {
	PostID string
}

func NewRedditEngineActor() actor.Actor {
	return &RedditEngineActor{
		engine: NewRedditEngine(),
	}
}

// Helper function to count total comments including replies
func countComments(comments []*Comment) int {
	count := len(comments)
	for _, comment := range comments {
		count += countComments(comment.Children)
	}
	return count
}

func PrintCommentTree(comments []*Comment, indent int) {
	for _, comment := range comments {
		fmt.Printf("%s- %s (by %s)\n", strings.Repeat("  ", indent), comment.Content, comment.Author)
		PrintCommentTree(comment.Children, indent+1)
	}
}

func (state *RedditEngineActor) Receive(context actor.Context) {
	switch msg := context.Message().(type) {
	case *RegisterUserMessage:
		err := state.engine.RegisterUser(msg.Username, msg.Password)
		context.Respond(err)

	case *CreateSubredditMessage:
		err := state.engine.CreateSubreddit(msg.Name, msg.Description, msg.Creator)
		context.Respond(err)

	case *JoinSubredditMessage:
		err := state.engine.JoinSubreddit(msg.Username, msg.Subreddit)
		context.Respond(err)

	case *LeaveSubredditMessage:
		err := state.engine.LeaveSubreddit(msg.Username, msg.Subreddit)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("User %s successfully left subreddit %s\n", msg.Username, msg.Subreddit)
		}
		context.Respond(err)

	case *CreatePostMessage:
		fmt.Printf("Engine: Creating post by %s\n", msg.Author)
		post, err := state.engine.CreatePost(msg.Title, msg.Content, msg.Author, msg.Subreddit)
		fmt.Printf("Engine: Post creation result - Post: %v, Error: %v\n", post != nil, err)
		context.Respond(&struct {
			Post *Post
			Err  error
		}{post, err})

	case *AddCommentMessage:
		fmt.Printf("Engine: Adding comment by %s\n", msg.Author)
		comment, err := state.engine.AddComment(msg.Content, msg.Author, msg.PostID, msg.ParentCommentID)
		context.Respond(&struct {
			Comment *Comment
			Err     error
		}{comment, err})

	case *GetCommentsMessage:
		comments, err := state.engine.GetComments(msg.PostID)
		if err != nil {
			fmt.Printf("Error retrieving comments for post %s: %v\n", msg.PostID, err)
			context.Respond(err)
		} else {
			fmt.Printf("Comments for post %s:\n", msg.PostID)
			PrintCommentTree(comments, 0) // Helper function for formatting
			context.Respond(comments)
		}

	case *VotePostMessage:
		fmt.Printf("Engine: Processing vote for post %s\n", msg.PostID)
		err := state.engine.VotePost(msg.PostID, msg.Upvote)
		context.Respond(err)

	case *GetFeedMessage:
		feed, err := state.engine.GetUserFeed(msg.Username)
		context.Respond(&struct {
			Feed []*Post
			Err  error
		}{feed, err})

	case *SendDMMessage:
		dm, err := state.engine.SendDirectMessage(msg.From, msg.To, msg.Content)
		context.Respond(&struct {
			DM  *DirectMessage
			Err error
		}{dm, err})

	case *GetDMsMessage:
		dms, err := state.engine.GetDirectMessages(msg.Username)
		context.Respond(&struct {
			DMs []*DirectMessage
			Err error
		}{dms, err})

	case *ReplyToDMMessage:
		reply, err := state.engine.ReplyToDirectMessage(msg.OriginalMessageID, msg.From, msg.Content)
		context.Respond(&struct {
			Reply *DirectMessage
			Err   error
		}{reply, err})

	case *GetStatsMessage:
		totalComments := 0
		totalUpvotes := 0
		totalDownvotes := 0
		for _, post := range state.engine.posts {
			totalComments += countComments(post.Comments)
			if post.Votes > 0 {
				totalUpvotes += post.Votes
			} else {
				totalDownvotes += -post.Votes
			}
		}

		// Get top 10 users by karma
		topUsers := make([]UserKarma, 0)
		for username, user := range state.engine.users {
			topUsers = append(topUsers, UserKarma{
				Username: username,
				Karma:    user.Karma,
			})
		}
		// Sort users by karma
		sort.Slice(topUsers, func(i, j int) bool {
			return topUsers[i].Karma > topUsers[j].Karma
		})
		// Keep only top 10
		if len(topUsers) > 10 {
			topUsers = topUsers[:10]
		}

		totalDMs := 0
		for _, messages := range state.engine.directMessages {
			totalDMs += len(messages)
		}

		context.Respond(&StatsResponse{
			Users:          len(state.engine.users),
			Subreddits:     len(state.engine.subreddits),
			Posts:          len(state.engine.posts),
			Comments:       totalComments,
			DirectMessages: totalDMs,
			TotalUpvotes:   totalUpvotes,
			TotalDownvotes: totalDownvotes,
			TopUsers:       topUsers,
		})
	}
}
