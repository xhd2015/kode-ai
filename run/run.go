package run

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xhd2015/kode-ai/internal/ioread"
	"github.com/xhd2015/kode-ai/providers"
	"github.com/xhd2015/kode-ai/tools"
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
  --max-round N                   maximum number of chat rounds
  --token TOKEN                   the token
  --base-url BASE_URL             the base url
  --model MODEL                   llm model(default: gpt-4.1)
  --system PROMPT                 set the system prompt, PROMPT can also be a file
  --tool NAME                     predefined tool: batch_read_file,list_dir,grep_search...
                                  use kode chat --tool list to see all possible tools
  --tool-custom FILE              tool provided to LLM
  --tool-custom-json JSON         tool provided to LLM, in json, see tool example
  --tool-default-cwd DIR          the default working directory for tools, default current dir
                                  use --tool-default-cwd=none to unset it
  --mcp SERVER                    connect to MCP server (ip:port or command)
  --record FILE                   record chat history to given json file, which can be used to store and resume the chat
  --no-cache                      disable token caching
  --show-usage                    show usage from the file specified by --record
  --ignore-duplicate-msg          ignore duplicate user msg
  --log-request                   log http request
  --log-chat                      log chat(default: true)
  -c,--config FILE                load configuration from JSON file
  -v,--verbose                    show verbose info

Examples:
  kode chat 'hello'               chat with llm

  # provide tools to LLM
  kode chat --max-round=10 --tool list_dir "What's in current dir? wd is: $PWD"

  # connect to MCP server
  kode chat --mcp "localhost:8080" "What tools are available?"
  kode chat --mcp "my-mcp-server-command" "Execute a task"

  # agent-like chat
  kode chat --record tmp/chat.json --model=claude-3-7-sonnet --system=tmp/TRACE_PROMPT_CURSOR_LIKE.md --tool=batch_read_file --tool=list_dir --tool=run_terminal_cmd -v --ignore-duplicate-msg "<user_query>gather some critical information for it</user_query>"

  # show token usage
  kode chat --record tmp/chat.json --model=claude-3-7-sonnet --show-usage

Tool example:
  kode example --tool

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
	var baseUrl string
	var systemPrompt string
	var model string

	var recordFile string

	var tools []string
	var toolCustomFiles []string
	var toolCustomJSONs []string

	var showUsage bool
	var ignoreDuplicateMsg bool

	var toolDefaultCwd string
	var maxRound int
	var noCache bool

	var logRequest bool
	var logChatFlag *bool
	var verbose bool
	var mcpServers []string
	var configFile string

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
		Bool("--log-chat", &logChatFlag).
		Bool("-v,--verbose", &verbose).
		StringSlice("--mcp", &mcpServers).
		String("-c,--config", &configFile).
		Help("-h,--help", getHelp(baesCmd))

	args, err = flagsParser.Parse(args)
	if err != nil {
		return err
	}

	if len(tools) > 0 {
		for _, tool := range tools {
			if tool == "list" {
				return listTools()
			}
		}
	}

	// Load and apply configuration file
	config, err := LoadConfig(configFile)
	if err != nil {
		return err
	}

	err = ApplyConfig(config, &token, &maxRound, &baseUrl, &model, &systemPrompt, &tools, &toolCustomFiles, &toolCustomJSONs, &toolDefaultCwd, &recordFile, &noCache, &showUsage, &ignoreDuplicateMsg, &logRequest, &logChatFlag, &verbose, &mcpServers)
	if err != nil {
		return err
	}

	if toolDefaultCwd == "" {
		toolDefaultCwd = cwd
	} else if toolDefaultCwd == "none" {
		stat, _ := os.Stat("none")
		if stat == nil {
			toolDefaultCwd = ""
		}
	} else if toolDefaultCwd == "/none" || toolDefaultCwd == "NONE" {
		toolDefaultCwd = ""
	}

	if showUsage {
		if recordFile == "" {
			return fmt.Errorf("requires --record")
		}
		return showUsageFromRecordFile(recordFile)
	}
	if model == "list" {
		return listModels()
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

	model = providers.GetUnderlyingModel(model)
	apiShape, err := providers.GetModelAPIShape(model)
	if err != nil {
		return err
	}
	provider, err := providers.GetModelProvider(model)
	if err != nil {
		return err
	}

	resolvedOpts, err := ResolveProviderDefaultEnvOptions(apiShape, provider, toolDefaultCwd, token, baseUrl, defaultBaseURL)
	if err != nil {
		return err
	}

	var logChat bool = true
	if logChatFlag != nil {
		logChat = *logChatFlag
	}

	c := ChatHandler{
		APIShape: apiShape,
	}
	return c.Handle(model, resolvedOpts.BaseUrl, resolvedOpts.Token, msg, ChatOptions{
		maxRound: maxRound,

		systemPrompt:   systemPrompt,
		logRequest:     logRequest,
		toolBuiltins:   tools,
		toolFiles:      toolCustomFiles,
		toolJSONs:      toolCustomJSONs,
		recordFile:     recordFile,
		toolDefaultCwd: resolvedOpts.AbsDefaultToolCwd,

		noCache: noCache,

		ignoreDuplicateMsg: ignoreDuplicateMsg,
		logChat:            logChat,
		verbose:            verbose,

		mcpServers: mcpServers,
	})
}

type ResolvedOptions struct {
	AbsDefaultToolCwd string
	Token             string
	BaseUrl           string
}

func ResolveProviderDefaultEnvOptions(apiShape providers.APIShape, provider providers.Provider, defaultToolCwd string, token string, baseUrl string, defaultBaseUrl string) (ResolvedOptions, error) {
	var tokenEnvKey string
	var baseUrlEnvKey string
	switch provider {
	case providers.ProviderOpenAI:
		tokenEnvKey = "OPENAI_API_KEY"
		baseUrlEnvKey = "OPENAI_BASE_URL"
	case providers.ProviderAnthropic:
		tokenEnvKey = "ANTHROPIC_API_KEY"
		baseUrlEnvKey = "ANTHROPIC_BASE_URL"
	case providers.ProviderGemini:
		tokenEnvKey = "GEMINI_API_KEY"
		baseUrlEnvKey = "GEMINI_BASE_URL"
	case providers.ProviderMoonshot:
		tokenEnvKey = "MOONSHOT_API_KEY"
		baseUrlEnvKey = "MOONSHOT_BASE_URL"
	case providers.ProviderOpenRouter:
		tokenEnvKey = "OPENROUTER_API_KEY"
		baseUrlEnvKey = "OPENROUTER_BASE_URL"
	default:
		return ResolvedOptions{}, fmt.Errorf("resolve provider env, unsupported provider: %s", apiShape)
	}

	resolvedOpts, err := ResolveEnvOptions(defaultToolCwd, token, tokenEnvKey, baseUrl, baseUrlEnvKey, "KODE_DEFAULT_BASE_URL", defaultBaseUrl)
	if err != nil {
		return ResolvedOptions{}, err
	}
	return resolvedOpts, nil
}

func ResolveEnvOptions(defaultToolCwd string, token string, tokenEnvKey string, baseUrl string, baseUrlEnvKey string, defaultBaseUrlEnvKey string, defaultBaseUrl string) (ResolvedOptions, error) {
	var absDefaultToolCwd string
	if defaultToolCwd != "" {
		var err error
		absDefaultToolCwd, err = filepath.Abs(defaultToolCwd)
		if err != nil {
			return ResolvedOptions{}, err
		}
	}

	if token == "" {
		var envOption string
		if tokenEnvKey != "" {
			token = os.Getenv(tokenEnvKey)
			envOption = " or " + tokenEnvKey
		}
		if token == "" {
			return ResolvedOptions{}, errors.New("requires --token" + envOption)
		}
	}
	if baseUrl == "" {
		var envBaseURL string
		if baseUrlEnvKey != "" {
			envBaseURL = os.Getenv(baseUrlEnvKey)
		}
		if envBaseURL == "" && defaultBaseUrlEnvKey != "" {
			envBaseURL = os.Getenv(defaultBaseUrlEnvKey)
		}
		if envBaseURL == "" {
			envBaseURL = defaultBaseUrl
		}
		baseUrl = envBaseURL
	}
	return ResolvedOptions{
		AbsDefaultToolCwd: absDefaultToolCwd,
		Token:             token,
		BaseUrl:           baseUrl,
	}, nil
}

func listModels() error {
	for _, model := range providers.AllModels {
		fmt.Println(model)
	}
	return nil
}

const viewHelp = `
kode view <files...>

Options:
  --last-assistant                show the last assistant message
  --show-usage                    show usage from the file specified by --record
  --tools                         show tools used in the chats
  -v,--verbose                    show verbose info

Examples:
  kode view tmp/chat.json
  kode view tmp/chat.json --last-assistant
  kode view tmp/chat.json --show-usage
  kode view tmp/chat.json --tools
`

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
		Help("-h,--help", viewHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return fmt.Errorf("requires files, try `kode view --help`")
	}
	if args[0] == "--help" || args[0] == "help" {
		fmt.Print(strings.TrimPrefix(viewHelp, "\n"))
		return nil
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
				provider, err := providers.GetModelAPIShape(m.Model)
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
	var tool bool
	var config bool
	var configDef bool
	args, err := flags.Bool("--tool", &tool).
		Bool("--config", &config).
		Bool("--config-def", &configDef).
		Parse(args)
	if err != nil {
		return err
	}
	if tool {
		fmt.Println(tools.ExampleTool)
		return nil
	}
	if config {
		fmt.Println(ExampleConfig)
		return nil
	}
	if configDef {
		fmt.Println("# config.go")
		fmt.Println(ConfigDef)
		fmt.Println("# tool.go")
		fmt.Println(tools.UnifiedToolDef)
		return nil
	}
	const examples = `
# fetch trace first
spl trace 5096cd609a7ddb8841b810d0dfa37b65 --dump tmp

# ask questions about the trace
kode chat --record tmp/chat-why-empty-instalments.json --model=claude-3-7-sonnet --system=tmp/TRACE_PROMPT.md --tool=batch_read_file --tool=list_dir --tool=run_terminal_cmd -v --ignore-duplicate-msg "<user_query>为什么返回的instalments列表是空的?</user_query> working directory: $PWD/some_trace"

# Test cache (NOTE)
kode chat --model=claude-3-7-sonnet --record tmp/cache-test/record.json -v --system='You are a Lucas
, a 18-year-old cowboy' 'Who are you?'
kode chat --model=claude-3-7-sonnet --record tmp/cache-test/record.json -v --system='You are a Lucas
, a 18-year-old cowboy' 'How old are you?'

# Trace compact
kode chat --model=claude-sonnet-4 --system tmp/TRACE_PROMPT_STRUCTURE.md "<user_query>please compact slightly the tr
ee output to save tokens without losing significant information</user_query> current working directory: $PWD/trace_working_dir/some_dir tree output: $(llm-tools tree --collapse --dir-only trace_working_dir/some_dir)"

# Test Gemini
kode chat --model=gemini-2.5-pro --tool get_workspace_root --tool list_dir --record=chat.json --ignore-duplicate-msg --max-round=10 "What's files under my current directory?" --system="You are a passionate coding agent that actively answer user's question without hesitation. When user asks a question, you understand it aggresively and do it without asking for proceeding."
`

	fmt.Print(strings.TrimPrefix(examples, "\n"))
	return nil
}
