package permalink

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Match holds values extracted from a URL path.
type Match struct {
	Year, Month, Day       string
	Hour, Minute, Second   string
	PostName               string
	PostID                 string
	Category               string
}

// matcher parses URL paths against a compiled permalink structure.
type matcher struct {
	re *regexp.Regexp
}

var tagPatterns = map[string]string{
	"%year%":     `(?P<year>\d{4})`,
	"%monthnum%": `(?P<monthnum>[0-9]{1,2})`,
	"%day%":      `(?P<day>[0-9]{1,2})`,
	"%hour%":     `(?P<hour>[0-9]{1,2})`,
	"%minute%":   `(?P<minute>[0-9]{1,2})`,
	"%second%":   `(?P<second>[0-9]{1,2})`,
	"%postname%":  `(?P<postname>[^/]+)`,
	"%post_id%":   `(?P<post_id>[0-9]+)`,
	"%category%":  `(?P<category>[^/]+)`,
}

// newMatcher compiles a permalink structure string into a regex-based URL matcher.
func newMatcher(structure string) *matcher {
	if structure == "" {
		return &matcher{}
	}

	// Build regex by walking the structure string
	var pattern strings.Builder
	pattern.WriteString("^")

	remaining := structure
	for remaining != "" {
		idx := strings.Index(remaining, "%")
		if idx < 0 {
			// No more tags — quote the rest as literal
			pattern.WriteString(regexp.QuoteMeta(remaining))
			break
		}

		// Quote literal text before the tag
		if idx > 0 {
			pattern.WriteString(regexp.QuoteMeta(remaining[:idx]))
		}

		// Find the closing %
		end := strings.Index(remaining[idx+1:], "%")
		if end < 0 {
			// Malformed tag — treat rest as literal
			pattern.WriteString(regexp.QuoteMeta(remaining[idx:]))
			break
		}

		tag := remaining[idx : idx+1+end+1]
		if p, ok := tagPatterns[tag]; ok {
			pattern.WriteString(p)
		} else {
			// Unknown tag — treat as literal
			pattern.WriteString(regexp.QuoteMeta(tag))
		}
		remaining = remaining[idx+1+end+1:]
	}

	pattern.WriteString("$")

	re := regexp.MustCompile(pattern.String())
	return &matcher{re: re}
}

// match parses a URL path against the compiled structure.
func (m *matcher) match(path string) *Match {
	if m.re == nil {
		return nil
	}

	submatch := m.re.FindStringSubmatch(path)
	if submatch == nil {
		return nil
	}

	result := &Match{}
	for i, name := range m.re.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}
		val := submatch[i]
		switch name {
		case "year":
			result.Year = val
		case "monthnum":
			result.Month = val
		case "day":
			result.Day = val
		case "hour":
			result.Hour = val
		case "minute":
			result.Minute = val
		case "second":
			result.Second = val
		case "postname":
			result.PostName = val
		case "post_id":
			result.PostID = val
		case "category":
			result.Category = val
		}
	}
	return result
}

// ValidateDate checks each non-empty date field against the post's actual date.
// Returns false on any mismatch.
func (m *Match) ValidateDate(postDate time.Time) bool {
	if m.Year != "" {
		if fmt.Sprintf("%04d", postDate.Year()) != m.Year {
			return false
		}
	}
	if m.Month != "" {
		v, _ := strconv.Atoi(m.Month)
		if v != int(postDate.Month()) {
			return false
		}
	}
	if m.Day != "" {
		v, _ := strconv.Atoi(m.Day)
		if v != postDate.Day() {
			return false
		}
	}
	if m.Hour != "" {
		v, _ := strconv.Atoi(m.Hour)
		if v != postDate.Hour() {
			return false
		}
	}
	if m.Minute != "" {
		v, _ := strconv.Atoi(m.Minute)
		if v != postDate.Minute() {
			return false
		}
	}
	if m.Second != "" {
		v, _ := strconv.Atoi(m.Second)
		if v != postDate.Second() {
			return false
		}
	}
	return true
}
