package config

import "testing"

func TestDefaultConfigIsValid(t *testing.T) {
	configuration := Default()
	if err := configuration.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(configuration.EnabledCounterCodes()) != 3 {
		t.Fatalf("unexpected counters: %+v", configuration.Counters)
	}
}

func TestConfigRejectsDuplicateCounter(t *testing.T) {
	configuration := Default()
	configuration.Counters[1].Code = configuration.Counters[0].Code
	if err := configuration.Validate(); err == nil {
		t.Fatal("expected validation error")
	}
}
