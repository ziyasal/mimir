package validation

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestEncodingDecodingAggregators(t *testing.T) {
	testCases := map[string]Aggregators{
		"single aggregator, single metric": {
			{
				Url: "http://a.b.c",
				Metrics: map[string]struct{}{
					"metric1": {},
				},
			},
		},
		"many aggregators, many metrics": {
			{
				Url: "http://endpoint1",
				Metrics: map[string]struct{}{
					"metric1": {},
					"metric2": {},
					"metric3": {},
					"metric4": {},
					"metric5": {},
					"metric6": {},
					"metric7": {},
					"metric8": {},
					"metric9": {},
				},
			}, {
				Url: "http://endpoint2",
				Metrics: map[string]struct{}{
					"metric1": {},
					"metric2": {},
					"metric3": {},
					"metric4": {},
					"metric5": {},
					"metric6": {},
					"metric7": {},
					"metric8": {},
					"metric9": {},
				},
			}, {
				Url: "http://endpoint3",
				Metrics: map[string]struct{}{
					"metric1": {},
					"metric2": {},
					"metric3": {},
					"metric4": {},
					"metric5": {},
					"metric6": {},
					"metric7": {},
					"metric8": {},
					"metric9": {},
				},
			},
		},
		"empty": {},
	}

	for name, testCase := range testCases {
		t.Run(name+" (JSON)", func(t *testing.T) {
			encoded, err := json.Marshal(testCase)
			require.NoError(t, err)

			var decoded Aggregators
			err = json.Unmarshal(encoded, &decoded)
			require.NoError(t, err)
		})

		t.Run(name+" (YAML)", func(t *testing.T) {
			encoded, err := yaml.Marshal(testCase)
			require.NoError(t, err)

			fmt.Println(encoded)
			var decoded Aggregators
			err = yaml.Unmarshal(encoded, &decoded)
			require.NoError(t, err)
		})
	}
}
