package main

import (
	"fmt"
	"github.com/jzelinskie/reddit"
	"strings"
	"time"
)

type tldrItem struct {
	Content string
	Id      string
}

func main() {
	posts, err := reddit.SubredditHeadlines("askreddit")
	if err != nil {
		panic(err)
	}

	for _, post := range posts {
		// Sleep for 2 seconds as a niave way to keep the number of hits down to a max of 30 a min		
		time.Sleep(2 * time.Second)
		comments, err := reddit.GetComments(post)
		if err != nil {
			panic(err)
		}
		// Simple search and print of tldr comments
		for _, comment := range comments {
			formattedBody := strings.ToLower(comment.Body)
			found, sentence := extractTLDR(formattedBody)
			if found {
				fmt.Println(tldrItem{Content: sentence, Id: comment.FullId})
			}
		}
	}
}

func extractTLDR(body string) (bool, string) {
	bodyContent := strings.Fields(body)
	// If tl;dr or tl dr exists within the body of the string
	// stop at that index and extract out until either the end of 
	// the string or the first period found
	for i := 0; i < len(bodyContent); i++ {
		if bodyContent[i] == "tl;dr" || bodyContent[i] == "tldr" {
			tldrSentence := bodyContent[i]
			// Only continue if there is data after the tldr string
			if i < len(bodyContent)-1 {
				for j := i + 1; j < len(bodyContent); j++ {
					tldrSentence = fmt.Sprintf("%s %s", tldrSentence, bodyContent[j])
					if strings.Contains(bodyContent[j], ".") {
						return true, tldrSentence
					}
				}
				return true, tldrSentence
			}
		}
	}
	return false, ""
}
