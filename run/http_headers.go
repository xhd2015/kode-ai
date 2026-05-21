package run

import (
	"fmt"
	"strings"

	"github.com/xhd2015/kode-ai/types"
)

func parseHTTPHeaders(rawHeaders []string) (types.HTTPHeaders, error) {
	if len(rawHeaders) == 0 {
		return nil, nil
	}
	headers := make(types.HTTPHeaders)
	for _, raw := range rawHeaders {
		name, value, ok := strings.Cut(raw, ":")
		if !ok {
			return nil, fmt.Errorf("invalid --header %q: expected curl-style \"Name: value\"", raw)
		}
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("invalid --header %q: header name is empty", raw)
		}
		headers[name] = append(headers[name], strings.TrimLeft(value, " \t"))
	}
	return headers, nil
}

func mergeHTTPHeaders(base types.HTTPHeaders, overrides types.HTTPHeaders) types.HTTPHeaders {
	if len(base) == 0 && len(overrides) == 0 {
		return nil
	}
	merged := make(types.HTTPHeaders, len(base)+len(overrides))
	for name, values := range base {
		merged[name] = append(merged[name], values...)
	}
	for name, values := range overrides {
		merged[name] = append(merged[name], values...)
	}
	return merged
}
