package model

import "time"

type Post struct {
	ID                  int64     `db:"id"`
	PostAuthor          int64     `db:"post_author"`
	PostDate            time.Time `db:"post_date"`
	PostDateGmt         time.Time `db:"post_date_gmt"`
	PostContent         string    `db:"post_content"`
	PostTitle           string    `db:"post_title"`
	PostExcerpt         string    `db:"post_excerpt"`
	PostStatus          string    `db:"post_status"`
	CommentStatus       string    `db:"comment_status"`
	PingStatus          string    `db:"ping_status"`
	PostPassword        string    `db:"post_password"`
	PostName            string    `db:"post_name"`
	ToPing              string    `db:"to_ping"`
	Pinged              string    `db:"pinged"`
	PostModified        time.Time `db:"post_modified"`
	PostModifiedGmt     time.Time `db:"post_modified_gmt"`
	PostContentFiltered string    `db:"post_content_filtered"`
	PostParent          int64     `db:"post_parent"`
	GUID                string    `db:"guid"`
	MenuOrder           int       `db:"menu_order"`
	PostType            string    `db:"post_type"`
	PostMimeType        string    `db:"post_mime_type"`
	CommentCount        int       `db:"comment_count"`
}
