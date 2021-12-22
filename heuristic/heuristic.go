package heuristic

import "github.com/turnage/graw/reddit"

type Heuristic struct {
	rules []Rule
}

// Apply rules to the posts passed in, storing the posts that matched each rule
// into matches.
func (h *Heuristic) appliedTo(posts *[]reddit.Post, matches *[]reddit.Post) {
	for _, post := range *posts {
		var postPasses bool = true
		for _, rule := range h.rules {
			if !rule.Match(post) {
				postPasses = false
			}
		}

		if postPasses {
			*matches = append(*matches, post)
		}
	}
}
