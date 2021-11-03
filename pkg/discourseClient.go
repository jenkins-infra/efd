package pkg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// MaxQueryFailure define how many we retry a failing http request
	MaxQueryFailure int = 10

	// HTTPHeaderContentType defines default contentType when querying discourse API
	HTTPHeaderContentType string = "application/json"

	// HTTPHeaderAuthorization defines default authorization type when authenticating with discourse API
	HTTPHeaderAuthorization string = "Bearer Token"
)

var (
	// ErrGroupHasNoMembers is returned when a Discouse group doesn't contain members
	ErrGroupHasNoMembers error = errors.New("discourse group has no members")
	// ErrUserHasNoEmail is returned when we can't find user email address
	ErrUserHasNoEmail error = errors.New("can find user email address")
)

// DiscourseClient contains configuration needed to interact with Discourse API
type DiscourseClient struct {
	Endpoint    string
	ApiUsername string
	ApiKey      string
}

// GetGroupMembers return all userID for a specific discourse group
func (d *DiscourseClient) GetGroupMembers(group string) ([]string, error) {
	type groupMember struct {
		Id       int
		Username string
		Name     string
	}

	type meta struct {
		Total  int
		Limit  int
		Offset int
	}

	type jsonResponse struct {
		Members []groupMember
		Errors  []string
		Meta    meta
	}

	queryOffset := 0
	queryLimit := 50
	allQueryDone := false
	members := []string{}

	failureCounter := 0

	// Ensure we either browse every pages returned by Discourse api or exit after 10 errors
	for failureCounter < MaxQueryFailure && !allQueryDone {

		logrus.Debugf("Fetching group %q information", group)

		url := fmt.Sprintf("https://%s/groups/%s/members.json?limit=%d&offset=%d", d.Endpoint, group, queryLimit, queryOffset)

		logrus.Debugln(url)

		client := http.Client{}
		req, err := http.NewRequest("GET", url, nil)

		if err != nil {
			failureCounter++
			continue
		}

		req.Header = http.Header{
			"Api-Key":       []string{d.ApiKey},
			"Api-Username":  []string{d.ApiUsername},
			"Content-Type":  []string{HTTPHeaderContentType},
			"Authorization": []string{HTTPHeaderAuthorization},
		}

		jResp := jsonResponse{}

		resp, err := client.Do(req)

		if err != nil {
			logrus.Debugf("Failure due to:\n%q\n", err.Error())
			failureCounter++
			continue
		}

		// In case of too many request, then sleep for 10 sec
		if resp.StatusCode == 429 {
			logrus.Debugf("%s, waiting 10 seconds", resp.Status)
			time.Sleep(10 * time.Second)
			continue
		} else if resp.StatusCode >= 400 {
			logrus.Debugf("Got response code %d", resp.Status)
			failureCounter++
			continue
		}

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			logrus.Debugf("Failure due to:\n%q\n", err.Error())
			failureCounter++
			continue
		}

		err = json.Unmarshal(body, &jResp)

		if err != nil {
			logrus.Debugf("Failure due to:\n%q\n", err.Error())
			failureCounter++
			continue
		}

		if len(jResp.Errors) > 0 {
			logrus.Errorf("%s", strings.Join(jResp.Errors, "\n"))
			failureCounter++
			continue
		}

		for _, member := range jResp.Members {
			members = append(members, member.Username)
		}

		// Try to identify next offset based on queries already done
		if remainingQueries := (jResp.Meta.Total - (jResp.Meta.Offset + jResp.Meta.Limit)); remainingQueries > 0 {
			queryOffset = jResp.Meta.Total - remainingQueries
			if remainingQueries < queryLimit {
				queryLimit = remainingQueries
			}
			logrus.Debugf("%d(remainingQueries) - %d(total) - %d(queryOffset)\n", remainingQueries, jResp.Meta.Total, queryOffset)
		} else if remainingQueries == 0 {
			logrus.Debugf("All members retrieved %d", len(members))
			allQueryDone = true
		} else {
			logrus.Errorln("something unexpected happened")
			failureCounter++
		}

	}

	if failureCounter >= MaxQueryFailure {
		return nil, fmt.Errorf("Failed %d times to retrieve members from group %q", MaxQueryFailure, group)
	}

	if len(members) == 0 {
		return nil, ErrGroupHasNoMembers
	}

	return members, nil
}

// GetUserEmail retrieve the main email address for a specific user
func (d *DiscourseClient) GetUserEmail(username string) (string, error) {

	type jsonResponse struct {
		Email            string
		Errors           []string
		Secondary_emails []string
	}

	var email string

	logrus.Debugf("Fetching user %q information", username)

	url := fmt.Sprintf("https://%s/u/%s/emails.json", d.Endpoint, username)

	logrus.Debugln(url)

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return "", err
	}

	req.Header = http.Header{
		"Api-Key":       []string{d.ApiKey},
		"Api-Username":  []string{d.ApiUsername},
		"Content-Type":  []string{HTTPHeaderContentType},
		"Authorization": []string{HTTPHeaderAuthorization},
	}

	// Ensure we re-try multiple time a http query in case of error
	failureCounter := 0
	for failureCounter < MaxQueryFailure {
		resp, err := client.Do(req)

		if err != nil {
			failureCounter++
			logrus.Debugln(err)
			continue
		}

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			failureCounter++
			logrus.Debugln(err)
			continue
		}

		jsonResp := jsonResponse{}

		if resp.StatusCode == 429 {
			logrus.Debugf("%s, waiting 10 seconds", resp.Status)
			time.Sleep(10 * time.Second)
			continue
		} else if resp.StatusCode >= 400 {
			failureCounter++
			continue
		}

		err = json.Unmarshal(body, &jsonResp)

		if err != nil {
			failureCounter++
			logrus.Debugln("Something went wrong while unmarshalling json")
			logrus.Debugf("---%s\n---", string(body))
			logrus.Debugln(err)
			continue
		}

		if len(jsonResp.Errors) > 0 {
			failureCounter++
			logrus.Debugln("Something went wrong while parsing json content")
			logrus.Debugln(jsonResp.Errors)
			continue
		}

		email = jsonResp.Email

		if len(email) == 0 {
			failureCounter++
			logrus.Debugln(ErrUserHasNoEmail)
			continue
		}

		if len(email) > 0 {
			return email, nil
		}
	}

	return "", fmt.Errorf("something went wrong")
}
