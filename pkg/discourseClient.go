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
func (d *DiscourseClient) GetGroupMembers(group string) (members []string, err error) {
	type groupMember struct {
		Id       int
		Username string
		Name     string
	}

	type jsonResponse struct {
		Members []groupMember
		Errors  []string
	}

	logrus.Debugf("Fetching group %q information", group)

	url := fmt.Sprintf("https://%s/groups/%s/members.json", d.Endpoint, group)

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return members, err
	}

	req.Header = http.Header{
		"Api-Key":       []string{d.ApiKey},
		"Api-Username":  []string{d.ApiUsername},
		"Content-Type":  []string{HTTPHeaderContentType},
		"Authorization": []string{HTTPHeaderAuthorization},
	}

	for i := 0; i < MaxQueryFailure; i++ {
		lastTry := (i == MaxQueryFailure)

		jResp := jsonResponse{}

		resp, err := client.Do(req)

		if err != nil {
			return members, err
		}

		if resp.StatusCode == 429 {
			if lastTry {
				return members, fmt.Errorf("%s", resp.Status)
			}
			logrus.Debugln(resp.Status)
			continue
		} else if resp.StatusCode >= 400 {
			return members, fmt.Errorf("%s", resp.Status)
		}

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			if lastTry {
				return members, err
			}
			logrus.Debugln(string(body))
			continue
		}

		err = json.Unmarshal(body, &jResp)

		if err != nil {
			if lastTry {
				return members, err
			}
			continue
		}

		if len(jResp.Errors) > 0 {
			if lastTry {
				return members, fmt.Errorf("%s", strings.Join(jResp.Errors, "\n"))
			}
			continue
		}

		if len(jResp.Members) > 0 {

			for _, member := range jResp.Members {
				members = append(members, member.Username)
			}
			break
		}

		if lastTry {
			return members, ErrGroupHasNoMembers
		}
	}

	return members, nil
}

// GetUserEmail retrieve the main email address for a specific user
func (d *DiscourseClient) GetUserEmail(username string) (email string, err error) {

	type jsonResponse struct {
		Email            string
		Errors           []string
		Secondary_emails []string
	}

	logrus.Debugf("Fetching user %q information", username)

	url := fmt.Sprintf("https://%s/u/%s/emails.json", d.Endpoint, username)

	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return email, err
	}

	req.Header = http.Header{
		"Api-Key":       []string{d.ApiKey},
		"Api-Username":  []string{d.ApiUsername},
		"Content-Type":  []string{HTTPHeaderContentType},
		"Authorization": []string{HTTPHeaderAuthorization},
	}

	for i := 0; i < MaxQueryFailure; i++ {
		lastTry := (i == MaxQueryFailure)
		resp, err := client.Do(req)

		if err != nil {
			if lastTry {
				return email, err
			}
			logrus.Debugln(err)
			continue
		}

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			if lastTry {
				return email, err
			}
			logrus.Debugln(err)
			continue
		}

		jsonResp := jsonResponse{}

		if resp.StatusCode == 429 {
			if lastTry {
				return email, fmt.Errorf("%s", resp.Status)
			}
			logrus.Debugf("%s, waiting 10 seconds", resp.Status)
			time.Sleep(10 * time.Second)
			continue
		} else if resp.StatusCode >= 400 {
			return email, fmt.Errorf("%s", resp.Status)
		}

		err = json.Unmarshal(body, &jsonResp)

		if err != nil {
			if lastTry {
				return email, err
			}
			logrus.Debugln("Something went wrong while unmarshalling json")
			logrus.Debugf("---%s\n---", string(body))
			logrus.Debugln(err)
			continue
		}

		if len(jsonResp.Errors) > 0 {
			if lastTry {
				return "", fmt.Errorf("%s", strings.Join(jsonResp.Errors, "\n"))
			}
			logrus.Debugln("Something went wrong while parsing json content")
			logrus.Debugln(jsonResp.Errors)
			continue
		}

		email = jsonResp.Email

		if len(email) == 0 {
			logrus.Debugln(ErrUserHasNoEmail)
			continue
		}

		if len(email) > 0 {
			return email, nil
		}
	}

	return "", fmt.Errorf("something went wrong")
}
