package demografix

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
)

type ResponseCountry struct {
	CountryID string `json:"country_id"`
	// Probability float64 `json:"probability"`
}

type NationalizeResponse struct {
	Name string `json:"name"`
	// Count int    `json:"count"`
	Country []ResponseCountry `json:"country"`
}

func (e *Enricher) GetNationalityByName(name string) (string, error) {
	query := url.Values{}
	query.Add("name", name)
	u := url.URL{
		Scheme:   "https",
		Host:     "api.nationalize.io",
		RawQuery: query.Encode(),
	}
	res, err := http.Get(u.String())
	if err != nil {
		return "", err
	}

	var response NationalizeResponse
	d := json.NewDecoder(res.Body)
	err = d.Decode(&response)
	if err != nil {
		return "", err
	}

	if len(response.Country) == 0 {
		return "", errors.New("couldn't determine nationality")
	}

	return response.Country[0].CountryID, nil
}
