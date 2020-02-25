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

func doDirectTimeLine(c *mastodon.Client) {
	convs, err := c.GetConversations(context.Background(), &mastodon.Pagination{
		//MaxID: "",
		Limit: 20,
	})
	if err != nil {
		log.Fatal(err)
	}

	location, err := time.LoadLocation("Local")

	var crawler func(replyto mastodon.ID)
	crawler = func(replyto mastodon.ID) {
		if replyto == "" {
			return
		}
		reply, err := c.GetStatus(context.Background(), replyto)
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(reply)
		fmt.Println(string(reply.ID) + " " + reply.CreatedAt.In(location).Format("2006-01-02 15:04:05") + " " + readHtml(reply.Content) + " @" + reply.Account.Username)
		if reply.InReplyToID != nil {
			crawler(mastodon.ID(reply.InReplyToID.(string)))
		}
	}

	for i := 0; i < len(convs); i++ {
		fmt.Println(string(convs[i].LastStatus.ID) + " " + convs[i].LastStatus.CreatedAt.In(location).Format("2006-01-02 15:04:05") + " " + readHtml(convs[i].LastStatus.Content) + " @" + convs[i].LastStatus.Account.Username)
		if convs[i].LastStatus.InReplyToID != nil {
			crawler(mastodon.ID(convs[i].LastStatus.InReplyToID.(string)))
		}
		fmt.Println("\n")
	}
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
		fmt.Println(string(timeline[i].ID) + " " + timeline[i].CreatedAt.In(location).Format("2006-01-02 15:04:05") + opts + " " + readHtml(timeline[i].Content) + " @" + timeline[i].Account.Username)
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
			fmt.Println(notifications[i].Type + "@" + notifications[i].Account.Username + " " + readHtml(notifications[i].Status.Content))
		} else {
			fmt.Println(notifications[i].Type + "@" + notifications[i].Account.Username)
		}
	}
}

func dotoot(c *mastodon.Client, sensitiveMessage string, visibility string, replyid mastodon.ID) {
	scanner := bufio.NewScanner(os.Stdin)
	var toot string
	for scanner.Scan() {
		toot += scanner.Text()
		toot += "\n"
	}
	_, err := c.PostStatus(context.Background(), &mastodon.Toot{
		Status:      toot,
		SpoilerText: sensitiveMessage,
		InReplyToID: replyid,
		Visibility:  visibility,
		Sensitive:   sensitiveMessage != "",
	})
	if err != nil {
		log.Fatal(err)
	}
}

func dodelete(c *mastodon.Client, id mastodon.ID) {
	err := c.DeleteStatus(context.Background(), id)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	serverURL := flag.String("server", "", "Server URL")

	historyCmd := flag.NewFlagSet("history", flag.ContinueOnError)
	initID := historyCmd.String("initid", "", "Initial ID")
	userID := historyCmd.String("userid", "", "User ID")

	tootCmd := flag.NewFlagSet("toot", flag.ContinueOnError)
	tootReplyID := tootCmd.String("replyid", "", "Reply to ID")
	tootSensitive := tootCmd.String("sensitive", "", "Call Sensitive (Insert Warn Message)")
	tootVisibility := tootCmd.String("visibility", "private", "Visibility private/direct/unlisted/public")

	deleteCmd := flag.NewFlagSet("delete", flag.ContinueOnError)
	deleteID := deleteCmd.String("deleteid", "", "Status to delete")

	flag.Usage = func() {
		fmt.Fprintln(historyCmd.Output(), "Usage: zovtyj [global args] <command> [command args]")
		fmt.Fprintln(historyCmd.Output(), "global args:")
		flag.PrintDefaults()
		fmt.Fprintln(historyCmd.Output(), "commands: history, home, toot, delete, direct")
		fmt.Fprintln(historyCmd.Output(), "command args of history:")
		historyCmd.PrintDefaults()
		fmt.Fprintln(historyCmd.Output(), "command args of toot:")
		tootCmd.PrintDefaults()
		fmt.Fprintln(historyCmd.Output(), "command args of delete:")
		deleteCmd.PrintDefaults()
	}

	/* Parse Common Arguments */
	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(-1)
	}

	remainingArgs := flag.Args()

	if remainingArgs[0] == "register" {
		doRegister(*serverURL)
		os.Exit(-1)
	}

	c := mastodon.NewClient(&mastodon.Config{
		Server:       *serverURL,
		ClientID:     os.Getenv("CLIENTID"),
		ClientSecret: os.Getenv("CLIENTSECRET"),
		AccessToken:  os.Getenv("ACCESSTOKEN"),
	})

	switch remainingArgs[0] {
	case "history":
		historyCmd.Parse(remainingArgs[1:])
		dohistory(c, *userID, *initID)
	case "toot":
		tootCmd.Parse(remainingArgs[1:])
		/*
			var visibility mastodon.Visibility
			switch *tootVisibility {
			case "direct":
				visibility = mastodon.VisibilityDirectMessage
			case "private":
				visibility = mastodon.VisibilityFollowersOnly
			case "unlisted":
				visibility = mastodon.VisibilityUnlisted
			case "public":
				visibility = mastodon.VisibilityPublic
			default:
				log.Fatal("Unknown visibility: " + *tootVisibility)
			}*/
		dotoot(c, *tootSensitive, *tootVisibility, mastodon.ID(*tootReplyID))
	case "home":
		doHomeTimeline(c)
	case "delete":
		deleteCmd.Parse(remainingArgs[1:])
		dodelete(c, mastodon.ID(*deleteID))
	case "direct":
		doDirectTimeLine(c)
	default:
		log.Fatal("Please use cmd: " + flag.Args()[0])
	}
}
