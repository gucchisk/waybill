package main

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	testCases := []struct {
		name           string
		ldflagsVersion string
		moduleVersion  string
		buildInfoFound bool
		want           string
	}{
		{name: "ldflags version wins over module version", ldflagsVersion: "1.0.0", moduleVersion: "v2.0.0", buildInfoFound: true, want: "1.0.0"},
		{name: "module version from go install", ldflagsVersion: "dev", moduleVersion: "v1.2.3", buildInfoFound: true, want: "v1.2.3"},
		{name: "devel module version falls back to dev", ldflagsVersion: "dev", moduleVersion: "(devel)", buildInfoFound: true, want: "dev"},
		{name: "empty module version falls back to dev", ldflagsVersion: "dev", moduleVersion: "", buildInfoFound: true, want: "dev"},
		{name: "missing build info falls back to dev", ldflagsVersion: "dev", buildInfoFound: false, want: "dev"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			readBuildInfo := func() (*debug.BuildInfo, bool) {
				if !testCase.buildInfoFound {
					return nil, false
				}
				return &debug.BuildInfo{Main: debug.Module{Version: testCase.moduleVersion}}, true
			}
			if got := resolveVersion(testCase.ldflagsVersion, readBuildInfo); got != testCase.want {
				t.Errorf("resolveVersion() = %q, want %q", got, testCase.want)
			}
		})
	}
}
