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
	"net/smtp"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	_ "github.com/cavcrosby/rsb/register"
	"github.com/cavcrosby/rsb/rule"
	"github.com/turnage/graw"
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
	defaultAgentPath            = strings.Join([]string{"./", progName, ".agent"}, "")
	defaultPostThreshold        = 5
	errfoundPost         error  = errors.New("found a reddit post")
	progConfig           string = strings.Join([]string{progName, ".json"}, "")
)

// A custom callback handler in the event improper cli flag/flag arguments or
// arguments are passed in.
var CustomOnUsageErrorFunc cli.OnUsageErrorFunc = func(context *cli.Context, err error, isSubcommand bool) error {
	cli.ShowAppHelp(context)
	log.Panic(err)
	return err
}

// A type that represents a post handler for graw. Mainly meant to store posts
// received from the 'subreddit' event stream.
type postGather struct {
	bot             reddit.Bot
	postQueue       []*reddit.Post
	postThreshold   int
	stickyPostQueue map[string]string
}

// Empty out the post queue.
func (g *postGather) flushPostQueue() {
	g.postQueue = nil
}

// Return the post queue.
func (g *postGather) getPostQueue() []*reddit.Post {
	return g.postQueue
}

// Determine if post threshold is met in the post queue.
func (g *postGather) atPostThreshold() bool {
	return len(g.postQueue) >= g.postThreshold
}

func (g *postGather) Post(p *reddit.Post) error {
	if _, ok := g.stickyPostQueue[p.ID]; !p.Stickied || !ok {
		g.postQueue = append(g.postQueue, p)
	}

	return errfoundPost
}

// A type used to represent the configuration file of the program.
//
// Example (includes RuleConfig(s)):
// {
//     "sendmail_from": "foo@bar.com",
//     "sendmail_to": "baz@bar.com",
//     "password": "foobarbaz",
//     "smtp_addr": "smtp.bar.com",
//     "smtp_port": "1234",
//     "rules": [
//         {
//             "id": "ramunderprice",
//             "configs": {
//                 "price": 100
//             }
//         }
//     ]
// }
//
type configTree struct {
	SendMailFrom string       `json:"sendmail_from"`
	SendMailTo   string       `json:"sendmail_to"`
	Password     string       `json:"password"`
	SmtpAddr     string       `json:"smtp_addr"`
	SmtpPort     string       `json:"smtp_port"`
	RuleConfigs  []RuleConfig `json:"rules"`
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
	agentPath        string
	altConfigPath    string
	exportConfig     bool
	helpFlagPassedIn bool
	showConfigPath   bool
	subredditName    string
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
		} else if stringInArr(val, []string{"-h", "-help", "--help"}) {
			pconfs.helpFlagPassedIn = true
		}
	}

	app := &cli.App{
		Name:            progName,
		Usage:           "searches Reddit posts and matches posts that meet known rules",
		UsageText:       strings.Join([]string{progName, " [global options] SUBREDDIT_NAME"}, ""),
		Description:     strings.Join([]string{progName, " - A (for) Reddit Search Bot"}, ""),
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
				Usage:       "displays the filesystem path to the program's default configuration file",
				Destination: &pconfs.showConfigPath,
			},
			&cli.PathFlag{
				Name:        "config-path",
				Aliases:     []string{"c"},
				Value:       pconfs.altConfigPath,
				Usage:       "alternative `PATH` for the program's configuration file",
				Destination: &pconfs.altConfigPath,
			},
			&cli.PathFlag{
				Name:        "agent-path",
				Aliases:     []string{"a"},
				Value:       defaultAgentPath,
				Usage:       "alternative `PATH` for agent configuration file",
				Destination: &pconfs.agentPath,
			},
		},
		Action: func(context *cli.Context) error {
			if context.NArg() < 1 && !pconfs.showConfigPath && !pconfs.exportConfig {
				cli.ShowAppHelp(context)
				log.Panic(errors.New("SUBREDDIT_NAME argument is required"))
			}

			pconfs.subredditName = context.Args().Get(0)
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
func stringInArr(strArg string, arr []string) bool {
	for _, val := range arr {
		if val == strArg {
			return true
		}
	}

	return false
}

// Retrieve the rules mentioned in the RuleConfigs, registering additional custom
// configurations for each rule if specified. Configurations are specific to each
// rule, meaning one configuration in one rule may not work in other rule.
func getRules(rcs []RuleConfig) ([]rule.Rule, error) {
	var rules []rule.Rule
	for _, rc := range rcs {
		if len(rc.Configs) > 0 {
			if configsData, err := json.Marshal(rc.Configs); err != nil {
				return rules, err
			} else if rule, err := rule.RuleInRuleRegistry(rc.ID); err != nil {
				return rules, err
			} else if err := rule.RegisterConfigs(configsData); err != nil {
				return rules, err
			} else {
				rules = append(rules, rule)
			}
		} else {
			if rule, err := rule.RuleInRuleRegistry(rc.ID); err != nil {
				return rules, err
			} else {
				rules = append(rules, rule)
			}
		}
	}

	return rules, nil
}

// Test each reddit post passed in to see if a post matches any of the rules passed
// in. If a post matches any rule, then said post will be aggregated with others
// that match a rule.
func matchPosts(rules []rule.Rule, posts []*reddit.Post) map[string]*reddit.Post {
	var matches = make(map[string]*reddit.Post)
	for _, post := range posts {
		for _, rule := range rules {
			if rule.Match(post) {
				matches[rule.Name()] = post
			}
		}
	}

	return matches
}

// Send a test email to the intended recipient to ensure smtp is functional.
// Returns the authentication struct for the sender.
func initSmtp(ct configTree) (smtp.Auth, error) {
	// Set up authentication information.
	auth := smtp.PlainAuth("", ct.SendMailFrom, ct.Password, ct.SmtpAddr)

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	to := []string{ct.SendMailTo}
	msg := []byte(strings.Join(
		[]string{
			fmt.Sprintf("To: %v", ct.SendMailTo),
			fmt.Sprintf("Subject: Initializing %v", progName),
			"",
			"foo",
		},
		"\r\n",
	))
	if err := smtp.SendMail(ct.SmtpAddr+":"+ct.SmtpPort, auth, ct.SendMailFrom, to, msg); err != nil {
		return nil, err
	}

	return auth, nil
}

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
		if pconfs.altConfigPath != "" {
			progConfigPath = pconfs.altConfigPath
		}
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
		smtpAuth, err := initSmtp(ct)
		if err != nil {
			log.Panic(fmt.Errorf("%v: failed to initialize smtp: %v", progName, err))
		}

		rules, err := getRules(ct.RuleConfigs)
		if err != nil {
			log.Panic(err)
		}

		bot, err := reddit.NewBotFromAgentFile(pconfs.agentPath, 0)
		if err != nil {
			log.Panic(fmt.Errorf("%v: failed to create bot handle: %v", progName, err))
		}

		// DISCUSS(cavcrosby): each subreddit might require a different polling strategy
		// than from another. Look into implementing this per subreddit.
		cfg := graw.Config{Subreddits: []string{pconfs.subredditName}}
		handler := &postGather{
			bot:           bot,
			postThreshold: defaultPostThreshold,
		}

		to := []string{ct.SendMailTo}
		for {
			if _, wait, err := graw.Run(handler, bot, cfg); err != nil {
				log.Panic(fmt.Errorf("%v: graw run failed", progName))
			} else if err := wait(); err != errfoundPost {
				log.Panic(fmt.Errorf("%v: an error occurred for the graw post handler: %v", progName, err))
			}

			if handler.atPostThreshold() {
				postQueue := handler.getPostQueue()
				handler.flushPostQueue()
				var postUrls []string
				for i, post := range postQueue {
					postUrls = append(postUrls, strconv.Itoa(i+1)+". "+post.URL)
				}

				msgStr := strings.Join(
					append(
						[]string{
							fmt.Sprintf("To: %v", ct.SendMailTo),
							fmt.Sprintf("Subject: %v Report: \"%v\"", progName, pconfs.subredditName),
							"",
							"Posts:",
						},
						postUrls...,
					),
					"\r\n",
				)

				matches := matchPosts(rules, handler.getPostQueue())
				var matchUrls []string
				var matchCounter int = 1
				for ruleId, post := range matches {
					matchUrls = append(matchUrls, strconv.Itoa(matchCounter)+"("+ruleId+"). "+post.URL)
					matchCounter += 1
				}

				msg := []byte(msgStr + strings.Join(
					append(
						[]string{
							"Matches:",
						},
						matchUrls...,
					),
					"\r\n",
				))
				if err := smtp.SendMail(ct.SmtpAddr+":"+ct.SmtpPort, smtpAuth, ct.SendMailFrom, to, msg); err != nil {
					log.Panic(err)
				}
			}
		}
	}
}
