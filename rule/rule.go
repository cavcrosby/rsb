// Copyright (c) 2021 Conner Crosby
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

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
func GetRegisteredRules(ruleNames []string) ([]Rule, error) {
	var rulesFound []Rule
	for _, ruleName := range ruleNames {
		if rule, err := RuleInRuleRegistry(ruleName); err == nil {
			rulesFound = append(rulesFound, rule)
		} else {
			return rulesFound, err
		}
	}

	return rulesFound, nil
}

// Get the internal rule registry.
func GetRuleRegistry() *RuleRegistry {
	return &ruleRegistry
}

func init() {
	ruleRegistry = make(RuleRegistry)
}

