// Package permalink builds post and archive URLs from a WordPress-style
// permalink structure string. When the structure is empty, URLs fall back
// to the WP 1.0 default query-string format (?p=<id>).
package permalink

import (
	"fmt"
	"strings"
	"time"
)

// Linker builds URLs from a permalink structure and site URL.
type Linker struct {
	SiteURL   string // e.g. "http://localhost:3000"
	Structure string // e.g. "/%year%/%monthnum%/%day%/%postname%/"
}

// Post returns the permalink for a post. If the structure is empty,
// falls back to ?p=<id>.
func (l *Linker) Post(id int64, date time.Time, postName string) string {
	if l.Structure == "" {
		return fmt.Sprintf("%s/?p=%d", l.SiteURL, id)
	}

	r := strings.NewReplacer(
		"%year%", fmt.Sprintf("%04d", date.Year()),
		"%monthnum%", fmt.Sprintf("%02d", int(date.Month())),
		"%day%", fmt.Sprintf("%02d", date.Day()),
		"%hour%", fmt.Sprintf("%02d", date.Hour()),
		"%minute%", fmt.Sprintf("%02d", date.Minute()),
		"%second%", fmt.Sprintf("%02d", date.Second()),
		"%postname%", postName,
		"%post_id%", fmt.Sprintf("%d", id),
	)
	return l.SiteURL + r.Replace(l.Structure)
}

// PostPath returns just the path portion (no site URL) for a post.
func (l *Linker) PostPath(id int64, date time.Time, postName string) string {
	if l.Structure == "" {
		return fmt.Sprintf("/?p=%d", id)
	}

	r := strings.NewReplacer(
		"%year%", fmt.Sprintf("%04d", date.Year()),
		"%monthnum%", fmt.Sprintf("%02d", int(date.Month())),
		"%day%", fmt.Sprintf("%02d", date.Day()),
		"%hour%", fmt.Sprintf("%02d", date.Hour()),
		"%minute%", fmt.Sprintf("%02d", date.Minute()),
		"%second%", fmt.Sprintf("%02d", date.Second()),
		"%postname%", postName,
		"%post_id%", fmt.Sprintf("%d", id),
	)
	return r.Replace(l.Structure)
}

// Month returns the URL for a month archive. Extracts the portion of
// the structure up through %monthnum%.
func (l *Linker) Month(year int, month int) string {
	if l.Structure == "" {
		return fmt.Sprintf("%s/?m=%04d%02d", l.SiteURL, year, month)
	}

	path := truncateAfterTag(l.Structure, "%monthnum%")
	r := strings.NewReplacer(
		"%year%", fmt.Sprintf("%04d", year),
		"%monthnum%", fmt.Sprintf("%02d", month),
	)
	return l.SiteURL + r.Replace(path)
}

// Day returns the URL for a day archive. Extracts the portion of
// the structure up through %day%.
func (l *Linker) Day(year, month, day int) string {
	if l.Structure == "" {
		return fmt.Sprintf("%s/?m=%04d%02d%02d", l.SiteURL, year, month, day)
	}

	path := truncateAfterTag(l.Structure, "%day%")
	r := strings.NewReplacer(
		"%year%", fmt.Sprintf("%04d", year),
		"%monthnum%", fmt.Sprintf("%02d", month),
		"%day%", fmt.Sprintf("%02d", day),
	)
	return l.SiteURL + r.Replace(path)
}

// Category returns the URL for a category archive.
// With pretty permalinks: uses the front of the structure + "category/" + slug.
// Without: falls back to ?cat=<id>.
func (l *Linker) Category(id int64, slug string) string {
	if l.Structure == "" {
		return fmt.Sprintf("%s/?cat=%d", l.SiteURL, id)
	}

	front := frontOf(l.Structure)
	return l.SiteURL + front + "category/" + slug + "/"
}

// truncateAfterTag returns the structure up to and including the given tag,
// with a trailing slash. If the tag isn't found, returns the full structure.
func truncateAfterTag(structure, tag string) string {
	idx := strings.Index(structure, tag)
	if idx < 0 {
		return structure
	}
	path := structure[:idx+len(tag)]
	// Ensure trailing slash
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

// frontOf returns everything before the first % tag in the structure.
func frontOf(structure string) string {
	idx := strings.Index(structure, "%")
	if idx < 0 {
		return structure
	}
	return structure[:idx]
}
