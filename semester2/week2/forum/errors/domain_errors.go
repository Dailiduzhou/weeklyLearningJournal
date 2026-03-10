package errors

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUnauthorized       = errors.New("unauthorized access")
	ErrPostNotFound       = errors.New("post not found")
	ErrCommentNotFound    = errors.New("comment not found")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidVoteAction  = errors.New("invalid vote action")
	ErrAlreadyVoted       = errors.New("user has already voted")
)
