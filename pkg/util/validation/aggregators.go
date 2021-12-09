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

	// AggregatorsEncoded is used to encode/decode as json
	AggregatorsEncoded struct {
		Aggregators []AggregatorEncoded `yaml:"aggregators" json:"aggregators"`
	}

	AggregatorEncoded struct {
		Url     string   `yaml:"url" json:"url"`
		Metrics []string `yaml:"metrics" json:"metrics"`
	}
)

func (a *Aggregators) UnmarshalJSON(s []byte) error {
	var aggsEnc AggregatorsEncoded

	err := json.Unmarshal(s, &aggsEnc)
	if err != nil {
		return err
	}

	a.applyEncoded(aggsEnc)

	return nil
}

func (a *Aggregators) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var aggsEnc AggregatorsEncoded

	err := unmarshal(&aggsEnc)
	if err != nil {
		return err
	}

	a.applyEncoded(aggsEnc)

	return nil
}

func (a *Aggregators) applyEncoded(aggsEnc AggregatorsEncoded) {
	// Reset Aggregators
	*a = (*a)[:0]

	for _, aggEnc := range aggsEnc.Aggregators {
		aggregator := Aggregator{
			Url:     aggEnc.Url,
			Metrics: make(map[string]struct{}),
		}
		for _, metric := range aggEnc.Metrics {
			aggregator.Metrics[metric] = struct{}{}
		}
		*a = append(*a, aggregator)
	}
}

func (a Aggregators) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.getEncoded())
}

func (a Aggregators) MarshalYAML() (interface{}, error) {
	encoded := a.getEncoded()
	return &encoded, nil
}

func (a Aggregators) getEncoded() AggregatorsEncoded {
	aggsEnc := AggregatorsEncoded{
		Aggregators: make([]AggregatorEncoded, 0, len(a)),
	}

	for _, aggregator := range a {
		aggEnc := AggregatorEncoded{
			Url:     aggregator.Url,
			Metrics: make([]string, 0, len(aggregator.Metrics)),
		}

		for metric := range aggregator.Metrics {
			aggEnc.Metrics = append(aggEnc.Metrics, metric)
		}

		aggsEnc.Aggregators = append(aggsEnc.Aggregators, aggEnc)
	}

	return aggsEnc
}
