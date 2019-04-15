package slack

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type User struct {
	RealName string `json:"real_name"`
	Name     string `json:"name"`
	ID       string `json:"id"`
}

type UserInfo struct {
	User User `json:"user"`
}

type Group struct {
	Group struct {
		Members []string `json:"members"`
	} `json:"group"`
}

func GetUsers(token, channel string) (users map[string]User) {
	users = make(map[string]User)
	resp, err := http.Get(fmt.Sprintf("https://slack.com/api/groups.info?token=%s&channel=%s", token, channel))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var group Group
	err = json.Unmarshal(body, &group)

	var userInfo UserInfo
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
