package main

import (
	"container/list"
	"fmt"
	"github.com/jzelinskie/reddit"
	"github.com/kurrik/oauth1a"
	"github.com/kurrik/twittergo"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type tldrItem struct {
	Content string
	Author  string
	Created float64
}

func main() {
	posts, err := reddit.SubredditHeadlines("askreddit")
	handleError(err)

	tldrItemList := list.New()
	for _, post := range posts {
		// Sleep for 2 seconds as a niave way to keep the number of hits down to a max of 30 a min		
		time.Sleep(2 * time.Second)
		comments, err := reddit.GetComments(post)
		handleError(err)
		// Simple search and print of tldr comments
		for _, comment := range comments {
			formattedBody := strings.ToLower(comment.Body)
			found, sentence := extractTLDR(formattedBody)
			// Only add tweetable items to the list of candidates, i.e. less than 140 characters
			if found && len(sentence) < 140 {
				foundItem := tldrItem{Content: sentence, Author: comment.Author, Created: comment.Created}
				fmt.Printf("%v\n", foundItem)
				tldrItemList.PushBack(foundItem)
			}
		}
	}

	if tldrItemList.Len() > 0 {
		tweetItem := tldrItemList.Front()
		client := LogIn()
		message := tweetItem.Value.(tldrItem).Content
		fmt.Println(message)
		TweetMessage(message, client)
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

// The login, loading of credentials, and tweeting has been addapted from the example provided with github.com/kurrik/twittergo
func LoadTwitterCredentials() (client *twittergo.Client, err error) {
	credentials, err := ioutil.ReadFile("CREDENTIALS")
	handleError(err)
	lines := strings.Split(string(credentials), "\n")
	config := &oauth1a.ClientConfig{
		ConsumerKey:    lines[0],
		ConsumerSecret: lines[1],
	}
	user := oauth1a.NewAuthorizedConfig(lines[2], lines[3])
	client = twittergo.NewClient(config, user)
	return
}

func LogIn() *twittergo.Client {
	client, err := LoadTwitterCredentials()
	handleError(err)
	return client
}

func TweetMessage(message string, client *twittergo.Client) {
	data := url.Values{}
	data.Set("status", message)
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", "/1.1/statuses/update.json", body)
	handleError(err)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.SendRequest(req)
	handleError(err)
}

func handleError(err error) {
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
}
