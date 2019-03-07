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

type user struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type schedule struct {
	Oncalls []oncall `json:oncalls`
}

type oncall struct {
	User user `json:"user"`
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

	r, err := onCallSchedule(key, escalationPolicy)
	if err != nil {
		log.Fatal(err)
	}

	var users []user
	seen := make(map[string]struct{}, len(r.Oncalls))
	for _, onCall := range r.Oncalls {
		if _, ok := seen[onCall.User.Email]; ok {
			continue
		}

		seen[onCall.User.Email] = struct{}{}
		users = append(users, onCall.User)
	}

	fmt.Print(string(message(users)))
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

func message(users []user) []byte {
	msg := `Current on-call users:`
	for _, u := range users {
		msg = fmt.Sprintf(`%s
- %s (%s)`, msg, u.Name, u.Email)
	}
	return []byte(msg)
}
