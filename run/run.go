package run

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xhd2015/kode-ai/internal/ioread"
	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/less-gen/flags"
)

//go:embed VERSION.txt
var version string

//go:embed REVISION.txt
var revision string

// TODO: add --max-cost-usd 2
const help = `
kode interact with LLMs

Usage: kode <cmd> [OPTIONS]

Available commands:
  chat <msg>                      chat with llm, msg can contain @file(path/to/file) directive
  view <files...>                 view recorded chat files
  example                         show examples
  version                         version info
  revision                        revision info
  help                            show help message

Options:
  --auto                          enable auto chat mode
  --max-round N                   maximum number of rounds in agent mode, 0 means no limit(default: 10)
  --token TOKEN                   the token
  --base-url BASE_URL             the base url
  --model MODEL                   llm model(default: gpt-4.1)
  --system PROMPT                 set the system prompt, PROMPT can also be a file
  --tool NAME                     predefined tool: batch_read_file,list_dir,grep_search...
                                  use kode --tool list to see all possible tools
  --tool-custom FILE              tool provided to LLM
  --tool-custom-json JSON         tool provided to LLM, in json, see tool example
  --tool-default-cwd DIR          the default working directory for tools, default current dir
                                  use --tool-default-cwd='' to unset it
  --record FILE                   record chat history to given json file, which can be used to store and resume the chat
  --no-cache                      disable token caching
  --show-usage                    show usage from the file specified by --record
  --ignore-duplicate-msg          ignore duplicate user msg
  --log-request                   log http request
  --log-chat                      log chat(default: true)
  -v,--verbose                    show verbose info

Examples:
  kode chat 'hello'               chat with llm

  # provide tools to LLM
  kode chat --max-round=10 --tool list_dir "What's in current dir? wd is: $PWD"

  # agent-like chat
  kode chat --record tmp/chat.json --model=claude-3-7-sonnet --system=tmp/TRACE_PROMPT_CURSOR_LIKE.md --tool-preset=batch_read_file --tool-preset=list_dir --tool-preset=run_terminal_cmd -v --ignore-duplicate-msg "<user_query>gather some critical information for it</user_query>"

  # show token usage
  kode chat --record tmp/chat.json --model=claude-3-7-sonnet --show-usage

Tool example:
  open-ai: '{"name":"record_names","description":"record the names needed to analyse","parameters":{"type":"object","properties":{"names":{"type":"array","description":"the names needed to analyse","items":{"type":"string"}}},"required":["names"]}}'

Available models:
  open-ai: gpt-4.1, gpt-4.1-mini, gpt-4o, gpt-4o-mini, gpt-4o-nano, o4-mini, o3
  anthropic: claude-3-7-sonnet

For system prompt, see https://docs.anthropic.com/en/docs/build-with-claude/prompt-engineering/system-prompts and https://platform.openai.com/docs/guides/text?api-mode=chat

To smoothly chat with llm, you can use VSCode to edit your cli:
  export EDITOR='code --wait'
  kode chat ... (click CTRL-X CTRL-E to enter edit, then save and close the editor)
`

type Options struct {
	BaseCmd        string
	DefaultBaseURL string
}

func getHelp(baseCmd string) string {
	if baseCmd == "" {
		return help
	}
	return strings.ReplaceAll(help, "kode", baseCmd)
}

func Main(args []string, opts Options) error {
	if len(args) == 0 {
		return fmt.Errorf("requires sub command: chat, view, examples. try `kode --help`")
	}
	cmd := args[0]
	args = args[1:]
	if cmd == "help" || cmd == "--help" {
		fmt.Print(strings.TrimPrefix(getHelp(opts.BaseCmd), "\n"))
		return nil
	}
	switch cmd {
	case "chat":
		return handleChat(cmd, args, opts.BaseCmd, opts.DefaultBaseURL)
	case "view":
		return handleView(args)
	case "example", "examples":
		return handleExample(args)
	case "version":
		fmt.Println(version)
		return nil
	case "revision":
		fmt.Println(revision)
		return nil
	default:
		return fmt.Errorf("unrecognized: %s, use 'kode help' to see available commands", cmd)
	}
}

func handleChat(mode string, args []string, baesCmd string, defaultBaseURL string) error {
	if len(args) > 0 {
		arg := args[0]
		switch arg {
		case "exmaple", "examples":
			// output example
			return handleExample(args[1:])
		case "view":
			return handleView(args[1:])
		}
	}

	if mode != "chat" {
		return fmt.Errorf("unrecognized mode: %s, use 'kode help' to see available commands", mode)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	var token string
	var baseUrl string = defaultBaseURL
	var systemPrompt string
	var model string

	var recordFile string

	var tools []string
	var toolCustomFiles []string
	var toolCustomJSONs []string

	var showUsage bool
	var ignoreDuplicateMsg bool

	var toolDefaultCwd string = cwd
	var maxRound int
	var noCache bool

	var logRequest bool
	var logChat bool = true
	var verbose bool

	flagsParser := flags.String("--token", &token).
		Int("--max-round", &maxRound).
		String("--base-url", &baseUrl).
		String("--system", &systemPrompt).
		StringSlice("--tool", &tools).
		StringSlice("--tool-custom", &toolCustomFiles).
		StringSlice("--tool-custom-json", &toolCustomJSONs).
		String("--tool-default-cwd", &toolDefaultCwd).
		String("--model", &model).
		String("--record", &recordFile).
		Bool("--no-cache", &noCache).
		Bool("--show-usage", &showUsage).
		Bool("--ignore-duplicate-msg", &ignoreDuplicateMsg).
		Bool("--log-request", &logRequest).
		Bool("--log-chat", &logChat).
		Bool("-v,--verbose", &verbose).
		Help("-h,--help", getHelp(baesCmd))

	args, err = flagsParser.Parse(args)
	if err != nil {
		return err
	}
	if showUsage {
		if recordFile == "" {
			return fmt.Errorf("requires --record")
		}
		return showUsageFromRecordFile(recordFile)
	}

	if model == "" {
		model = providers.ModelGPT4_1
	}

	var msg string
	if len(args) > 0 {
		msg = args[0]
		args = args[1:]

		msg, err = ioread.ReadOrContent(msg)
		if err != nil {
			return err
		}
	}

	if len(args) > 0 {
		return fmt.Errorf("unrecognized extra: %s", strings.Join(args, ","))
	}

	if maxRound != 0 {
		if maxRound < 0 {
			return fmt.Errorf("invalid --max-round: %d, must be positive", maxRound)
		}
	}

	opts := ChatOptions{
		maxRound: maxRound,

		systemPrompt:   systemPrompt,
		logRequest:     logRequest,
		toolPresets:    tools,
		toolFiles:      toolCustomFiles,
		toolJSONs:      toolCustomJSONs,
		recordFile:     recordFile,
		toolDefaultCwd: toolDefaultCwd,

		noCache: noCache,

		ignoreDuplicateMsg: ignoreDuplicateMsg,
		logChat:            logChat,
		verbose:            verbose,
	}

	if model == providers.ModelClaude3_7Sonnet {
		model = providers.ModelClaude3_7Sonnet_20250219
	}
	if model == providers.ModelClaudeSonnet4 {
		model = providers.ModelClaudeSonnet4_20250514
	}
	provider, err := providers.GetModelProvider(model)
	if err != nil {
		return err
	}

	var tokenEnvKey string
	var baseUrlEnvKey string
	switch provider {
	case providers.ProviderOpenAI:
		tokenEnvKey = "OPENAI_API_KEY"
		baseUrlEnvKey = "OPENAI_BASE_URL"
	case providers.ProviderAnthropic:
		tokenEnvKey = "ANTHROPIC_API_KEY"
		baseUrlEnvKey = "ANTHROPIC_BASE_URL"
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	c := ChatHandler{
		Provider:             provider,
		TokenEnvKey:          tokenEnvKey,
		BaseUrlEnvKey:        baseUrlEnvKey,
		DefaultBaseUrlEnvKey: "KODE_DEFAULT_BASE_URL",
	}
	return c.Handle(model, baseUrl, token, msg, opts)
}

// just like replay the whole messages
func handleView(args []string) error {
	var verbose bool
	var lastAssistant bool
	var showUsage bool
	var tools bool
	args, err := flags.Bool("-v,--verbose", &verbose).
		Bool("--last-assistant", &lastAssistant).
		Bool("--show-usage", &showUsage).
		Bool("--tools", &tools).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("requires files")
	}

	if showUsage && lastAssistant {
		return fmt.Errorf("--show-usage and --last-assistant cannot be specified at the same time")
	}

	files := args
	if showUsage {
		var allMessages Messages
		for _, file := range files {
			msg, err := loadHistoricalMessages(file)
			if err != nil {
				return err
			}
			allMessages = append(allMessages, msg...)
		}
		return showUsageFromMessages(allMessages)
	}

	if lastAssistant {
		n := len(files)
		for i := n - 1; i >= 0; i-- {
			msg, err := loadHistoricalMessages(files[i])
			if err != nil {
				return err
			}
			m := len(msg)
			for j := m - 1; j >= 0; j-- {
				if msg[j].Type == MsgType_Msg && msg[j].Role == "assistant" {
					fmt.Println(msg[j].Content)
					return nil
				}
			}
		}
		return fmt.Errorf("no assistant message found")
	}

	var total TokenUsageCost
	for _, file := range files {
		msg, err := loadHistoricalMessages(file)
		if err != nil {
			return err
		}

		for _, m := range msg {
			if tools {
				switch m.Type {
				case MsgType_ToolCall, MsgType_ToolResult:
				default:
					continue
				}
			}

			switch m.Type {
			case MsgType_Msg:
				fmt.Printf("%s: %s\n", m.Role, m.Content)
			case MsgType_ToolCall:
				limitedContent := limitPrintLength(m.Content)
				fmt.Printf("%s: <tool_call tool=%q>%s</tool_call>\n", m.Role, m.ToolName, limitedContent)
			case MsgType_ToolResult:
				limitedContent := limitPrintLength(m.Content)
				fmt.Printf("%s: <tool_result tool=%q>%s</tool_result>\n", m.Role, m.ToolName, limitedContent)
			case MsgType_TokenUsage:
				provider, err := providers.GetModelProvider(m.Model)
				if err != nil {
					fmt.Printf("%s: token cost: %v\n", m.Role, err)
					continue
				}
				var tokenUsage TokenUsage
				if m.TokenUsage != nil {
					tokenUsage = *m.TokenUsage
				}

				total.Usage = total.Usage.Add(tokenUsage)

				cost, costOK := computeCost(provider, m.Model, tokenUsage)
				var costUSD string
				if costOK {
					costUSD = "$" + cost.TotalUSD
					total.Cost = total.Cost.Add(cost)
				}
				printTokenUsage(os.Stdout, "Token Usage", tokenUsage, costUSD)
			case MsgType_StopReason:
				// nothing
			default:
				limitedContent := limitPrintLength(m.Content)
				fmt.Printf("%s: (unrecognized msg type: %s)%s\n", m.Role, m.Type, limitedContent)
			}
		}
	}

	var totalCostUSD string
	if total.Cost.TotalUSD != "" {
		totalCostUSD = "$" + total.Cost.TotalUSD
	}
	printTokenUsage(os.Stdout, "Total Usage", total.Usage, totalCostUSD)
	return nil
}

func mustMarshal(v interface{}) string {
	jsonData, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(jsonData)
}

// loadHistoricalMessages loads historical chat messages from the record file in unified format
func loadHistoricalMessages(recordFile string) (Messages, error) {
	var messages []Message

	file, err := os.Open(recordFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return empty messages
			return messages, nil
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var chatMsg Message
		if err := decoder.Decode(&chatMsg); err != nil {
			if err == io.EOF {
				// End of file, break the loop
				break
			}
			return nil, fmt.Errorf("failed to parse chat message: %v", err)
		}

		// Convert to unified message format (now just the same as ChatMessage)
		messages = append(messages, chatMsg)
	}

	return messages, nil
}

// appendToRecordFile appends a chat message to the record file
func appendToRecordFile(recordFile string, msg *Message) error {
	if msg.Time == "" {
		cloneMsg := *msg
		cloneMsg.Time = time.Now().Format("2006-01-02 15:04:05-07:00")
		msg = &cloneMsg
	}
	file, err := os.OpenFile(recordFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = file.WriteString(string(jsonData) + "\n")
	return err
}

func handleExample(args []string) error {
	const examples = `
# fetch trace first
spl trace 5096cd609a7ddb8841b810d0dfa37b65 --dump tmp

# ask questions about the trace
kode chat --record tmp/chat-why-empty-instalments.json --model=claude-3-7-sonnet --system=tmp/TRACE_PROMPT.md --tool-preset=batch_read_file --tool-preset=list_dir --tool-preset=run_terminal_cmd -v --ignore-duplicate-msg "<user_query>为什么返回的instalments列表是空的?</user_query> working directory: $PWD/some_trace"

# Test cache (NOTE)
kode chat --model=claude-3-7-sonnet --record tmp/cache-test/record.json -v --system='You are a Lucas
, a 18-year-old cowboy' 'Who are you?'
kode chat --model=claude-3-7-sonnet --record tmp/cache-test/record.json -v --system='You are a Lucas
, a 18-year-old cowboy' 'How old are you?'

# Trace compact
kode chat --model=claude-sonnet-4 --system tmp/TRACE_PROMPT_STRUCTURE.md "<user_query>please compact slightly the tr
ee output to save tokens without losing significant information</user_query> current working directory: $PWD/trace_working_dir/some_dir tree output: $(llm-tools tree --collapse --dir-only trace_working_dir/some_dir)"
`

	fmt.Print(strings.TrimPrefix(examples, "\n"))
	return nil
}
