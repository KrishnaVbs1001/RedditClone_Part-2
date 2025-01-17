package main

type RegisterUserMessage struct {
	Username string
	Password string
}

type CreateSubredditMessage struct {
	Name        string
	Description string
	Creator     string
}

type JoinSubredditMessage struct {
	Username  string
	Subreddit string
}

type CreatePostMessage struct {
	Title     string
	Content   string
	Author    string
	Subreddit string
}

type AddCommentMessage struct {
	Content         string
	Author          string
	PostID          string
	ParentCommentID string
}

type VotePostMessage struct {
	PostID string
	Upvote bool
}

type GetFeedMessage struct {
	Username string
}

type SendDMMessage struct {
	From    string
	To      string
	Content string
}

type GetDMsMessage struct {
	Username string
}

type ReplyToDMMessage struct {
	OriginalMessageID string
	From              string
	Content           string
}

type GetStatsMessage struct{}

type StatsResponse struct {
	Users          int
	Subreddits     int
	Posts          int
	Comments       int
	DirectMessages int
	TotalUpvotes   int
	TotalDownvotes int
	TopUsers       []UserKarma // For displaying top users by karma
}

type UserKarma struct {
	Username string
	Karma    int
}

type LeaveSubredditMessage struct {
	Username  string
	Subreddit string
}
