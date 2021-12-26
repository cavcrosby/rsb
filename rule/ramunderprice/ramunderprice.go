package ramunderprice

import (
	"encoding/json"
	"log"
	"regexp"
	"strconv"

	"github.com/cavcrosby/rsb/rule"
	"github.com/turnage/graw/reddit"
)

var (
	defaultPrice int = 0
	reRamInTitle  = regexp.MustCompile(`(?i)\bRAM\b`)
	reCostInTitle = regexp.MustCompile(`^\$\d+\.*\d*$`)
)

type RamUnderPrice struct {
	Price int `json:"price"`
}

func (r *RamUnderPrice) Name() string {
	return "ramunderprice"
}

func (r *RamUnderPrice) RegisterConfigs(configs []byte) error {
	if err := json.Unmarshal(configs, r); err != nil {
		return err
	}

	return nil
}

func (r *RamUnderPrice) Match(post reddit.Post) bool {
	if reRamInTitle.FindStringIndex(post.Title) == nil {
		return false
	}

	var allSubStrings int = -1
	costs := reCostInTitle.FindAllString(post.Title, allSubStrings)
	if len(costs) != 1 {
		// TODO(cavcrosby): return false but there numerous reasons why there might exist
		// more than one "cost" in the title and we may wish to include those cases (e.g.
		// price difference from msrp minus discount could be under 100). Obviously 0
		// costs found should not have the rule match.
		return false
	}

	if cost, err := strconv.Atoi(regexp.MustCompile(`\d+$`).FindAllString(costs[0], -1)[0]); err != nil {
		log.Panic(err)
	} else if cost > r.Price {
		return false
	}

	return true
}

func init() {
	var ramUnderPrice *RamUnderPrice = &RamUnderPrice{
		Price: defaultPrice,	
	}

	rule.RegisterRule(ramUnderPrice)
}
