package validation

import (
	"encoding/json"
)

type (
	Aggregators []Aggregator
	Aggregator  struct {
		Url     string
		Metrics map[string]struct{}
	}

	// aggregatorsJson is used to encode/decode as json
	aggregatorsJson []aggregatorJson
	aggregatorJson  struct {
		Url     string   `json:"url"`
		Metrics []string `json:"metrics"`
	}
)

func (a *Aggregators) UnmarshalJSON(s []byte) error {
	var ajs aggregatorsJson

	err := json.Unmarshal(s, &ajs)
	if err != nil {
		return err
	}

	// Reset Aggregators
	*a = (*a)[:0]

	for _, aj := range ajs {
		aggregator := Aggregator{
			Url:     aj.Url,
			Metrics: make(map[string]struct{}),
		}
		for _, metric := range aj.Metrics {
			aggregator.Metrics[metric] = struct{}{}
		}
		*a = append(*a, aggregator)
	}

	return nil
}

func (a Aggregators) MarshalJson(s []byte) ([]byte, error) {
	ajs := make(aggregatorsJson, 0, len(a))

	for _, aggregator := range a {
		aj := aggregatorJson{
			Url:     aggregator.Url,
			Metrics: make([]string, 0, len(aggregator.Metrics)),
		}

		for metric := range aggregator.Metrics {
			aj.Metrics = append(aj.Metrics, metric)
		}

		ajs = append(ajs, aj)
	}

	return json.Marshal(ajs)
}
