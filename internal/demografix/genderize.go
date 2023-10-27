package demografix

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type GenderizeResponse struct {
	Gender string `json:"gender"`
	Name   string `json:"name"`
	// Count       int     `json:"count"`
	// Probability float64 `json:"probability"`
}

func (e *Enricher) GetGenderByName(name string) (string, error) {
	query := url.Values{}
	query.Add("name", name)
	u := url.URL{
		Scheme:   "https",
		Host:     "api.genderize.io",
		RawQuery: query.Encode(),
	}
	res, err := http.Get(u.String())
	if err != nil {
		return "", err
	}

	var response GenderizeResponse
	d := json.NewDecoder(res.Body)
	err = d.Decode(&response)
	if err != nil {
		return "", err
	}

	if response.Gender == "" {
		return "", errors.New("coudn't determine gender")
	}

	return response.Gender, nil
}
