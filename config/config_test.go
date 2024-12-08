package config_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/TheRebelOfBabylon/tandem/config"
)

type validateTestCase struct {
	name           string
	config         *config.Config
	expectedErr    error
	expectedConfig *config.Config
}

var validateTestCases = []validateTestCase{
	{
		name: "ErrorCase_InvalidLogLevel",
		config: &config.Config{
			Log: config.Log{
				Level: "foo",
			},
		},
		expectedErr: config.ErrInvalidLogLevel,
	},
	{
		name: "ValidCase_DefaultLogLevel",
		config: &config.Config{
			HTTP: config.HTTP{
				Host: "localhost",
				Port: 5000,
			},
		},
		expectedErr: nil,
		expectedConfig: &config.Config{
			Log: config.Log{
				Level: "info",
			},
			HTTP: config.HTTP{
				Host: "localhost",
				Port: 5000,
			},
		},
	},
	{
		name: "ValidCase_DefaultHTTPHost",
		config: &config.Config{
			Log: config.Log{
				Level: "info",
			},
			HTTP: config.HTTP{
				Port: 5000,
			},
		},
		expectedErr: nil,
		expectedConfig: &config.Config{
			Log: config.Log{
				Level: "info",
			},
			HTTP: config.HTTP{
				Host: "localhost",
				Port: 5000,
			},
		},
	},
	{
		name: "ValidCase_DefaultHTTPPort",
		config: &config.Config{
			Log: config.Log{
				Level: "info",
			},
			HTTP: config.HTTP{
				Host: "localhost",
			},
		},
		expectedErr: nil,
		expectedConfig: &config.Config{
			Log: config.Log{
				Level: "info",
			},
			HTTP: config.HTTP{
				Host: "localhost",
				Port: 5000,
			},
		},
	},
}

// TestValidate ensures the Validate method of the config struct behaves as expected
func TestValidate(t *testing.T) {
	for _, testCase := range validateTestCases {
		t.Logf("starting test case %s...", testCase.name)
		if err := testCase.config.Validate(); !errors.Is(err, testCase.expectedErr) {
			t.Errorf("unexpected error for test case %s: expected %v got %v", testCase.name, testCase.expectedErr, err)
		}
		// check if the config matches the expected config
		if testCase.expectedConfig != nil && !reflect.DeepEqual(testCase.config, testCase.expectedConfig) {
			t.Errorf("unexpected config for test case %s: expected %v got %v", testCase.name, testCase.config, testCase.expectedConfig)
		}
	}
	t.Log("all tests completed")
}
