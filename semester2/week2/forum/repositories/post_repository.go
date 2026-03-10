package repositories

import (
	"context"
	"errors"
	"time"

	"forum/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type postRepository struct {
	collection *mongo.Collection
}

func NewPostRepository(db *mongo.Database) PostRepository {
	return &postRepository{
		collection: db.Collection("posts"),
	}
}

func (r *postRepository) Create(ctx context.Context, post *models.Post) error {
	post.ID = primitive.NewObjectID()
	post.CreatedAt = time.Now()
	post.UpdatedAt = time.Now()
	post.Comments = []models.Comment{}
	post.UpVotes = 0
	post.DownVotes = 0
	post.UpVoterIDs = []string{}
	post.DownVoterIDs = []string{}

	_, err := r.collection.InsertOne(ctx, post)
	return err
}

func (r *postRepository) FindByID(ctx context.Context, id string) (*models.Post, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var post models.Post
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&post)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("post not found")
		}
		return nil, err
	}

	return &post, nil
}

func (r *postRepository) FindAll(ctx context.Context, page, limit int) ([]models.Post, error) {
	skip := (page - 1) * limit

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var posts []models.Post
	if err = cursor.All(ctx, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *postRepository) Update(ctx context.Context, id string, update map[string]interface{}) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update["updated_at"] = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": update},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("post not found")
	}

	return nil
}

func (r *postRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return errors.New("post not found")
	}

	return nil
}

func (r *postRepository) AddComment(ctx context.Context, postID string, comment *models.Comment) error {
	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return err
	}

	comment.ID = primitive.NewObjectID()
	comment.CreatedAt = time.Now()
	comment.UpVotes = 0
	comment.DownVotes = 0
	comment.UpVoterIDs = []string{}
	comment.DownVoterIDs = []string{}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$push": bson.M{"comments": comment}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("post not found")
	}

	return nil
}

func (r *postRepository) DeleteComment(ctx context.Context, postID, commentID string) error {
	postObjectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return err
	}

	commentObjectID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return err
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": postObjectID},
		bson.M{"$pull": bson.M{"comments": bson.M{"_id": commentObjectID}}},
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("post not found")
	}

	return nil
}

func (r *postRepository) VotePost(ctx context.Context, postID, userID, action string) error {
	objectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return err
	}

	var update bson.M
	if action == "up" {
		update = bson.M{
			"$inc":      bson.M{"up_votes": 1},
			"$addToSet": bson.M{"up_voter_ids": userID},
		}
	} else {
		update = bson.M{
			"$inc":      bson.M{"down_votes": 1},
			"$addToSet": bson.M{"down_voter_ids": userID},
		}
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		update,
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("post not found")
	}

	return nil
}

func (r *postRepository) VoteComment(ctx context.Context, postID, commentID, userID, action string) error {
	postObjectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		return err
	}

	commentObjectID, err := primitive.ObjectIDFromHex(commentID)
	if err != nil {
		return err
	}

	var update bson.M
	if action == "up" {
		update = bson.M{
			"$inc":      bson.M{"comments.$[elem].up_votes": 1},
			"$addToSet": bson.M{"comments.$[elem].up_voter_ids": userID},
		}
	} else {
		update = bson.M{
			"$inc":      bson.M{"comments.$[elem].down_votes": 1},
			"$addToSet": bson.M{"comments.$[elem].down_voter_ids": userID},
		}
	}

	arrayFilters := options.ArrayFilters{
		Filters: []interface{}{
			bson.M{"elem._id": commentObjectID},
		},
	}

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": postObjectID},
		update,
		options.Update().SetArrayFilters(arrayFilters),
	)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return errors.New("post or comment not found")
	}

	return nil
}
