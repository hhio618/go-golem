package props

import (
	"testing"

	"github.com/hhio618/go-golem/pkg/testutil"
)

type TestCase struct {
	Properties Props
	Err        string
}

func TestProps(t *testing.T) {
	testCases := []TestCase{
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []float32{0.001, 0.002, 0.0},
				"golem.com.usage.vector":                []string{"golem.usage.cpu_sec", "golem.usage.duration_sec"},
				"golem.com.scheme":                      "payu",
			},
			Err: "",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []float32{0.001, 0.002, 0.0},
				"golem.com.usage.vector":                []string{"golem.usage.cpu_sec", "golem.usage.duration_sec"},
				"golem.com.scheme":                      "payu",
				"golem.superfluous.key":                 "Some other stuff",
			},
			Err: "",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []float32{0.001, 0.002, 0.0},
				"golem.com.usage.vector":                []string{"golem.usage.cpu_sec", "golem.usage.duration_sec"},
				// "golem.com.scheme": "payu",
			},
			Err: "",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model": "linear",
				// "golem.com.pricing.model.linear.coeffs": []string{0.001, 0.002, 0.0},
				"golem.com.usage.vector": []string{"golem.usage.cpu_sec", "golem.usage.duration_sec"},
				"golem.com.scheme":       "payu",
			},
			Err: "missing key: 'golem.com.pricing.model.linear.coeffs'",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []string{},
				"golem.com.usage.vector":                []string{"golem.usage.cpu_sec", "golem.usage.duration_sec"},
				"golem.com.scheme":                      "payu",
			},
			Err: "pop from empty list",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []float32{0.001, 0.002, 0.0},
				"golem.com.usage.vector":                []string{"golem.usage.cpu_sec"},
				"golem.com.scheme":                      "payu",
			},
			Err: "list index out of range",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []interface{}{"spam", 0.002, 0.0},
				"golem.com.usage.vector":                []string{"golem.usage.cpu_sec", "golem.usage.duration_sec"},
				"golem.com.scheme":                      "payu",
			},
			Err: "could not convert string to float",
		},
		{
			Properties: map[string]interface{}{
				"golem.com.pricing.model":               "linear",
				"golem.com.pricing.model.linear.coeffs": []interface{}{"spam", 0.002, 0.0},
				"golem.com.usage.vector":                "not a vector",
				"golem.com.scheme":                      "payu",
			},
			Err: "error when decoding 'not a vector'",
		},
	}
	for _, testCase := range testCases {
		model := &ComLinear{}
		err := FromProperties(testCase.Properties, model)
		if testCase.Err == "" {
			testutil.Ok(t, err)
			t.Logf("TestCase data: %v", model)
		} else {
			t.Logf("TestCase error: %v", testCase.Err)
			testutil.NotOk(t, err)
			testutil.Equals(t, testCase.Err, err.Error())
		}
	}
}
