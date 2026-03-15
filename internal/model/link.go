package model

import "time"

type Link struct {
	LinkID          int64     `db:"link_id"`
	LinkURL         string    `db:"link_url"`
	LinkName        string    `db:"link_name"`
	LinkImage       string    `db:"link_image"`
	LinkTarget      string    `db:"link_target"`
	LinkDescription string    `db:"link_description"`
	LinkVisible     string    `db:"link_visible"`
	LinkOwner       int64     `db:"link_owner"`
	LinkRating      int       `db:"link_rating"`
	LinkUpdated     time.Time `db:"link_updated"`
	LinkRel         string    `db:"link_rel"`
	LinkNotes       string    `db:"link_notes"`
	LinkRSS         string    `db:"link_rss"`
}
