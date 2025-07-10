# `kode-ai`
The prime AI tool, including low level completion primitive as well as high level agentic coding.

# Install

`go install`:
```sh
go install github.com/xhd2015/kode-ai/cmd/kode@latest
```

`curl` from github:
```sh
curl -fsSL https://github.com/xhd2015/kode-ai/raw/master/install.sh | bash
```

# Usage

```sh
# chat with gpt-4.1
export OPENAI_API_KEY=...
kode chat 'hello'

# chat with cluade, stop after initial response(1 round)
export ANTHROPIC_API_KEY=...
kode chat --model=claude-sonnet-4 --record=chat.json --system=EXAMPLE_SYSTEM.md --tool batch_read_file "What's in the file?"

# chat with --max-round
kode chat --max-round=10 ...
```