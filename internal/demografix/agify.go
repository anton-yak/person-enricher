package demografix

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type AgifyResponse struct {
	Age  uint   `json:"age"`
	Name string `json:"name"`
	// Count int    `json:"count"`
}

func (e *Enricher) GetAgeByName(name string) (uint, error) {
	query := url.Values{}
	query.Add("name", name)
	u := url.URL{
		Scheme:   "https",
		Host:     "api.agify.io",
		RawQuery: query.Encode(),
	}
	res, err := http.Get(u.String())
	if err != nil {
		return 0, err
	}

	var response AgifyResponse
	d := json.NewDecoder(res.Body)
	err = d.Decode(&response)
	if err != nil {
		return 0, err
	}

	if response.Age == 0 {
		return 0, errors.New("coudn't determine age")
	}

	return response.Age, nil
}
