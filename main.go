package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

type pagerdutyUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type slackUser struct {
	RealName string `json:"real_name"`
	Name     string `json:"name"`
}

type slackUserInfo struct {
	User slackUser `json:"user"`
}

type slackGroup struct {
	Group struct {
		Members []string `json:"members"`
	} `json:"group"`
}

type schedule struct {
	Oncalls []oncall `json:oncalls`
}

type oncall struct {
	User pagerdutyUser `json:"user"`
}

func main() {
	key := os.Getenv("PAGERDUTY_API_KEY")
	if key == "" {
		log.Fatal(fmt.Errorf("Empty or unset environment variable PAGERDUTY_API_KEY"))
		os.Exit(1)
	}

	escalationPolicy := os.Getenv("PAGERDUTY_ESCALATION_POLICY")
	if escalationPolicy == "" {
		log.Fatal(fmt.Errorf("Empty or unset environment variable PAGERDUTY_ESCALATION_POLICY"))
		os.Exit(1)
	}

	slackToken := os.Getenv("SLACK_TOKEN")
	if slackToken == "" {
		log.Fatal(fmt.Errorf("Empty or unset environment variable SLACK_TOKEN"))
		os.Exit(1)
	}

	channel := os.Getenv("SLACK_CHANNEL")
	if channel == "" {
		log.Fatal(fmt.Errorf("Empty or unset environment variable SLACK_CHANNEL"))
		os.Exit(1)
	}

	slackUsers := getSlackUsers(slackToken, channel)
	onCallUsers := getPagerdutyUsers(key, escalationPolicy)

	fmt.Print(message(onCallUsers, slackUsers))
}

func getSlackUsers(token, channel string) (users map[string]slackUser) {
	users = make(map[string]slackUser)
	resp, err := http.Get(fmt.Sprintf("https://slack.com/api/groups.info?token=%s&channel=%s", token, channel))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var group slackGroup
	err = json.Unmarshal(body, &group)

	var userInfo slackUserInfo
	for _, member := range group.Group.Members {
		r, err := http.Get(fmt.Sprintf("https://slack.com/api/users.info?token=%s&user=%s", token, member))
		if err != nil {
			log.Fatal(err)
		}

		body, err := ioutil.ReadAll(r.Body)

		err = json.Unmarshal(body, &userInfo)
		users[userInfo.User.RealName] = userInfo.User
		r.Body.Close()
	}
	return
}

func getPagerdutyUsers(key, escalationPolicy string) []pagerdutyUser {
	r, err := onCallSchedule(key, escalationPolicy)
	if err != nil {
		log.Fatal(err)
	}

	var users []pagerdutyUser
	seen := make(map[string]struct{}, len(r.Oncalls))
	for _, onCall := range r.Oncalls {
		if _, ok := seen[onCall.User.Email]; ok {
			continue
		}

		seen[onCall.User.Email] = struct{}{}
		users = append(users, onCall.User)
	}

	return users
}

func onCallSchedule(key, escalationPolicy string) (resp schedule, err error) {
	u, err := url.Parse(fmt.Sprintf("https://api.pagerduty.com/oncalls?time_zone=UTC&include[]=users&escalation_policy_ids[]=%s", escalationPolicy))
	if err != nil {
		return
	}
	req := &http.Request{
		Method: "GET",
		Header: map[string][]string{
			"Accept":        {"application/vnd.pagerduty+json;version=2"},
			"Authorization": {fmt.Sprintf("Token token=%s", key)},
		},
		URL: u,
	}

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer r.Body.Close()

	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		err = fmt.Errorf("%s: %s", r.Status, string(bodyBytes))
		return
	}

	err = json.Unmarshal(bodyBytes, &resp)

	return
}

func message(users []pagerdutyUser, slackUsers map[string]slackUser) string {
	msg := `Current on-call users:`
	for _, u := range users {
		contactMethod := u.Email

		if _, ok := slackUsers[u.Name]; ok {
			contactMethod = fmt.Sprintf("@%s", slackUsers[u.Name].Name)
		}

		msg = fmt.Sprintf(`%s
- %s ( %s )`, msg, u.Name, contactMethod)
	}
	return msg
}
