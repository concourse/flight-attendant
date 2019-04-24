package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/concourse/flight-attendant/pagerduty"
	"github.com/concourse/flight-attendant/slack"
)

type timeResourceOutput struct {
	Version timeResourceVersion `json:"version"`
}

type timeResourceVersion struct {
	Time string `json:"time"`
}

const (
	today      = "Current"
	nextDay    = "Next"
	timeFormat = "Mon, Jan 02"
)

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

	timeframe := os.Getenv("CREW_TIMEFRAME")
	if timeframe == "" {
		log.Fatal(fmt.Errorf("Empty or unset environment variable CREW_TIMEFRAME"))
		os.Exit(1)
	}

	var scheduleDate time.Time
	now, err := readTime("input")
	if err != nil {
		log.Fatal(fmt.Errorf("Could not read time: %s", err))
		os.Exit(1)
	}

	switch timeframe {
	case today:
		scheduleDate = now
	case nextDay:
		scheduleDate = getNextWorkDay(now)
	default:
		log.Fatal(fmt.Errorf("CREW_TIMEFRAME must be one of: (%s|%s)", today, nextDay))
		os.Exit(1)
	}

	slackUsers := slack.GetUsers(slackToken, channel)
	onCallUsers := pagerduty.GetUsers(key, escalationPolicy, scheduleDate.Format(time.RFC3339))
	body := formatMessageBody(onCallUsers, slackUsers)

	concourseMessage(timeframe, scheduleDate.Format(timeFormat), body)
	wingsMessage(body)
}

func concourseMessage(timeframe, date, body string) {
	msg := fmt.Sprintf("%s on-call users for %s:\n", timeframe, date)
	msg += body

	err := ioutil.WriteFile("wings.txt", []byte(msg), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func wingsMessage(body string) {
	msg := "Good morning, your pilots (interrupt pair) for today are:\n"
	msg += body

	err := ioutil.WriteFile("wings.txt", []byte(msg), 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func formatMessageBody(users []pagerduty.User, slackUsers map[string]slack.User) string {
	msg := ""
	for _, u := range users {
		contactMethod := u.Email

		if _, ok := slackUsers[u.Name]; ok {
			contactMethod = fmt.Sprintf("<@%s>", slackUsers[u.Name].ID)
		}

		msg = msg + fmt.Sprintf("- %s ( %s )\n", u.Name, contactMethod)
	}
	return msg
}

func readTime(inputFileName string) (time.Time, error) {
	file, err := ioutil.ReadFile(inputFileName)
	if err != nil {
		return time.Now(), err
	}

	var version timeResourceOutput
	err = json.Unmarshal(file, &version)
	if err != nil {
		return time.Now(), err
	}

	return time.Parse(time.RFC3339, version.Version.Time)
}

func getNextWorkDay(now time.Time) time.Time {
	var delta = 1
	if now.Weekday() == time.Friday {
		delta = 3
	}
	return now.AddDate(0, 0, delta)
}
