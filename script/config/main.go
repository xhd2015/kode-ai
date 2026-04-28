package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type Catalog struct {
	Root      string       `json:"root"`
	Files     []string     `json:"files"`
	Models    []ModelEntry `json:"models"`
	Aliases   []AliasEntry `json:"aliases"`
	Constants []ConstEntry `json:"constants"`
}

type ModelEntry struct {
	Key        string    `json:"key"`
	Name       string    `json:"name"`
	ConstName  string    `json:"const_name,omitempty"`
	Provider   string    `json:"provider"`
	APIShape   string    `json:"api_shape"`
	MapName    string    `json:"map_name"`
	SourceFile string    `json:"source_file"`
	SourceLine int       `json:"source_line,omitempty"`
	Cost       ModelCost `json:"cost"`
	Aliases    []string  `json:"aliases,omitempty"`
}

type ModelCost struct {
	InputUSDPer1M           string `json:"input_usd_per_1m,omitempty"`
	InputCacheWriteUSDPer1M string `json:"input_cache_write_usd_per_1m,omitempty"`
	InputCacheReadUSDPer1M  string `json:"input_cache_read_usd_per_1m,omitempty"`
	OutputUSDPer1M          string `json:"output_usd_per_1m,omitempty"`
}

type AliasEntry struct {
	Alias       string `json:"alias"`
	AliasConst  string `json:"alias_const,omitempty"`
	Target      string `json:"target"`
	TargetConst string `json:"target_const,omitempty"`
	SourceFile  string `json:"source_file"`
	SourceLine  int    `json:"source_line,omitempty"`
}

type ConstEntry struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	SourceFile string `json:"source_file"`
	SourceLine int    `json:"source_line,omitempty"`
}

type parseContext struct {
	root        string
	constByName map[string]string
	constByVal  map[string]string
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 && args[0] == "config" {
		args = args[1:]
	}
	if len(args) > 0 {
		switch args[0] {
		case "show":
			return handleShow(args[1:])
		case "export":
			return handleExport(args[1:])
		case "serve":
			return handleServe(args[1:])
		case "help", "--help", "-h":
			fmt.Print(helpText())
			return nil
		}
	}
	return handleServe(args)
}

func handleShow(args []string) error {
	var rootFlag string
	fs := newFlagSet("show")
	fs.StringVar(&rootFlag, "root", "", "kode-ai repository root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), ", "))
	}
	catalog, err := BuildCatalog(rootFlag)
	if err != nil {
		return err
	}
	printCatalog(catalog)
	return nil
}

func handleExport(args []string) error {
	var rootFlag string
	fs := newFlagSet("export")
	fs.StringVar(&rootFlag, "root", "", "kode-ai repository root")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: %s export [--root DIR] <file>", appName())
	}
	catalog, err := BuildCatalog(rootFlag)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(fs.Arg(0), data, 0644); err != nil {
		return fmt.Errorf("write export: %w", err)
	}
	fmt.Printf("Exported %d models to: %s\n", len(catalog.Models), fs.Arg(0))
	return nil
}

func handleServe(args []string) error {
	var rootFlag string
	var listen string
	var noOpen bool
	fs := newFlagSet("serve")
	fs.StringVar(&rootFlag, "root", "", "kode-ai repository root")
	fs.StringVar(&listen, "listen", "127.0.0.1:0", "listen address")
	fs.BoolVar(&noOpen, "no-open", false, "do not open browser")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), ", "))
	}
	root, err := resolveRoot(rootFlag)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	url := "http://" + listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, indexHTML)
	})
	mux.HandleFunc("/api/catalog", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		catalog, err := BuildCatalog(root)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = json.NewEncoder(w).Encode(catalog)
	})

	fmt.Printf("Model config UI available at: %s\n", url)
	fmt.Printf("Repository root: %s\n", root)
	fmt.Println("Press Ctrl+C to stop.")
	if !noOpen {
		openBrowser(url)
	}
	return http.Serve(listener, mux)
}

func BuildCatalog(rootFlag string) (*Catalog, error) {
	root, err := resolveRoot(rootFlag)
	if err != nil {
		return nil, err
	}
	ctx := &parseContext{
		root:        root,
		constByName: map[string]string{},
		constByVal:  map[string]string{},
	}

	var files []string
	consts, err := parseModelConstants(ctx, filepath.Join(root, "types", "model.go"))
	if err != nil {
		return nil, err
	}
	files = append(files, "types/model.go")

	modelFiles, err := filepath.Glob(filepath.Join(root, "types", "models", "*.go"))
	if err != nil {
		return nil, err
	}
	sort.Strings(modelFiles)
	var models []ModelEntry
	for _, file := range modelFiles {
		rel := mustRel(root, file)
		files = append(files, rel)
		entries, err := parseModelInfoMaps(ctx, file)
		if err != nil {
			return nil, err
		}
		models = append(models, entries...)
	}

	aliasFiles := []string{
		filepath.Join(root, "types", "providers", "model.go"),
		filepath.Join(root, "providers", "model.go"),
	}
	var aliases []AliasEntry
	for _, file := range aliasFiles {
		if _, err := os.Stat(file); err != nil {
			continue
		}
		files = append(files, mustRel(root, file))
		entries, err := parseAliasMaps(ctx, file)
		if err != nil {
			return nil, err
		}
		aliases = append(aliases, entries...)
	}

	constByValue := map[string]string{}
	for _, c := range consts {
		if _, ok := constByValue[c.Value]; !ok {
			constByValue[c.Value] = c.Name
		}
	}
	aliasesByTarget := map[string][]string{}
	for _, alias := range aliases {
		aliasesByTarget[alias.Target] = append(aliasesByTarget[alias.Target], alias.Alias)
	}
	for i := range models {
		if models[i].ConstName == "" {
			models[i].ConstName = constByValue[models[i].Name]
		}
		models[i].Aliases = aliasesByTarget[models[i].Name]
		sort.Strings(models[i].Aliases)
	}

	sort.Slice(models, func(i, j int) bool {
		if models[i].Provider != models[j].Provider {
			return models[i].Provider < models[j].Provider
		}
		return models[i].Name < models[j].Name
	})
	sort.Slice(aliases, func(i, j int) bool {
		return aliases[i].Alias < aliases[j].Alias
	})
	sort.Slice(consts, func(i, j int) bool {
		return consts[i].Name < consts[j].Name
	})
	files = uniqueStrings(files)

	return &Catalog{
		Root:      root,
		Files:     files,
		Models:    models,
		Aliases:   aliases,
		Constants: consts,
	}, nil
}

func parseModelConstants(ctx *parseContext, file string) ([]ConstEntry, error) {
	fset, parsed, err := parseGoFile(file)
	if err != nil {
		return nil, err
	}
	var consts []ConstEntry
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				value, ok := stringValue(valueSpec.Values[i], ctx.constByName)
				if !ok {
					continue
				}
				ctx.constByName[name.Name] = value
				ctx.constByName["types."+name.Name] = value
				if _, exists := ctx.constByVal[value]; !exists {
					ctx.constByVal[value] = name.Name
				}
				consts = append(consts, ConstEntry{
					Name:       name.Name,
					Value:      value,
					SourceFile: mustRel(ctx.root, file),
					SourceLine: fset.Position(name.Pos()).Line,
				})
			}
		}
	}
	return consts, nil
}

func parseModelInfoMaps(ctx *parseContext, file string) ([]ModelEntry, error) {
	fset, parsed, err := parseGoFile(file)
	if err != nil {
		return nil, err
	}
	var models []ModelEntry
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.CompositeLit)
				if !ok || !strings.HasSuffix(name.Name, "Models") {
					continue
				}
				entries := parseModelMap(ctx, fset, file, name.Name, lit)
				models = append(models, entries...)
			}
		}
	}
	return models, nil
}

func parseModelMap(ctx *parseContext, fset *token.FileSet, file string, mapName string, lit *ast.CompositeLit) []ModelEntry {
	var models []ModelEntry
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := stringValue(kv.Key, ctx.constByName)
		if !ok {
			continue
		}
		entry := ModelEntry{
			Key:        key,
			Name:       key,
			MapName:    mapName,
			SourceFile: mustRel(ctx.root, file),
			SourceLine: fset.Position(kv.Pos()).Line,
		}
		valueLit, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			models = append(models, entry)
			continue
		}
		for _, field := range valueLit.Elts {
			fieldKV, ok := field.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			fieldName := identName(fieldKV.Key)
			switch fieldName {
			case "Name":
				if value, ok := stringValue(fieldKV.Value, ctx.constByName); ok {
					entry.Name = value
				}
			case "Provider":
				entry.Provider = providerValue(fieldKV.Value)
			case "APIShape":
				entry.APIShape = apiShapeValue(fieldKV.Value)
			case "Cost":
				entry.Cost = parseCost(ctx, fieldKV.Value)
			}
		}
		entry.ConstName = ctx.constByVal[entry.Name]
		models = append(models, entry)
	}
	return models
}

func parseCost(ctx *parseContext, expr ast.Expr) ModelCost {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok {
		return ModelCost{}
	}
	var cost ModelCost
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		value, _ := stringValue(kv.Value, ctx.constByName)
		switch identName(kv.Key) {
		case "InputUSDPer1M":
			cost.InputUSDPer1M = value
		case "InputCacheWriteUSDPer1M":
			cost.InputCacheWriteUSDPer1M = value
		case "InputCacheReadUSDPer1M":
			cost.InputCacheReadUSDPer1M = value
		case "OutputUSDPer1M":
			cost.OutputUSDPer1M = value
		}
	}
	return cost
}

func parseAliasMaps(ctx *parseContext, file string) ([]AliasEntry, error) {
	fset, parsed, err := parseGoFile(file)
	if err != nil {
		return nil, err
	}
	var aliases []AliasEntry
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				if name.Name != "modelAlias" || i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.CompositeLit)
				if !ok {
					continue
				}
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					alias, aliasOK := stringValue(kv.Key, ctx.constByName)
					target, targetOK := stringValue(kv.Value, ctx.constByName)
					if !aliasOK || !targetOK {
						continue
					}
					aliases = append(aliases, AliasEntry{
						Alias:       alias,
						AliasConst:  constRefName(kv.Key, ctx),
						Target:      target,
						TargetConst: constRefName(kv.Value, ctx),
						SourceFile:  mustRel(ctx.root, file),
						SourceLine:  fset.Position(kv.Pos()).Line,
					})
				}
			}
		}
	}
	return aliases, nil
}

func parseGoFile(file string) (*token.FileSet, *ast.File, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", file, err)
	}
	return fset, parsed, nil
}

func stringValue(expr ast.Expr, constByName map[string]string) (string, bool) {
	switch expr := expr.(type) {
	case *ast.BasicLit:
		if expr.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(expr.Value)
		return value, err == nil
	case *ast.Ident:
		value, ok := constByName[expr.Name]
		return value, ok
	case *ast.SelectorExpr:
		key := expr.Sel.Name
		if ident, ok := expr.X.(*ast.Ident); ok {
			key = ident.Name + "." + expr.Sel.Name
		}
		value, ok := constByName[key]
		if ok {
			return value, true
		}
		value, ok = constByName[expr.Sel.Name]
		return value, ok
	}
	return "", false
}

func constRefName(expr ast.Expr, ctx *parseContext) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.BasicLit:
		value, ok := stringValue(expr, ctx.constByName)
		if ok {
			return ctx.constByVal[value]
		}
	}
	return ""
}

func identName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}

func providerValue(expr ast.Expr) string {
	symbol := symbolName(expr)
	switch symbol {
	case "ProviderAnthropic":
		return "anthropic"
	case "ProviderGemini":
		return "gemini"
	case "ProviderOpenAI":
		return "openai"
	case "ProviderMoonshot":
		return "moonshot"
	case "ProviderDeepSeek":
		return "deepseek"
	case "ProviderQwen":
		return "qwen"
	case "ProviderOpenRouter":
		return "openrouter"
	default:
		return trimSymbolPrefix(symbol, "Provider")
	}
}

func apiShapeValue(expr ast.Expr) string {
	symbol := symbolName(expr)
	switch symbol {
	case "APIShapeOpenAI":
		return "openai"
	case "APIShapeAnthropic":
		return "anthropic"
	case "APIShapeGemini":
		return "gemini"
	default:
		return trimSymbolPrefix(symbol, "APIShape")
	}
}

func symbolName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.Sel.Name
	case *ast.BasicLit:
		value, err := strconv.Unquote(expr.Value)
		if err == nil {
			return value
		}
	}
	return ""
}

func trimSymbolPrefix(symbol string, prefix string) string {
	if strings.HasPrefix(symbol, prefix) {
		return strings.ToLower(strings.TrimPrefix(symbol, prefix))
	}
	return symbol
}

func resolveRoot(rootFlag string) (string, error) {
	var candidates []string
	if rootFlag != "" {
		candidates = append(candidates, rootFlag)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, cwd, filepath.Join(cwd, "kode-ai"))
		candidates = append(candidates, ancestors(cwd)...)
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		scriptRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
		candidates = append(candidates, scriptRoot)
		candidates = append(candidates, ancestors(filepath.Dir(file))...)
	}
	for _, candidate := range uniqueStrings(candidates) {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if isRepoRoot(abs) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("cannot locate kode-ai root; pass --root DIR")
}

func ancestors(start string) []string {
	var result []string
	dir := filepath.Clean(start)
	for {
		result = append(result, dir)
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return result
}

func isRepoRoot(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "types", "model.go")); err != nil {
		return false
	}
	if stat, err := os.Stat(filepath.Join(dir, "types", "models")); err != nil || !stat.IsDir() {
		return false
	}
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "module github.com/xhd2015/kode-ai")
}

func mustRel(root string, file string) string {
	rel, err := filepath.Rel(root, file)
	if err != nil {
		return file
	}
	return filepath.ToSlash(rel)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func printCatalog(catalog *Catalog) {
	fmt.Printf("Repository root: %s\n", catalog.Root)
	fmt.Printf("Parsed files: %d  Models: %d  Aliases: %d\n\n", len(catalog.Files), len(catalog.Models), len(catalog.Aliases))
	fmt.Printf("%-34s %-11s %-10s %8s %8s %8s %8s %s\n", "MODEL", "PROVIDER", "API", "IN", "CACHE-W", "CACHE-R", "OUT", "SOURCE")
	for _, model := range catalog.Models {
		fmt.Printf("%-34s %-11s %-10s %8s %8s %8s %8s %s\n",
			model.Name,
			model.Provider,
			model.APIShape,
			emptyDash(model.Cost.InputUSDPer1M),
			emptyDash(model.Cost.InputCacheWriteUSDPer1M),
			emptyDash(model.Cost.InputCacheReadUSDPer1M),
			emptyDash(model.Cost.OutputUSDPer1M),
			sourceRef(model.SourceFile, model.SourceLine),
		)
	}
	if len(catalog.Aliases) > 0 {
		fmt.Println("\nAliases:")
		for _, alias := range catalog.Aliases {
			fmt.Printf("  %s -> %s (%s)\n", alias.Alias, alias.Target, sourceRef(alias.SourceFile, alias.SourceLine))
		}
	}
}

func sourceRef(file string, line int) string {
	if line <= 0 {
		return file
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func emptyDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, helpText())
	}
	return fs
}

func appName() string {
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "kode-ai-config"
}

func helpText() string {
	return fmt.Sprintf(`%s inspects kode-ai model and pricing source config.

Usage:
  %s [config] [--root DIR] [--listen ADDR] [--no-open]
  %s [config] show [--root DIR]
  %s [config] export [--root DIR] <file>

The web UI and API parse Go source files directly:
  - types/model.go
  - types/models/*.go
  - types/providers/model.go
  - providers/model.go
`, appName(), appName(), appName(), appName())
}

func writeJSONError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return
	}
	_ = cmd.Start()
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Kode AI Model Config</title>
<style>
  * { box-sizing: border-box; }
  body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f6f7f8; color: #1f252d; }
  header { padding: 20px 24px 14px; background: #ffffff; border-bottom: 1px solid #dfe3e8; }
  h1 { margin: 0 0 8px; font-size: 22px; line-height: 1.2; }
  .meta { color: #65717f; font-size: 13px; overflow-wrap: anywhere; }
  main { padding: 16px 24px 28px; }
  .toolbar { display: grid; grid-template-columns: minmax(220px, 1fr) 160px 160px auto; gap: 10px; margin-bottom: 14px; align-items: center; }
  input, select, button { height: 36px; border: 1px solid #c9d1d9; border-radius: 4px; font-size: 14px; background: #fff; color: #1f252d; padding: 0 10px; }
  button { cursor: pointer; background: #2f7dd1; border-color: #2f7dd1; color: #fff; padding: 0 16px; }
  button:hover { background: #2468b3; }
  table { width: 100%; border-collapse: collapse; background: #fff; border: 1px solid #dfe3e8; font-size: 13px; }
  th, td { padding: 9px 10px; border-bottom: 1px solid #edf0f3; text-align: left; vertical-align: top; }
  th { position: sticky; top: 0; background: #f1f4f7; color: #4b5563; font-weight: 700; z-index: 1; }
  tbody tr:hover { background: #f8fafc; }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .chip { display: inline-block; padding: 2px 7px; border-radius: 999px; background: #e9f2fd; color: #245f9e; font-size: 12px; margin: 1px 3px 1px 0; }
  .muted { color: #6b7280; }
  .err { margin: 14px 0; padding: 10px 12px; background: #fdecec; color: #a12a2a; border-radius: 4px; display: none; }
  @media (max-width: 900px) {
    .toolbar { grid-template-columns: 1fr 1fr; }
    table { font-size: 12px; }
    th, td { padding: 7px 8px; }
  }
</style>
</head>
<body>
<header>
  <h1>Model Config</h1>
  <div id="summary" class="meta"></div>
  <div id="root" class="meta"></div>
</header>
<main>
  <div class="toolbar">
    <input id="query" placeholder="Filter models, files, aliases" oninput="render()" />
    <select id="provider" onchange="render()"><option value="">All providers</option></select>
    <select id="shape" onchange="render()"><option value="">All API shapes</option></select>
    <button onclick="load()">Reload</button>
  </div>
  <div id="err" class="err"></div>
  <table>
    <thead>
      <tr>
        <th>Model</th><th>Provider</th><th>API</th>
        <th class="num">Input</th><th class="num">Cache Write</th><th class="num">Cache Read</th><th class="num">Output</th>
        <th>Aliases</th><th>Source</th>
      </tr>
    </thead>
    <tbody id="rows"></tbody>
  </table>
</main>
<script>
let catalog = { models: [] };
const $ = id => document.getElementById(id);

async function load() {
  try {
    const r = await fetch('/api/catalog');
    const d = await r.json();
    if (d.error) throw new Error(d.error);
    catalog = d;
    $('err').style.display = 'none';
    $('root').textContent = d.root || '';
    $('summary').textContent = d.models.length + ' models, ' + d.aliases.length + ' aliases, ' + d.files.length + ' source files';
    fillSelect('provider', [...new Set(d.models.map(m => m.provider).filter(Boolean))]);
    fillSelect('shape', [...new Set(d.models.map(m => m.api_shape).filter(Boolean))]);
    render();
  } catch (e) {
    $('err').textContent = e.message || 'Failed to load catalog';
    $('err').style.display = 'block';
  }
}

function fillSelect(id, values) {
  const el = $(id);
  const current = el.value;
  const label = id === 'shape' ? 'All API shapes' : 'All providers';
  el.innerHTML = '<option value="">' + esc(label) + '</option>' + values.sort().map(v => '<option value="' + esc(v) + '">' + esc(v) + '</option>').join('');
  el.value = current;
}

function render() {
  const q = $('query').value.toLowerCase();
  const provider = $('provider').value;
  const shape = $('shape').value;
  const rows = catalog.models.filter(m => {
    if (provider && m.provider !== provider) return false;
    if (shape && m.api_shape !== shape) return false;
    const source = sourceRef(m);
    const hay = [m.name, m.key, m.const_name, m.provider, m.api_shape, source, ...(m.aliases || [])].join(' ').toLowerCase();
    return !q || hay.includes(q);
  });
  $('rows').innerHTML = rows.map(m =>
    '<tr>' +
      '<td><strong>' + esc(m.name) + '</strong><div class="muted">' + esc(m.const_name || m.key || '') + '</div></td>' +
      '<td>' + esc(m.provider || '') + '</td>' +
      '<td>' + esc(m.api_shape || '') + '</td>' +
      '<td class="num">' + esc(m.cost.input_usd_per_1m || '-') + '</td>' +
      '<td class="num">' + esc(m.cost.input_cache_write_usd_per_1m || '-') + '</td>' +
      '<td class="num">' + esc(m.cost.input_cache_read_usd_per_1m || '-') + '</td>' +
      '<td class="num">' + esc(m.cost.output_usd_per_1m || '-') + '</td>' +
      '<td>' + (m.aliases || []).map(a => '<span class="chip">' + esc(a) + '</span>').join('') + '</td>' +
      '<td class="muted">' + esc(sourceRef(m)) + '<div>' + esc(m.map_name || '') + '</div></td>' +
    '</tr>').join('');
}

function sourceRef(m) {
  if (!m || !m.source_file) return '';
  return m.source_line ? m.source_file + ':' + m.source_line : m.source_file;
}

function esc(v) {
  return String(v ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
}

load();
</script>
</body>
</html>`
