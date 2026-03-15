package model

import "time"

type Comment struct {
	CommentID          int64     `db:"comment_id"`
	CommentPostID      int64     `db:"comment_post_id"`
	CommentAuthor      string    `db:"comment_author"`
	CommentAuthorEmail string    `db:"comment_author_email"`
	CommentAuthorURL   string    `db:"comment_author_url"`
	CommentAuthorIP    string    `db:"comment_author_ip"`
	CommentDate        time.Time `db:"comment_date"`
	CommentDateGmt     time.Time `db:"comment_date_gmt"`
	CommentContent     string    `db:"comment_content"`
	CommentKarma       int       `db:"comment_karma"`
	CommentApproved    string    `db:"comment_approved"`
	CommentAgent       string    `db:"comment_agent"`
	CommentType        string    `db:"comment_type"`
	CommentParent      int64     `db:"comment_parent"`
	UserID             int64     `db:"user_id"`
}
