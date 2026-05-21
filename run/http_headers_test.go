package run

import (
	"reflect"
	"testing"

	"github.com/xhd2015/kode-ai/types"
)

func TestParseHTTPHeaders(t *testing.T) {
	headers, err := parseHTTPHeaders([]string{
		"x-opencode-client: kode",
		"x-opencode-session: ses_test",
		"x-empty:",
	})
	if err != nil {
		t.Fatalf("parseHTTPHeaders returned error: %v", err)
	}

	want := types.HTTPHeaders{
		"x-opencode-client":  {"kode"},
		"x-opencode-session": {"ses_test"},
		"x-empty":            {""},
	}
	if !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers mismatch\nwant: %#v\n got: %#v", want, headers)
	}
}

func TestParseHTTPHeadersRejectsInvalid(t *testing.T) {
	for _, raw := range []string{"x-opencode-client", ": value"} {
		if _, err := parseHTTPHeaders([]string{raw}); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

func TestMergeHTTPHeaders(t *testing.T) {
	headers := mergeHTTPHeaders(
		types.HTTPHeaders{"x-one": {"config"}, "x-two": {"config"}},
		types.HTTPHeaders{"x-one": {"cli"}},
	)
	want := types.HTTPHeaders{"x-one": {"config", "cli"}, "x-two": {"config"}}
	if !reflect.DeepEqual(headers, want) {
		t.Fatalf("headers mismatch\nwant: %#v\n got: %#v", want, headers)
	}
}
