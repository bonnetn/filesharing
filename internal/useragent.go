package internal

import (
	"encoding/json"
	"net/http"
)

type UserAgentMap map[string]string

func FetchUserAgentDatabase() (UserAgentMap, error) {
	resp, err := http.Get("https://raw.githubusercontent.com/monperrus/crawler-user-agents/master/crawler-user-agents.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userAgentList []struct {
		Pattern   string   `json:"pattern"`
		Instances []string `json:"instances"`
	}
	err = json.NewDecoder(resp.Body).Decode(&userAgentList)
	result := make(UserAgentMap, len(userAgentList))
	for _, crawler := range userAgentList {
		for _, userAgent := range crawler.Instances {
			result[userAgent] = crawler.Pattern
		}
	}
	return result, err
}
