package validation

import (
	"encoding/json"

	"gopkg.in/yaml.v2"
)

type (
	Aggregators []Aggregator
	Aggregator  struct {
		Url     string
		Metrics map[string]struct{}
	}

	// aggregatorsEncoded is used to encode/decode as json
	aggregatorsEncoded []aggregatorEncoded
	aggregatorEncoded  struct {
		Url     string   `yaml:"url" json:"url"`
		Metrics []string `yaml:"metrics" json:"metrics"`
	}
)

func (a *Aggregators) UnmarshalJSON(s []byte) error {
	var aggsEnc aggregatorsEncoded

	err := json.Unmarshal(s, &aggsEnc)
	if err != nil {
		return err
	}

	a.applyEncoded(aggsEnc)

	return nil
}

func (a *Aggregators) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var aggsEnc aggregatorsEncoded

	err := unmarshal(&aggsEnc)
	if err != nil {
		return err
	}

	a.applyEncoded(aggsEnc)

	return nil
}

func (a *Aggregators) applyEncoded(aggsEnc aggregatorsEncoded) {
	// Reset Aggregators
	*a = (*a)[:0]

	for _, aggEnc := range aggsEnc {
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
	return yaml.Marshal(a.getEncoded())
}

func (a Aggregators) getEncoded() aggregatorsEncoded {
	ajs := make(aggregatorsEncoded, 0, len(a))

	for _, aggregator := range a {
		aj := aggregatorEncoded{
			Url:     aggregator.Url,
			Metrics: make([]string, 0, len(aggregator.Metrics)),
		}

		for metric := range aggregator.Metrics {
			aj.Metrics = append(aj.Metrics, metric)
		}

		ajs = append(ajs, aj)
	}

	return ajs
}
