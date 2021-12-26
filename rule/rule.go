package rule

import (
	"fmt"

	"github.com/turnage/graw/reddit"
)

var (
	ruleRegistry RuleRegistry
)

// A type that defines what a rule is.
type Rule interface {
	Name() string
	RegisterConfigs(configs []byte) error
	Match(post reddit.Post) bool
}

// A type to map rules keyed by their name.
type RuleRegistry map[string]Rule

// Register a rule for inclusion in the internal rule registry.
func RegisterRule(r Rule) {
	ruleRegistry[r.Name()] = r
}

// Look to see if the rule is in the internal rule registry.
func RuleInRuleRegistry(ruleName string) (Rule, error) {
	// The returned error is necessary otherwise other parts of the code will have to
	// guess the zero value of 'rule'.
	if rule, ok := ruleRegistry[ruleName]; ok {
		return rule, nil
	} else {
		return rule, fmt.Errorf("the following rule is not known: %v", ruleName)
	}
}

// Get some rules from the internal rule registry.
func GetRegisteredRules(ruleNames *[]string, rulesFound *[]Rule) error {
	for _, ruleName := range *ruleNames {
		if rule, err := RuleInRuleRegistry(ruleName); err == nil {
			*rulesFound = append(*rulesFound, rule)
		} else {
			return err
		}
	}

	return nil
}

// Get the internal rule registry.
func GetRuleRegistry() *RuleRegistry {
	return &ruleRegistry
}

func init() {
	ruleRegistry = make(RuleRegistry)
}

