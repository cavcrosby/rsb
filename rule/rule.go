package rule

import "github.com/turnage/graw/reddit"

type Rule interface {
	Name() string
	Match(post reddit.Post) bool
}
