# OpenCode Public Zen Request Notes

Research target: `opencode` commit `13006d6d7c9c84e790c9ecc001e6b42b34c05b96`.

This note summarizes how the official OpenCode client uses the public/free Zen path and where the `x-opencode-*` request headers are generated. These headers are client/session metadata; the code does not show them as a quota bypass.

## Public API key selection

The `opencode` provider declares `OPENCODE_API_KEY` as its env key and uses the Zen base URL:

- `opencode/packages/opencode/test/tool/fixtures/models-api.json:57813`
- `opencode/packages/opencode/test/tool/fixtures/models-api.json:57815`
- `opencode/packages/opencode/test/tool/fixtures/models-api.json:57817`

Provider setup checks whether the user has any real OpenCode credential:

- an env key listed by the provider, currently `OPENCODE_API_KEY`
- saved provider auth
- `provider.opencode.options.apiKey` in config

If none exists, OpenCode enters the public/free path:

- it removes every model whose `cost.input` is not `0`
- it returns `options: { apiKey: "public" }`

Source:

- `opencode/packages/opencode/src/provider/provider.ts:160`
- `opencode/packages/opencode/src/provider/provider.ts:166`
- `opencode/packages/opencode/src/provider/provider.ts:171`
- `opencode/packages/opencode/src/provider/provider.ts:178`

The string `public` is not generated dynamically. It is a literal fallback API key. Later, provider resolution copies this API key into the provider options when creating the model client:

- `opencode/packages/opencode/src/provider/provider.ts:1524`
- `opencode/packages/opencode/src/provider/provider.ts:1545`
- `opencode/packages/opencode/src/provider/provider.ts:1546`

The LLM HTTP transport uses bearer auth by default, so `apiKey: "public"` becomes:

```http
Authorization: Bearer public
```

Source:

- `opencode/packages/llm/src/route/transport/http.ts:55`
- `opencode/packages/llm/src/route/transport/http.ts:61`
- `opencode/packages/llm/src/route/transport/http.ts:93`
- `opencode/packages/llm/src/route/auth.ts:126`
- `opencode/packages/llm/src/route/auth.ts:133`

## Request URL

The provider base URL is:

```text
https://opencode.ai/zen/v1
```

For OpenAI-compatible chat completions, the route appends `/chat/completions`, making the effective public/free request URL:

```text
https://opencode.ai/zen/v1/chat/completions
```

Source:

- `opencode/packages/opencode/test/tool/fixtures/models-api.json:57817`
- `opencode/packages/llm/src/route/transport/http.ts:48`

## `x-opencode-*` headers

The headers are assembled for any provider whose ID starts with `opencode`, so they apply to `opencode` public/free requests as well as authenticated `opencode` and `opencode-go` requests.

Generated headers:

```http
x-opencode-project: <project id>
x-opencode-session: <session id>
x-opencode-request: <user message id>
x-opencode-client: <client name>
User-Agent: opencode/<installation version>
```

Example values:

```http
x-opencode-project: 13006d6d7c9c84e790c9ecc001e6b42b34c05b96
x-opencode-session: ses_f899e8d33ab7U1x2Y3z4A5b6C7
x-opencode-request: msg_019ab9c4d2a1001x2Y3z4A5b6C
x-opencode-client: cli
User-Agent: opencode/local
```

The example IDs are illustrative. To generate values close to what a local OpenCode CLI request would send for the current repository, run:

```sh
(cd kode-ai && go run ./script/gen-opencode-headers)
```

For another repository or working directory:

```sh
(cd kode-ai && go run ./script/gen-opencode-headers --dir /path/to/project)
```

The script prints ready-to-paste `kode chat -H` flags.

For machine-readable output:

```sh
(cd kode-ai && go run ./script/gen-opencode-headers --json)
```

Source:

- `opencode/packages/opencode/src/session/llm.ts:330`
- `opencode/packages/opencode/src/session/llm.ts:334`
- `opencode/packages/opencode/src/session/llm.ts:337`
- `opencode/packages/opencode/src/session/llm.ts:338`
- `opencode/packages/opencode/src/session/llm.ts:339`
- `opencode/packages/opencode/src/session/llm.ts:340`
- `opencode/packages/opencode/src/session/llm.ts:341`

The assembled header map is passed into both LLM execution paths:

- native runtime input: `opencode/packages/opencode/src/session/llm.ts:365`
- native runtime input: `opencode/packages/opencode/src/session/llm.ts:368`
- AI SDK `streamText` input: `opencode/packages/opencode/src/session/llm.ts:404`
- AI SDK `streamText` input: `opencode/packages/opencode/src/session/llm.ts:440`

## Header value generation

### `x-opencode-project`

Value:

```ts
(yield* InstanceState.context).project.id
```

Source:

- `opencode/packages/opencode/src/session/llm.ts:330`
- `opencode/packages/opencode/src/session/llm.ts:331`

Project ID mechanism:

- `ProjectID` is a branded string, with `global` as the fallback value.
- `Project.fromDirectory` first looks for a cached project ID in an `opencode` file under the git directory/common directory.
- If there is no cached ID, it runs `git rev-list --max-parents=0 HEAD`, sorts root commits, uses the first root commit hash as the project ID, and writes it back to the git common directory as `opencode`.
- If there is no git repository or no usable ID, it falls back to `global`.

Source:

- `opencode/packages/opencode/src/project/schema.ts:5`
- `opencode/packages/opencode/src/project/schema.ts:9`
- `opencode/packages/opencode/src/project/schema.ts:11`
- `opencode/packages/opencode/src/project/project.ts:185`
- `opencode/packages/opencode/src/project/project.ts:193`
- `opencode/packages/opencode/src/project/project.ts:203`
- `opencode/packages/opencode/src/project/project.ts:214`
- `opencode/packages/opencode/src/project/project.ts:239`
- `opencode/packages/opencode/src/project/project.ts:243`
- `opencode/packages/opencode/src/project/project.ts:251`
- `opencode/packages/opencode/src/project/project.ts:253`
- `opencode/packages/opencode/src/project/project.ts:257`

### `x-opencode-session`

Value:

```ts
input.sessionID
```

Source:

- `opencode/packages/opencode/src/session/llm.ts:338`

Session ID mechanism:

- sessions are created with `SessionID.descending(input.id)`
- `SessionID` delegates to `@opencode-ai/core/session`
- if no ID is supplied, it returns `ses_` plus a descending monotonic identifier
- the identifier uses current `Date.now()`, a per-process counter, bitwise inversion for descending order, 6 timestamp bytes encoded as hex, and random base62 bytes

Source:

- `opencode/packages/opencode/src/session/session.ts:521`
- `opencode/packages/opencode/src/session/session.ts:533`
- `opencode/packages/opencode/src/session/session.ts:534`
- `opencode/packages/core/src/session.ts:7`
- `opencode/packages/core/src/session.ts:10`
- `opencode/packages/core/src/util/identifier.ts:28`
- `opencode/packages/core/src/util/identifier.ts:29`
- `opencode/packages/core/src/util/identifier.ts:35`
- `opencode/packages/core/src/util/identifier.ts:37`
- `opencode/packages/core/src/util/identifier.ts:39`
- `opencode/packages/core/src/util/identifier.ts:41`
- `opencode/packages/core/src/util/identifier.ts:46`

### `x-opencode-request`

Value:

```ts
input.user.id
```

Source:

- `opencode/packages/opencode/src/session/llm.ts:339`

User request/message ID mechanism:

- the TUI pre-generates a `MessageID.ascending()` before submitting a prompt
- the prompt service uses `input.messageID ?? MessageID.ascending()`
- `MessageID.ascending()` delegates to `Identifier.ascending("message")`
- generated IDs start with `msg_`
- the shared ID generator uses current `Date.now()`, a per-process counter, 6 timestamp bytes encoded as hex, and random base62 bytes

Source:

- `opencode/packages/opencode/src/cli/cmd/tui/component/prompt/index.tsx:1094`
- `opencode/packages/opencode/src/session/prompt.ts:717`
- `opencode/packages/opencode/src/session/prompt.ts:718`
- `opencode/packages/opencode/src/session/schema.ts:10`
- `opencode/packages/opencode/src/session/schema.ts:13`
- `opencode/packages/opencode/src/id/id.ts:22`
- `opencode/packages/opencode/src/id/id.ts:30`
- `opencode/packages/opencode/src/id/id.ts:41`
- `opencode/packages/opencode/src/id/id.ts:51`
- `opencode/packages/opencode/src/id/id.ts:52`
- `opencode/packages/opencode/src/id/id.ts:58`
- `opencode/packages/opencode/src/id/id.ts:60`
- `opencode/packages/opencode/src/id/id.ts:62`
- `opencode/packages/opencode/src/id/id.ts:64`
- `opencode/packages/opencode/src/id/id.ts:69`

### `x-opencode-client`

Value:

```ts
flags.client
```

Source:

- `opencode/packages/opencode/src/session/llm.ts:340`

Client value mechanism:

- read from env/config key `OPENCODE_CLIENT`
- defaults to `cli`

Source:

- `opencode/packages/opencode/src/effect/runtime-flags.ts:54`

### `User-Agent`

Value:

```ts
`opencode/${InstallationVersion}`
```

Source:

- `opencode/packages/opencode/src/session/llm.ts:341`

Version mechanism:

- `InstallationVersion` is the build-time global `OPENCODE_VERSION`
- if the build-time global is unavailable, it falls back to `local`

Source:

- `opencode/packages/core/src/installation/version.ts:1`
- `opencode/packages/core/src/installation/version.ts:6`

## Important limit behavior

When the free tier is exhausted, OpenCode treats the server response as a real limit. The retry/action layer maps `FreeUsageLimitError` to the upsell message `Free usage exceeded, subscribe to Go`; the headers above are not used in this code as a bypass.

Source:

- `opencode/packages/opencode/src/session/retry.ts:75`
- `opencode/packages/opencode/src/session/retry.ts:76`
- `opencode/packages/opencode/src/session/retry.ts:83`
