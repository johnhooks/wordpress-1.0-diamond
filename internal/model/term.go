package model

type Term struct {
	TermID    int64  `db:"term_id"`
	Name      string `db:"name"`
	Slug      string `db:"slug"`
	TermGroup int64  `db:"term_group"`
}

type TermTaxonomy struct {
	TermTaxonomyID int64  `db:"term_taxonomy_id"`
	TermID         int64  `db:"term_id"`
	Taxonomy       string `db:"taxonomy"`
	Description    string `db:"description"`
	Parent         int64  `db:"parent"`
	Count          int64  `db:"count"`
}

type TermRelationship struct {
	ObjectID       int64 `db:"object_id"`
	TermTaxonomyID int64 `db:"term_taxonomy_id"`
	TermOrder      int   `db:"term_order"`
}

// Category is a convenience struct joining wp_terms + wp_term_taxonomy for taxonomy='category'.
type Category struct {
	TermID      int64  `db:"term_id"`
	Name        string `db:"name"`
	Slug        string `db:"slug"`
	TermGroup   int64  `db:"term_group"`
	TaxonomyID  int64  `db:"term_taxonomy_id"`
	Description string `db:"description"`
	Parent      int64  `db:"parent"`
	Count       int64  `db:"count"`
}
