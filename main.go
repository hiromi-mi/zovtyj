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
	"context"
	"flag"
	"fmt"
	"github.com/mattn/go-mastodon"
	"log"
	"os"
	"time"
)

func doRegister(serverName string) *mastodon.Application {
	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:       serverName,
		ClientName:   "zovtyj",
		Scopes:       "read write",
		RedirectURIs: "urn:ietf:wg:oauth:2.0:oob",
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

	err = c.AuthenticateToken(context.Background(), authCode, "urn:ietf:wg:oauth:2.0:oob")
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
		//timeline, err := c.GetTimelineHome(context.Background(), nil)
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

func main() {
	register := flag.Bool("register", false, "Register New ID")
	history := flag.Bool("history", false, "Show History")
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
	}

	timeline, err := c.GetTimelineHome(context.Background(), &mastodon.Pagination{
		MaxID: "",
		Limit: 50,
	})
	if err != nil {
		log.Fatal(err)
	}
	location, err := time.LoadLocation("Local")
	for i := 0; i < len(timeline); i++ {
		fmt.Println(string(timeline[i].ID) + " " + timeline[i].CreatedAt.In(location).Format("2006-01-02 15:04:05") + " " + timeline[i].Content + " @" + timeline[i].Account.Username)
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
			fmt.Println(notifications[i].Type + " @" + notifications[i].Account.Username + " " + notifications[i].Status.Content)
		} else {
			fmt.Println(notifications[i].Type + " @" + notifications[i].Account.Username)
		}
	}
}
