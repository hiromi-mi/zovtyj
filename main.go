// License: Apache License Version 2.0 (See LICENSE)
//
//   Copyright hiromi-mi 2020
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/mattn/go-mastodon"
	"golang.org/x/net/html"
	"log"
	"os"
	"strings"
	"time"
)

func doRegister(serverName string) *mastodon.Application {
	const redirectURI = "urn:ietf:wg:oauth:2.0:oob"
	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:       serverName,
		ClientName:   "zovtyj",
		Scopes:       "read write",
		RedirectURIs: redirectURI,
		Website:      "https://github.com/hiromi-mi/zovtyj",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Client ID: %s\n", app.ClientID)
	fmt.Printf("Client Secret: %s\n", app.ClientSecret)
	fmt.Printf("Open this URI To Auth: %s\nInsert Token: ", app.AuthURI)
	var authCode string
	_, err = fmt.Scanf("%s\n", &authCode)
	if err != nil {
		log.Fatal(err)
	}
	c := mastodon.NewClient(&mastodon.Config{
		Server:       serverName,
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
	})

	err = c.AuthenticateToken(context.Background(), authCode, redirectURI)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Access Token: %s\n", c.Config.AccessToken)
	return app
}

func dohistory(c *mastodon.Client, userID string, initID string) {
	var InitialID = mastodon.ID(initID)
	location, err := time.LoadLocation("Local")
	if err != nil {
		log.Fatal(err)
	}

	for {
		statuses, err := c.GetAccountStatuses(context.Background(), mastodon.ID(userID), &mastodon.Pagination{
			MaxID: InitialID,
			Limit: 50,
		})

		if err != nil {
			log.Fatal(err)
		}
		for i := 0; i < len(statuses); i++ {
			fmt.Println(string(statuses[i].ID) + " " + statuses[i].CreatedAt.In(location).Format("2006-01-02 15:04:05") + " " + statuses[i].Content)
		}
		if len(statuses) <= 0 {
			// Exit toot
			break
		}
		InitialID = statuses[len(statuses)-1].ID
		time.Sleep(time.Millisecond * 1200)
	}
}

func readHtml(content string) string {
	doc, err := html.Parse(strings.NewReader(content))
	var b strings.Builder
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Input should be UTF-8 encoded
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		// node.Data : type
		if node.Type == html.ElementNode && node.Data == "p" {
			for _, a := range node.Attr {
				b.WriteString(a.Key)
				switch a.Key {
				case "href":
					b.WriteString(a.Val)
					b.WriteString(" ")
				}
			}
		}
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			crawler(c)
		}
	}

	crawler(doc)
	return b.String()
}

func doHomeTimeline(c *mastodon.Client) {
	timeline, err := c.GetTimelineHome(context.Background(), &mastodon.Pagination{
		MaxID: "",
		Limit: 50,
	})
	if err != nil {
		log.Fatal(err)
	}
	location, err := time.LoadLocation("Local")
	var opts string
	for i := 0; i < len(timeline); i++ {
		opts = ""
		if timeline[i].Reblog != nil {
			opts += " from " + timeline[i].Reblog.Account.Username + ": "
		}
		fmt.Println(string(timeline[i].ID) + " " + timeline[i].CreatedAt.In(location).Format("2006-01-02 15:04:05") + opts + " " + readHtml(timeline[i].Content) + "@" + timeline[i].Account.Username)
	}

	notifications, err := c.GetNotifications(context.Background(), &mastodon.Pagination{
		MaxID: "",
		Limit: 10,
	})
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < len(notifications); i++ {
		if notifications[i].Status != nil {
			fmt.Println(notifications[i].Type + " @" + notifications[i].Account.Username + " " + readHtml(notifications[i].Status.Content))
		} else {
			fmt.Println(notifications[i].Type + " @" + notifications[i].Account.Username)
		}
	}
}

func dotoot(c *mastodon.Client) {
	scanner := bufio.NewScanner(os.Stdin)
	var toot string
	for scanner.Scan() {
		toot += scanner.Text()
		toot += "\n"
	}
	_, err := c.PostStatus(context.Background(), &mastodon.Toot{
		Status:     toot,
		Visibility: mastodon.VisibilityFollowersOnly,
		Sensitive:  false,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	register := flag.Bool("register", false, "Register New ID")
	history := flag.Bool("history", false, "Show History")
	toot := flag.Bool("toot", false, "Toot")
	serverURL := flag.String("server", "", "Server URL")
	initID := flag.String("initid", "", "Initial ID")
	userID := flag.String("userid", "", "User ID")

	flag.Parse()
	if *register {
		doRegister(*serverURL)
		return
	}

	c := mastodon.NewClient(&mastodon.Config{
		Server:       *serverURL,
		ClientID:     os.Getenv("CLIENTID"),
		ClientSecret: os.Getenv("CLIENTSECRET"),
		AccessToken:  os.Getenv("ACCESSTOKEN"),
	})

	if *history {
		dohistory(c, *userID, *initID)
		return
	}

	if *toot {
		dotoot(c)
		return
	}

	doHomeTimeline(c)
}
