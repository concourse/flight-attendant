package pagerduty

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Schedule struct {
	Oncalls []Oncall `json:oncalls`
}

type Oncall struct {
	User User `json:"user"`
}

func GetUsers(key, escalationPolicy, date string) []User {
	r, err := onCallSchedule(key, escalationPolicy, date)
	if err != nil {
		log.Fatal(err)
	}

	var users []User
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

func onCallSchedule(key, escalationPolicy, date string) (resp Schedule, err error) {
	u, err := url.Parse(fmt.Sprintf("https://api.pagerduty.com/oncalls?time_zone=UTC&include[]=users&escalation_policy_ids[]=%s&since=%s&until=%s", escalationPolicy, date, date))
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
