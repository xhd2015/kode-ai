module github.com/xhd2015/kode-ai/chat/integration

go 1.24

require (
	github.com/xhd2015/kode-ai v0.0.0-00010101000000-000000000000
	github.com/xhd2015/kode-ai/cli v0.0.0-00010101000000-000000000000
	github.com/xhd2015/kode-ai/types v0.0.2
	github.com/xhd2015/llm-tools v0.0.17
)

require (
	cloud.google.com/go v0.116.0 // indirect
	cloud.google.com/go/auth v0.9.3 // indirect
	cloud.google.com/go/compute/metadata v0.5.0 // indirect
	github.com/anthropics/anthropic-sdk-go v1.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/s2a-go v0.1.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.4 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/mark3labs/mcp-go v0.33.0 // indirect
	github.com/openai/openai-go v1.8.3 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/xhd2015/less-gen v0.0.18 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/term v0.32.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	google.golang.org/genai v1.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240903143218-8af14fe29dc1 // indirect
	google.golang.org/grpc v1.66.2 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace github.com/xhd2015/kode-ai/cli => ../../cli

replace github.com/xhd2015/kode-ai => ../..

replace github.com/xhd2015/kode-ai/types => ../../types
