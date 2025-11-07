module github.com/xhd2015/kode-ai/cli

// NOTE: cli is go1.18, so it should not depend on  github.com/xhd2015/kode-ai(go1.24)
go 1.18

require github.com/xhd2015/kode-ai/types v0.0.6

require github.com/xhd2015/llm-tools v0.0.19

require github.com/gorilla/websocket v1.5.3

require github.com/shopspring/decimal v1.4.0 // indirect
