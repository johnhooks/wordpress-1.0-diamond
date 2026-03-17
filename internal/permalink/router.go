package permalink

import "strings"

// RouteType identifies what kind of content a URL resolves to.
type RouteType int

const (
	RoutePost RouteType = iota
	RouteCategory
	RouteDayArchive
	RouteMonthArchive
	RouteYearArchive
)

// Route is a resolved URL: a type and the values captured from the path.
type Route struct {
	Type  RouteType
	Match *Match
}

type rule struct {
	matcher   *matcher
	routeType RouteType
}

// Router holds an ordered list of rewrite rules compiled from the
// permalink structure. Resolve walks them top-to-bottom, first match wins.
type Router struct {
	rules []rule
}

// NewRouter compiles a permalink structure into rewrite rules for all
// content types: categories, posts, and date archives. When structure is
// empty (query-string permalinks), the router has no rules and Resolve
// always returns nil.
func NewRouter(structure string) *Router {
	if structure == "" {
		return &Router{}
	}

	var rules []rule

	// Category: front + "category/{slug}/"
	front := frontOf(structure)
	rules = append(rules, rule{
		matcher:   newMatcher(front + "category/%category%/"),
		routeType: RouteCategory,
	})

	// Post: the full permalink structure
	rules = append(rules, rule{
		matcher:   newMatcher(structure),
		routeType: RoutePost,
	})

	// Date archives — only generated when the structure contains the
	// relevant tag. More specific (day) before less specific (year).
	if strings.Contains(structure, "%day%") {
		rules = append(rules, rule{
			matcher:   newMatcher(truncateAfterTag(structure, "%day%")),
			routeType: RouteDayArchive,
		})
	}
	if strings.Contains(structure, "%monthnum%") {
		rules = append(rules, rule{
			matcher:   newMatcher(truncateAfterTag(structure, "%monthnum%")),
			routeType: RouteMonthArchive,
		})
	}
	if strings.Contains(structure, "%year%") {
		rules = append(rules, rule{
			matcher:   newMatcher(truncateAfterTag(structure, "%year%")),
			routeType: RouteYearArchive,
		})
	}

	return &Router{rules: rules}
}

// Resolve walks the rewrite rules and returns the first matching route.
// Returns nil if no rule matches (fall through to query-string handling).
func (rt *Router) Resolve(path string) *Route {
	for _, r := range rt.rules {
		if m := r.matcher.match(path); m != nil {
			return &Route{Type: r.routeType, Match: m}
		}
	}
	return nil
}
