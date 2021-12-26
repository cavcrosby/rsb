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

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "github.com/cavcrosby/rsb/register"
	"github.com/cavcrosby/rsb/rule"
	"github.com/turnage/graw/reddit"
	"github.com/urfave/cli/v2"
)

const (
	progName = "rsb"
)

const (
	ModeFile       = 0x0
	OS_READ        = 04
	OS_WRITE       = 02
	OS_EX          = 01
	OS_USER_SHIFT  = 6
	OS_GROUP_SHIFT = 3
	OS_OTH_SHIFT   = 0
	OS_USER_R      = OS_READ << OS_USER_SHIFT
	OS_USER_W      = OS_WRITE << OS_USER_SHIFT
	OS_USER_X      = OS_EX << OS_USER_SHIFT
	OS_USER_RW     = OS_USER_R | OS_USER_W
	OS_USER_RWX    = OS_USER_RW | OS_USER_X
	OS_GROUP_R     = OS_READ << OS_GROUP_SHIFT
	OS_GROUP_W     = OS_WRITE << OS_GROUP_SHIFT
	OS_GROUP_X     = OS_EX << OS_GROUP_SHIFT
	OS_GROUP_RW    = OS_GROUP_R | OS_GROUP_W
	OS_GROUP_RWX   = OS_GROUP_RW | OS_GROUP_X
	OS_OTH_R       = OS_READ << OS_OTH_SHIFT
	OS_OTH_W       = OS_WRITE << OS_OTH_SHIFT
	OS_OTH_X       = OS_EX << OS_OTH_SHIFT
	OS_OTH_RW      = OS_OTH_R | OS_OTH_W
	OS_OTH_RWX     = OS_OTH_RW | OS_OTH_X
)

var (
	progConfig string = strings.Join([]string{progName, ".json"}, "")
)

// A custom callback handler in the event improper cli flag/flag arguments or
// arguments are passed in.
var CustomOnUsageErrorFunc cli.OnUsageErrorFunc = func(context *cli.Context, err error, isSubcommand bool) error {
	cli.ShowAppHelp(context)
	log.Panic(err)
	return err
}

// A type used to represent the configuration file of the program.
type configTree struct {
	RuleConfigs []RuleConfig `json:"rules"`
}

// A type used to serve as a frontend to allow certain rules to be selected
// for use and to modify the rule's behavior to some extent through custom
// configurations. This configuration is made available through configTree.
type RuleConfig struct {
	ID      string                 `json:"id"`
	Configs map[string]interface{} `json:"configs"`
}

// A type used to store command flag argument values and argument values.
type progConfigs struct {
	exportConfig     bool
	helpFlagPassedIn bool
	showConfigPath   bool
}

// Interpret the command arguments passed in. Saving particular flag/flag arguments
// of interest into 'pconfs'.
func (pconfs *progConfigs) parseCmdArgs() {
	var localOsArgs []string = os.Args

	for i, val := range localOsArgs {
		if i < 1 {
			continue
		} else if val == "--" {
			break
		} else if stringInArr(val, &[]string{"-h", "-help", "--help"}) {
			pconfs.helpFlagPassedIn = true
		}
	}

	app := &cli.App{
		Name:            progName,
		Usage:           "searchs Reddit posts and matches posts that meet known rules",
		UsageText:       strings.Join([]string{progName, " [global options]"}, ""),
		Description:     strings.Join([]string{progName, " - Reddit Search Bot"}, ""),
		HideHelpCommand: true,
		OnUsageError:    CustomOnUsageErrorFunc,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "export-config",
				Aliases:     []string{"e"},
				Usage:       "exports the specific program configuration file",
				Destination: &pconfs.exportConfig,
			},
			&cli.BoolFlag{
				Name:        "show-config-path",
				Aliases:     []string{"s"},
				Usage:       "displays the filesystem path to the program's configuration file",
				Destination: &pconfs.showConfigPath,
			},
		},
		Action: func(context *cli.Context) error {
			if context.NArg() > 0 {
				cli.ShowAppHelp(context)
				os.Exit(1)
			}

			return nil
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	app.Run(localOsArgs)
	if pconfs.helpFlagPassedIn {
		os.Exit(0)
	}
}

// Look to see if the string is in the string array.
func stringInArr(strArg string, arr *[]string) bool {
	for _, val := range *arr {
		if val == strArg {
			return true
		}
	}

	return false
}

// Retrieve the rules mentioned in the RuleConfigs, registering additional custom
// configurations for each rule if specified. Configurations are specific to each
// rule, meaning one configuration in one rule may not work in other rule.
func getRules(rcs *[]RuleConfig, rules *[]rule.Rule) error {
	for _, rc := range *rcs {
		if len(rc.Configs) > 0 {
			if configsData, err := json.Marshal(rc.Configs); err != nil {
				return err
			} else if rule, err := rule.RuleInRuleRegistry(rc.ID); err != nil {
				return err
			} else if err := rule.RegisterConfigs(configsData); err != nil {
				return err
			} else {
				*rules = append(*rules, rule)
			}
		} else {
			if rule, err := rule.RuleInRuleRegistry(rc.ID); err != nil {
				return err
			} else {
				*rules = append(*rules, rule)
			}
		}
	}

	return nil
}

// func matchRules(rules *[]rule.Rule, posts, matches *[]reddit.Post) {
// 	for _, post := range *posts {
// 		for _, rule := range *rules {
// 			if rule.Match(post) {
// 				*matches = append(*matches, post)
// 			}
// 		}
// 	}
// }

// Creates the default program configuration file.
func createDefaultProgConfig(progConfigDirPath, progConfig string) error {
	if _, err := os.Stat(progConfigDirPath); errors.Is(err, fs.ErrNotExist) {
		os.MkdirAll(progConfigDirPath, os.ModeDir|(OS_USER_R|OS_USER_W|OS_USER_X|OS_GROUP_R|OS_GROUP_X|OS_OTH_R|OS_OTH_X))
	}

	defaultConfigTree := &configTree{RuleConfigs: []RuleConfig{
		{
			ID:      "",
			Configs: map[string]interface{}{},
		},
	}}

	// use 4 spaces vs a tab character for indenting
	if defaultConfigTreeBytes, err := json.MarshalIndent(defaultConfigTree, "", "    "); err != nil {
		return err
	} else if err := ioutil.WriteFile(
		filepath.Join(progConfigDirPath, progConfig),
		defaultConfigTreeBytes,
		os.ModeDir|(OS_USER_R|OS_USER_W|OS_USER_X|OS_GROUP_R|OS_GROUP_X|OS_OTH_R|OS_OTH_X),
	); err != nil {
		return err
	}

	return nil
}

// Start the main program execution.
func main() {
	pconfs := &progConfigs{}
	pconfs.parseCmdArgs()

	configDirPath, err := os.UserConfigDir()
	if err != nil {
		log.Panic(err)
	}

	var progConfigPath string = filepath.Join(configDirPath, progName, progConfig)
	if _, err := os.Stat(progConfigPath); errors.Is(err, fs.ErrNotExist) {
		if err := createDefaultProgConfig(
			filepath.Join(configDirPath, progName),
			progConfig,
		); err != nil {
			log.Panic(err)
		}
	}

	switch {
	case pconfs.exportConfig:
		progConfigFd, err := os.Open(progConfigPath)
		if err != nil {
			log.Panic(err)
		}
		defer progConfigFd.Close()

		progConfigBytes, err := ioutil.ReadAll(progConfigFd)
		if err != nil {
			log.Panic(err)
		}

		fmt.Println(string(progConfigBytes))
	case pconfs.showConfigPath:
		fmt.Println(progConfigPath)
	default:
		progConfigFd, err := os.Open(progConfigPath)
		if err != nil {
			log.Panic(err)
		}
		defer progConfigFd.Close()

		progConfigBytes, err := ioutil.ReadAll(progConfigFd)
		if err != nil {
			log.Panic(err)
		}

		var ct configTree
		if err := json.Unmarshal(progConfigBytes, &ct); err != nil {
			log.Panic(err)
		}

		var rules []rule.Rule
		if err := getRules(&ct.RuleConfigs, &rules); err != nil {
			log.Panic(err)
		}
	}

	// ctData, err := json.Marshal(ct)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(string(ctData))

	bot, err := reddit.NewBotFromAgentFile("rsb.agent", 0)
	if err != nil {
		log.Panic(fmt.Errorf("Failed to create bot handle: %v", err))
	}

	// harvest, err := bot.Listing("/r/buildapcsales/", "")
	// if err != nil {
	// 	log.Panic(fmt.Errorf("Failed to fetch /r/buildapcsales/: %v", err))
	// }

	// for _, post := range harvest.Posts[:5] {
	// 	fmt.Printf("[%s] posted [%s]\n", post.Author, post.Title)
	// }
	os.Exit(0)
}
