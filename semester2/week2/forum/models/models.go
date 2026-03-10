package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Comment 评论实体
type Comment struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Author    string              `bson:"author" json:"author"`
	Content   string              `bson:"content" json:"content"`
	ReplyToID *primitive.ObjectID `bson:"reply_to_id,omitempty" json:"replyToId,omitempty"`
	CreatedAt time.Time           `bson:"created_at" json:"createdAt"`
	Custom    map[string]any      `bson:"custom,omitempty" json:"custom,omitempty"`

	UpVotes      int      `bson:"up_votes" json:"upVotes"`
	DownVotes    int      `bson:"down_votes" json:"downVotes"`
	UpVoterIDs   []string `bson:"up_voter_ids" json:"-"`
	DownVoterIDs []string `bson:"down_voter_ids" json:"-"`
}

// Post 主帖实体
type Post struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title     string             `bson:"title" json:"title"`
	Content   string             `bson:"content" json:"content"`
	Author    string             `bson:"author" json:"author"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updatedAt"`
	Custom    map[string]any     `bson:"custom,omitempty" json:"custom,omitempty"`
	Comments  []Comment          `bson:"comments" json:"comments"`

	UpVotes      int      `bson:"up_votes" json:"upVotes"`
	DownVotes    int      `bson:"down_votes" json:"downVotes"`
	UpVoterIDs   []string `bson:"up_voter_ids" json:"-"`
	DownVoterIDs []string `bson:"down_voter_ids" json:"-"`
}
