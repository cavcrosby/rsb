package main

import (
	"fmt"
	"log"

	"github.com/turnage/graw/reddit"
)

func main() {
	bot, err := reddit.NewBotFromAgentFile("rsb.agent", 0)
	if err != nil {
		log.Panic(fmt.Errorf("Failed to create bot handle: %v", err))
	}

	harvest, err := bot.Listing("/r/buildapcsales/", "")
	if err != nil {
		log.Panic(fmt.Errorf("Failed to fetch /r/buildapcsales/: %v", err))
	}

	for _, post := range harvest.Posts[:5] {
		fmt.Printf("[%s] posted [%s]\n", post.Author, post.Title)
	}
}
