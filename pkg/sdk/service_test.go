package sdk

import "testing"

func TestSkipCache(t *testing.T) {
	opt := SkipCache()
	var cfg stateListConfig
	opt(&cfg)
	if !cfg.skipCache {
		t.Error("SkipCache() should set skipCache to true")
	}
}

func TestApplyStateListOptions_NoOpts(t *testing.T) {
	cfg := ApplyStateListOptions(nil)
	if cfg.ShouldSkipCache() {
		t.Error("empty opts should not skip cache")
	}
}

func TestApplyStateListOptions_WithSkipCache(t *testing.T) {
	cfg := ApplyStateListOptions([]StateListOption{SkipCache()})
	if !cfg.ShouldSkipCache() {
		t.Error("SkipCache option should make ShouldSkipCache() true")
	}
}
