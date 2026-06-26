// Package boot — builder.go houses the Builder struct and phase methods.
// Build() in boot.go orchestrates these phases in sequence; each method reads
// from b.* set by prior phases and writes its own outputs back to b.*.
// Splitting the monolith this way makes each phase independently readable and
// lets future tests inject a pre-built builder at any phase boundary.
package boot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/agentlog"
	"reasonix/internal/command"
	"reasonix/internal/config"
	"reasonix/internal/control"
	"reasonix/internal/event"
	"reasonix/internal/history"
	"reasonix/internal/hook"
	"reasonix/internal/instruction"
	"reasonix/internal/installsource"
	"reasonix/internal/jobs"
	"reasonix/internal/learn"
	"reasonix/internal/lsp"
	"reasonix/internal/memory"
	"reasonix/internal/mesh"
	"reasonix/internal/migration"
	"reasonix/internal/netclient"
	"reasonix/internal/outputstyle"
	"reasonix/internal/permission"
	"reasonix/internal/plugin"
	"reasonix/internal/provider"
	"reasonix/internal/sandbox"
	"reasonix/internal/scheduler"
	"reasonix/internal/tool/sessiontool"
	"reasonix/internal/skill"
	"reasonix/internal/tool"
	"reasonix/internal/tool/builtin"
)

// builder holds all intermediate state produced by sequential Build phases.
// Callers create a builder via newBuilder, call each phase method in order,
// then call assemble() to obtain the ready-to-drive controller.
type builder struct {
	ctx  context.Context
	opts Options

	// ── loadConfig outputs ──────────────────────────────────────────────────
	cfg          *config.Config
	root         string
	entry        *config.ProviderEntry
	entryPrice   *provider.Pricing
	modelRef     string
	tokenEconomy bool
	keepPolicy   agent.KeepPolicy
	sink         event.Sink
	proxySpec    netclient.ProxySpec
	balanceClient *http.Client
	sessionDir   string
	maxSteps     int
	stderr       io.Writer

	// ── buildProviders outputs ───────────────────────────────────────────────
	execProv        provider.Provider
	compressionProv provider.Provider
	visionProv      provider.Provider
	webExtractProv  provider.Provider

	// ── buildPrompt outputs ──────────────────────────────────────────────────
	sysPrompt     string
	mem           *memory.Set
	projectChecks []instruction.VerifyCheck
	skillStore    *skill.Store
	skills        []skill.Skill
	allSkillStore *skill.Store
	allSkills     []skill.Skill

	// ── buildToolRegistry outputs ────────────────────────────────────────────
	reg         *tool.Registry
	bashSpec    sandbox.Spec
	bashTimeout time.Duration
	searchSpec  builtin.SearchSpec
	shell       sandbox.Shell

	// ── buildPlugins outputs ─────────────────────────────────────────────────
	pluginHost       *plugin.Host
	cleanup          func()
	lspMgr           *lsp.Manager
	onDemandMCPSpecs map[string]plugin.Spec
	onDemandMCPNames []string

	// ── buildPermissions outputs ─────────────────────────────────────────────
	policy       permission.Policy
	headlessGate *permission.Gate
	hookRunner   *hook.Runner

	// ── buildSubagents outputs ───────────────────────────────────────────────
	jm            *jobs.Manager
	subagentStore *agent.SubagentStore

	// ── buildToolSurface outputs ─────────────────────────────────────────────
	cmds []command.Command

	// ── buildExecutor outputs ────────────────────────────────────────────────
	executor   *agent.Agent
	runner     agent.Runner
	label      string
	classifier *control.ProviderAutoPlanClassifier

	// ── buildLearner outputs ─────────────────────────────────────────────────
	lc *learn.Learner
}

func newBuilder(ctx context.Context, opts Options) *builder {
	return &builder{ctx: ctx, opts: opts}
}

// ── Phase 1: loadConfig ──────────────────────────────────────────────────────
// Loads and validates configuration, resolves the model entry, sets up the
// event sink, runs legacy migrations, and resolves the network proxy.

func (b *builder) loadConfig() error {
	b.stderr = b.opts.Stderr
	if b.stderr == nil {
		b.stderr = os.Stderr
	}

	b.root = resolveWorkspaceRoot(b.opts.WorkspaceRoot)
	migrated, migErr := config.MigrateLegacyIfNeededForRoot(b.root)
	cfg, err := config.LoadForRoot(b.root)
	if err != nil {
		return err
	}
	b.cfg = cfg
	agentlog.Init(cfg.AgentLog)

	modelName := b.opts.Model
	if modelName == "" {
		modelName = cfg.DefaultModel
	}
	config.NormalizeLegacyMimoCustomProvidersForRefs(cfg, modelName)

	tokenMode := NormalizeTokenMode(b.opts.TokenMode)
	b.tokenEconomy = tokenMode == TokenModeEconomy
	b.keepPolicy = agentKeepPolicy(cfg.Agent.Keep)

	entry, ok := cfg.ResolveModel(modelName)
	if !ok {
		return fmt.Errorf("%w %q (configured: %s); note: defining [[providers]] replaces the built-in presets, so add a [[providers]] entry for it or use a configured name, or run `reasonix setup` to reconfigure", ErrUnknownModel, modelName, providerNames(cfg))
	}
	b.entry = entry
	b.modelRef = entry.Name + "/" + entry.Model
	slog.Info("boot.model", "ref", b.modelRef, "provider", entry.Kind)

	if b.opts.EffortOverride != nil {
		entry.Effort = *b.opts.EffortOverride
		if entry.Kind == "anthropic" && strings.TrimSpace(entry.Effort) != "" && strings.TrimSpace(entry.Thinking) == "" {
			entry.Thinking = "adaptive"
		}
	}
	if b.opts.RequireKey {
		if err := cfg.Validate(modelName); err != nil {
			return err
		}
	}

	b.entryPrice = applyExchangeRate(entry.Price, cfg)
	b.sink = event.Sync(b.opts.Sink)

	if migErr != nil {
		b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: "config migration from ~/.reasonix failed: " + migErr.Error()})
	} else if migrated != nil {
		b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: migrated.Notice()})
	}
	migration.MigrateLegacyMemorySources(b.sink)
	migration.MigrateLegacySessionSources(b.sink)

	if !b.opts.RequireKey && entry.RequiresAPIKey() && entry.APIKey() == "" {
		b.sink.Emit(event.Event{Kind: event.Notice, Text: fmt.Sprintf("model %q is selected but its API key %s is not set — requests will fail until you set it", modelName, entry.APIKeyEnv)})
	}

	b.jm = jobs.NewManager(b.sink, jobs.WithStalledWarningAfter(time.Duration(cfg.BackgroundJobStalledWarningSeconds())*time.Second))

	b.sessionDir = b.opts.SessionDir
	if b.sessionDir == "" {
		b.sessionDir = config.SessionDir()
	}

	reconcileCleanupPending := b.opts.CleanupPendingReconciler
	if reconcileCleanupPending == nil {
		reconcileCleanupPending = control.ReconcileCleanupPending
	}
	if err := reconcileCleanupPending(b.sessionDir); err != nil {
		b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: "cleanup-pending reconciliation failed: " + err.Error()})
	}

	proxySpec := cfg.NetworkProxySpec()
	if err := netclient.Validate(proxySpec); err != nil {
		return err
	}
	balanceClient, err := netclient.NewHTTPClient(proxySpec, netclient.TransportOptions{})
	if err != nil {
		return err
	}
	b.proxySpec = proxySpec
	b.balanceClient = balanceClient

	b.maxSteps = cfg.Agent.MaxSteps
	if b.opts.MaxSteps > 0 {
		b.maxSteps = b.opts.MaxSteps
	}

	return nil
}

// ── Phase 2: buildProviders ──────────────────────────────────────────────────
// Constructs the main execution provider and optional auxiliary providers for
// compaction summarization, vision, and web extraction.

func (b *builder) buildProviders() error {
	execProv, err := NewProviderWithProxy(b.entry, b.proxySpec)
	if err != nil {
		return err
	}
	b.execProv = execProv
	b.compressionProv, b.visionProv, b.webExtractProv = resolveAuxProviders(b.cfg, b.proxySpec, b.sink)
	return nil
}

// ── Phase 3: buildPrompt ─────────────────────────────────────────────────────
// Composes the cache-stable system prompt by layering base config, output style,
// user-decision policy, language policy, persistent memory, and skill index.
// Discovers project and global skills for the skill store.

func (b *builder) buildPrompt() error {
	cfg := b.cfg
	root := b.root

	sysPrompt, err := cfg.ResolveSystemPromptForRoot(root)
	if err != nil {
		return err
	}
	if st, ok := outputstyle.Resolve(cfg.Agent.OutputStyle, outputstyle.Dirs()); ok {
		sysPrompt = outputstyle.Apply(sysPrompt, st)
	}
	sysPrompt += "\n\n" + config.UserDecisionPolicy
	sysPrompt += "\n\n" + languagePolicy(cfg.Language)
	if b.tokenEconomy {
		sysPrompt += "\n\n" + tokenEconomyPrompt
	}

	mem := memory.Load(memory.Options{CWD: root, UserDir: config.MemoryUserDir()})
	b.projectChecks = instruction.ExtractHostChecks(mem.Docs)
	sysPrompt = memory.Compose(sysPrompt, mem)
	b.mem = mem

	skillStore := skill.New(skill.Options{
		ProjectRoot:   root,
		CustomPaths:   cfg.SkillCustomPaths(),
		ExcludedPaths: cfg.SkillExcludedPaths(),
		DisabledNames: cfg.DisabledSkillNames(),
		MaxDepth:      cfg.SkillMaxDepth(),
		Stderr:        b.opts.Stderr,
	})
	skills := skillStore.List()
	allSkillStore := skill.New(skill.Options{
		ProjectRoot:   root,
		CustomPaths:   cfg.SkillCustomPaths(),
		ExcludedPaths: cfg.SkillExcludedPaths(),
		MaxDepth:      cfg.SkillMaxDepth(),
		Stderr:        io.Discard,
	})
	allSkills := allSkillStore.List()
	if !b.tokenEconomy {
		sysPrompt = skill.ApplyIndex(sysPrompt, skills)
	}

	b.sysPrompt = sysPrompt
	b.skillStore = skillStore
	b.skills = skills
	b.allSkillStore = allSkillStore
	b.allSkills = allSkills
	return nil
}

// ── Phase 4: buildToolRegistry ───────────────────────────────────────────────
// Creates the tool registry and registers all enabled built-in tools with their
// workspace-specific sandbox/search/proxy confinement.

func (b *builder) buildToolRegistry() {
	cfg := b.cfg
	root := b.root

	b.reg = tool.NewRegistry()

	bashSpec := sandbox.Spec{
		Mode:        cfg.BashMode(),
		WriteRoots:  cfg.WriteRootsForRoot(root),
		Network:     cfg.Sandbox.Network,
		RemoteURL:   cfg.Sandbox.RemoteSandboxURL,
		RemoteToken: cfg.Sandbox.RemoteSandboxToken,
	}
	shell := sandbox.ResolveShell(cfg.Tools.Shell.Prefer, cfg.Tools.Shell.Path, b.stderr)
	bashSpec.Shell = shell
	if bashSpec.Mode == "enforce" && !sandbox.Available() {
		fmt.Fprintln(b.stderr, "warning: bash sandbox requested but unavailable on this platform; running bash unconfined")
	} else if bashSpec.Mode == "remote" && bashSpec.RemoteURL == "" {
		fmt.Fprintln(b.stderr, "warning: remote sandbox mode requested but remote_sandbox_url is not set; running bash unconfined")
	}
	if autoShellPrefer(cfg.Tools.Shell.Prefer) && shell.Kind == sandbox.ShellPowerShell {
		fmt.Fprintln(b.stderr, "warning: bash not found on PATH; the shell tool will run commands under Windows PowerShell. Install Git for Windows or WSL to use bash, or set [tools.shell] prefer=\"powershell\" to silence this.")
	}

	searchSpec := builtin.ResolveSearch(cfg.Tools.Search.Engine, cfg.Tools.Search.RgPath, b.stderr)
	bashTimeout := time.Duration(cfg.BashTimeoutSeconds()) * time.Second

	enabledBuiltins := cfg.Tools.Enabled
	if b.tokenEconomy {
		enabledBuiltins = tokenEconomyBuiltins(enabledBuiltins)
	}
	addBuiltins(b.reg, enabledBuiltins, cfg.WriteRootsForRoot(root), bashSpec, bashTimeout, searchSpec, b.stderr, root, b.proxySpec)

	b.bashSpec = bashSpec
	b.bashTimeout = bashTimeout
	b.searchSpec = searchSpec
	b.shell = shell
}

// ── Phase 5: buildPlugins ────────────────────────────────────────────────────
// Connects MCP plugin servers: eager (blocking), background (lazy/placeholder),
// and registers LSP tools when enabled.

func (b *builder) buildPlugins() error {
	cfg := b.cfg
	root := b.root
	ctx := b.ctx
	opts := b.opts

	pluginHost := opts.SharedHost
	if pluginHost == nil {
		pluginHost = plugin.NewHost()
	}

	autoStartEntries := cfg.AutoStartPlugins()
	eagerEntries, bgEntries := partitionByTier(autoStartEntries)
	extraSpecs := applyKnownPluginOverrides(opts.ExtraPlugins, root)
	onDemandMCPSpecs := map[string]plugin.Spec{}
	onDemandMCPNames := []string{}
	if b.tokenEconomy {
		for _, spec := range append(PluginSpecsForRoot(autoStartEntries, root), extraSpecs...) {
			name := strings.TrimSpace(spec.Name)
			if name == "" {
				continue
			}
			if _, exists := onDemandMCPSpecs[name]; !exists {
				onDemandMCPNames = append(onDemandMCPNames, name)
			}
			onDemandMCPSpecs[name] = spec
		}
		eagerEntries, bgEntries = nil, nil
	}

	// Auto-demote chronically-slow eager plugins to background.
	var demoteMessages []string
	budget := plugin.DefaultStartupBudget()
	kept := eagerEntries[:0]
	for _, e := range eagerEntries {
		rec := plugin.Recommend(e.Name, budget, 0)
		if rec.Demote {
			demoteMessages = append(demoteMessages, rec.Reason)
			bgEntries = append(bgEntries, e)
			continue
		}
		kept = append(kept, e)
	}
	eagerEntries = kept

	eagerSpecs := PluginSpecsForRoot(eagerEntries, root)
	bgSpecs := PluginSpecsForRoot(bgEntries, root)
	if !b.tokenEconomy {
		eagerSpecs = append(eagerSpecs, extraSpecs...)
	}
	if opts.Stderr != nil {
		for i := range eagerSpecs {
			eagerSpecs[i].Stderr = opts.Stderr
		}
		for i := range bgSpecs {
			bgSpecs[i].Stderr = opts.Stderr
		}
	}

	if len(eagerSpecs) > 0 {
		if opts.SharedHost != nil {
			for _, s := range eagerSpecs {
				if pluginHost.HasClient(s.Name) {
					tools, err := pluginHost.ToolsFor(ctx, s.Name)
					if err == nil {
						for _, t := range tools {
							b.reg.Add(t)
						}
						continue
					}
				}
				addCtx, addCancel := context.WithTimeout(ctx, 5*time.Second)
				tools, err := pluginHost.Add(addCtx, s)
				addCancel()
				if err != nil {
					if plugin.IsServerAlreadyConnected(err) || errors.Is(err, plugin.ErrSpawningInFlight) {
						tools, err2 := pluginHost.ToolsFor(ctx, s.Name)
						if err2 == nil {
							for _, t := range tools {
								b.reg.Add(t)
							}
							continue
						}
					}
					b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn,
						Text: fmt.Sprintf("mcp %s: %v", s.Name, err)})
					continue
				}
				for _, t := range tools {
					b.reg.Add(t)
				}
			}
		} else {
			host, ptools := plugin.StartAvailable(ctx, eagerSpecs)
			pluginHost = host
			for _, t := range ptools {
				b.reg.Add(t)
			}
			go host.StartPhaseB(ctx, b.sink)
			if text, ok := MCPStartupNotice(host.Failures()); ok {
				b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: text})
			}
		}
	}

	// Background: register placeholder tools now, spawn asynchronously.
	for _, s := range bgSpecs {
		if pluginHost.HasClient(s.Name) {
			tools, err := pluginHost.ToolsFor(ctx, s.Name)
			if err == nil {
				for _, t := range tools {
					b.reg.Add(t)
				}
				continue
			}
		}
		cs, _ := plugin.LoadCachedSchema(s.Name, plugin.SpecFingerprint(s))
		for _, t := range plugin.LazyToolset(s, cs, pluginHost, b.reg, ctx, true) {
			b.reg.Add(t)
		}
	}

	for _, msg := range demoteMessages {
		b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo, Text: msg})
	}

	cleanup := pluginHost.Close
	if opts.SharedHost != nil {
		cleanup = func() {}
	}
	b.pluginHost = pluginHost
	b.cleanup = cleanup
	b.onDemandMCPSpecs = onDemandMCPSpecs
	b.onDemandMCPNames = onDemandMCPNames

	// LSP tools.
	if cfg.LSP.Enabled {
		lspMgr := lsp.NewManager(root, LSPSpecs(cfg.LSP))
		b.lspMgr = lspMgr
		if !b.tokenEconomy {
			addTools(b.reg, lsp.Tools(lspMgr))
		}
		prev := cleanup
		b.cleanup = func() { prev(); lspMgr.Close() }
	}

	return nil
}

// ── Phase 6: buildPermissions ─────────────────────────────────────────────────
// Constructs the permission policy, headless gate, and hook runner.

func (b *builder) buildPermissions() {
	cfg := b.cfg
	root := b.root

	b.policy = permission.New(cfg.Permissions.Mode, cfg.Permissions.Allow, cfg.Permissions.Ask, cfg.Permissions.Deny)
	b.headlessGate = permission.NewGate(b.policy, nil)


	hooksTrusted := hook.IsTrusted(root, "")
	b.hookRunner = hook.NewRunner(
		hook.Load(hook.LoadOptions{ProjectRoot: root, Trusted: hooksTrusted}),
		root, nil,
		func(msg string) { b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelWarn, Text: msg}) },
	)
	if hook.ProjectDefinesHooks(root) && !hooksTrusted {
		b.sink.Emit(event.Event{Kind: event.Notice, Level: event.LevelInfo,
			Text: "this project defines hooks but they are not trusted — run /hooks trust to enable them"})
	}
}

// ── Phase 7: buildSubagents ──────────────────────────────────────────────────
// Initialises the subagent transcript store (cleans up any stale running entries).

func (b *builder) buildSubagents() error {
	store, err := newSubagentStore(b.sessionDir)
	if err != nil {
		return err
	}
	if store != nil {
		store.WithDestroyedChecker(b.jm.IsDestroying)
	}
	b.subagentStore = store
	return nil
}

// ── Phase 8: buildToolSurface ────────────────────────────────────────────────
// Wires the higher-level tools: task/parallel-task, memory retrieval, skill
// runner, custom slash commands, install_source, and the economy connector.
// This phase is closure-heavy because these tools reference each other and the
// agent's session state; the closures remain self-contained within the method.

func (b *builder) buildToolSurface() {
	cfg := b.cfg
	root := b.root
	ctx := b.ctx

	// resolveSubagentProvider returns a provider for the given model+effort,
	// cloning from the current entry if no override is given.
	resolveSubagentProvider := func(modelRef, effort string) (provider.Provider, *provider.Pricing, int, error) {
		me := *b.entry
		if strings.TrimSpace(modelRef) != "" {
			resolved, ok := cfg.ResolveModel(modelRef)
			if !ok {
				return nil, nil, 0, fmt.Errorf("unknown model %q", modelRef)
			}
			me = *resolved
		}
		if strings.TrimSpace(effort) != "" {
			normalized, err := config.NormalizeEffort(&me, effort)
			if err != nil {
				return nil, nil, 0, err
			}
			me.Effort = normalized
			if me.Kind == "anthropic" && strings.TrimSpace(me.Effort) != "" && strings.TrimSpace(me.Thinking) == "" {
				me.Thinking = "adaptive"
			}
		}
		p, err := NewProviderWithProxy(&me, b.proxySpec)
		if err != nil {
			return nil, nil, 0, err
		}
		return p, me.Price, me.ContextWindow, nil
	}
	subagentIdentity := func(modelRef, effort string) (string, string) {
		return subagentEffectiveIdentity(cfg, b.modelRef, b.entry, modelRef, effort)
	}

	// task tool
	taskModel := firstNonEmpty(cfg.Agent.SubagentModels["task"], cfg.Agent.SubagentModel)
	taskEffort := firstNonEmpty(cfg.Agent.SubagentEfforts["task"], cfg.Agent.SubagentEffort)
	taskToolAdded := false
	addTaskTool := func() string {
		if taskToolAdded {
			return "task tool is already enabled."
		}
		taskToolAdded = true
		tt := agent.NewTaskTool(b.execProv, b.entry.Price, b.reg, b.maxSteps,
			b.entry.ContextWindow, cfg.Agent.RecentKeep, cfg.Agent.SoftCompactRatio, cfg.Agent.CompactRatio, cfg.Agent.CompactForceRatio,
			cfg.Agent.Temperature, config.ArchiveDir(), "", b.headlessGate,
			b.keepPolicy,
			taskModel, taskEffort, resolveSubagentProvider).
			WithTranscripts(b.subagentStore, root, b.modelRef, b.entry.Effort).
			WithTranscriptIdentityResolver(subagentIdentity)
		b.reg.Add(tt)
		b.reg.Add(agent.NewParallelTasksTool(tt, b.reg))
		return "enabled task."
	}
	if !b.tokenEconomy {
		addTaskTool()
	}

	// Memory and session tools
	b.reg.Add(history.NewTool(history.Options{SessionDir: b.sessionDir, GlobalSessionDir: config.SessionDir(), ArchiveDir: config.ArchiveDir()}))
	b.reg.Add(sessiontool.NewListSessionsTool(b.sessionDir))
	b.reg.Add(sessiontool.NewReadSessionTool(b.sessionDir))
	b.reg.Add(memory.NewRecallTool(b.mem.Store))
	b.reg.Add(memory.NewRememberTool(b.mem.Store))
	b.reg.Add(memory.NewForgetTool(b.mem.Store))
	b.reg.Add(agent.NewAskTool())

	// Skill runner closure
	skillRunner := func(sctx context.Context, sk skill.Skill, task string, runOpts skill.SubagentRunOptions) (string, error) {
		sk = skill.WithCodeGraphTools(sk, skill.CodeGraphReadTools(b.reg))
		prov, price, ctxWin := b.execProv, b.entry.Price, b.entry.ContextWindow
		modelRef := subagentModelRef(cfg, sk)
		effortRef := subagentEffortRef(cfg, sk)
		if modelRef != "" || effortRef != "" {
			p, pr, cw, err := resolveSubagentProvider(modelRef, effortRef)
			if err != nil {
				return "", fmt.Errorf("subagent skill %q profile: %w", sk.Name, err)
			}
			prov, price, ctxWin = p, pr, cw
		}
		subReg := agent.SubagentToolRegistry(b.reg, sk.AllowedTools)
		continueFrom, forkFrom := strings.TrimSpace(runOpts.ContinueFrom), strings.TrimSpace(runOpts.ForkFrom)
		if continueFrom != "" && forkFrom != "" {
			return "", fmt.Errorf("continue_from and fork_from are mutually exclusive")
		}
		parentID, _, _, _ := agent.CallContext(sctx)
		parentSession := agent.ParentSession(sctx)
		var run *agent.SubagentRun
		if b.subagentStore == nil || parentSession == "" {
			if continueFrom != "" || forkFrom != "" {
				return "", fmt.Errorf("continue_from/fork_from require a persisted session; none is active in this run")
			}
			run = agent.EphemeralSubagentRun(sk.Body)
		} else {
			identityModel, identityEffort := subagentIdentity(modelRef, effortRef)
			spec := agent.SubagentSpec{
				Kind:             "skill",
				Name:             sk.Name,
				WorkspaceRoot:    root,
				ParentSession:    parentSession,
				ParentToolCallID: parentID,
				SystemPrompt:     sk.Body,
				Registry:         subReg,
				Model:            identityModel,
				Effort:           identityEffort,
			}
			var prepErr error
			switch {
			case continueFrom != "":
				run, prepErr = b.subagentStore.PrepareContinue(continueFrom, spec)
			case forkFrom != "":
				run, prepErr = b.subagentStore.PrepareFork(forkFrom, spec)
			default:
				run, prepErr = b.subagentStore.PrepareFresh(spec)
			}
			if prepErr != nil {
				return "", prepErr
			}
		}
		defer run.Release()
		steps := b.maxSteps
		if steps > 0 {
			if steps /= 2; steps < 5 {
				steps = 5
			}
		}
		answer, err := agent.RunSubAgentWithSession(sctx, prov, subReg, run.Session, task, agent.Options{
			MaxSteps:          steps,
			Temperature:       cfg.Agent.Temperature,
			Pricing:           price,
			UsageSource:       event.UsageSourceSubagent,
			Gate:              b.headlessGate,
			ContextWindow:     ctxWin,
			RecentKeep:        cfg.Agent.RecentKeep,
			ArchiveDir:        config.ArchiveDir(),
			KeepPolicy:        b.keepPolicy,
			ReasoningLanguage: agent.ReasoningLanguageFromContext(sctx),
		}, agent.NestedSink(sctx, event.Discard))
		if err != nil {
			return "", errors.Join(err, b.subagentStore.SaveFailed(run))
		}
		if err := b.subagentStore.SaveCompleted(run); err != nil {
			return "", errors.Join(err, b.subagentStore.SaveFailed(run))
		}
		return agent.FormatSubagentResult(answer, run.Ref, false), nil
	}
	skillProfile := func(sk skill.Skill) *event.Profile {
		model, effort := subagentModelRef(cfg, sk), subagentEffortRef(cfg, sk)
		if model == "" && effort == "" {
			return nil
		}
		return &event.Profile{Model: model, Effort: effort}
	}

	// Custom slash commands
	b.cmds, _ = command.Load(config.CommandDirsForRoot(root)...)
	addSlashCommandTool := func(includeSkills bool) {
		var slashEntries []command.SlashEntry
		if includeSkills {
			for _, sk := range b.skills {
				sk := sk
				slashEntries = append(slashEntries, command.SlashEntry{
					Name:        sk.Name,
					Description: sk.Description,
					Render:      func(args []string) string { return skill.Render(sk, strings.Join(args, " ")) },
				})
			}
		}
		for _, cmd := range b.cmds {
			cmd := cmd
			slashEntries = append(slashEntries, command.SlashEntry{
				Name:        cmd.Name,
				Description: cmd.Description,
				ArgHint:     cmd.ArgHint,
				Render:      func(args []string) string { return cmd.Render(args) },
			})
		}
		b.reg.Add(command.NewSlashCommandTool(slashEntries))
	}

	// install_source tool
	installSourceAdded := false
	addInstallSourceTool := func() string {
		if installSourceAdded {
			return "install_source is already enabled."
		}
		installSourceAdded = true
		b.reg.Add(installsource.NewTool(installsource.Options{
			ProjectRoot: root,
			HTTPClient:  b.balanceClient,
			ConnectMCP: func(e config.PluginEntry) (installsource.MCPConnectResult, error) {
				spec := pluginSpecFromEntry(e, root)
				if b.opts.Stderr != nil {
					spec.Stderr = b.opts.Stderr
				}
				tools, err := b.pluginHost.Add(ctx, spec)
				if err != nil {
					return installsource.MCPConnectResult{}, err
				}
				b.reg.RemovePrefix(plugin.ToolPrefix(spec.Name))
				for _, t := range tools {
					b.reg.Add(t)
				}
				disconnect := func() {
					if prefix, ok := b.pluginHost.Remove(spec.Name); ok {
						b.reg.RemovePrefix(prefix)
					}
				}
				return installsource.MCPConnectResult{
					ToolCount:  len(tools),
					Disconnect: disconnect,
				}, nil
			},
			OnDisconnect: func(serverName string) bool {
				if prefix, ok := b.pluginHost.Remove(serverName); ok {
					b.reg.RemovePrefix(prefix)
					return true
				}
				return false
			},
		}))
		return "enabled install_source."
	}

	// Skill tools
	lspToolsAdded := false
	addLSPTools := func() []string {
		if b.lspMgr == nil || lspToolsAdded {
			return nil
		}
		lspToolsAdded = true
		return addTools(b.reg, lsp.Tools(b.lspMgr))
	}

	skillToolsAdded := false
	addSkillTools := func() string {
		if skillToolsAdded {
			return "skills are already enabled.\n\n" + skill.IndexBlock(b.skills)
		}
		skillToolsAdded = true
		b.reg.Add(skill.NewRunSkillTool(b.skillStore, skillRunner, skillProfile))
		b.reg.Add(skill.NewReadSkillTool(b.skillStore))
		b.reg.Add(skill.NewInstallSkillTool(b.skillStore, nil))
		for _, t := range skill.BuiltinSubagentTools(b.skillStore, skillRunner, skillProfile) {
			b.reg.Add(t)
		}
		addSlashCommandTool(true)
		return "enabled skills. Use run_skill/read_skill or the dedicated skill tools on the next model request.\n\n" + skill.IndexBlock(b.skills)
	}

	if b.tokenEconomy {
		addSlashCommandTool(false)
	} else {
		addInstallSourceTool()
		addSkillTools()
	}

	if b.tokenEconomy {
		b.reg.Add(&toolSourceConnector{
			skills: func(context.Context) (string, error) {
				return addSkillTools(), nil
			},
			task: func(context.Context) (string, error) {
				return addTaskTool(), nil
			},
			install: func(context.Context) (string, error) {
				return addInstallSourceTool(), nil
			},
			webFetch: func(context.Context) (string, error) {
				if !builtinToolEnabled(cfg.Tools.Enabled, "web_fetch") {
					return "web_fetch is disabled by [tools].enabled.", nil
				}
				names := addTools(b.reg, builtin.Workspace{
					Dir:         root,
					WriteRoots:  cfg.WriteRootsForRoot(root),
					Bash:        b.bashSpec,
					BashTimeout: b.bashTimeout,
					Search:      b.searchSpec,
					ProxySpec:   b.proxySpec,
				}.Tools("web_fetch"))
				if len(names) == 0 {
					return "web_fetch is already enabled or unavailable.", nil
				}
				return "enabled " + strings.Join(names, ", ") + ".", nil
			},
			lsp: func(context.Context) (string, error) {
				if b.lspMgr == nil {
					return "", fmt.Errorf("LSP is disabled in config")
				}
				names := addLSPTools()
				if len(names) == 0 {
					return "LSP tools are already enabled.", nil
				}
				return "enabled " + strings.Join(names, ", ") + ".", nil
			},
			mcp: func(_ context.Context, name string) (string, error) {
				spec, ok := b.onDemandMCPSpecs[name]
				if !ok {
					return "", fmt.Errorf("no configured MCP server named %q", name)
				}
				if b.opts.Stderr != nil {
					spec.Stderr = b.opts.Stderr
				}
				tools, err := b.pluginHost.Add(ctx, spec)
				if err != nil {
					if plugin.IsServerAlreadyConnected(err) || errors.Is(err, plugin.ErrSpawningInFlight) {
						tools, err2 := b.pluginHost.ToolsFor(ctx, spec.Name)
						if err2 != nil {
							return "", err2
						}
						b.reg.RemovePrefix(plugin.ToolPrefix(spec.Name))
						names := addTools(b.reg, tools)
						if len(names) == 0 {
							return fmt.Sprintf("MCP server %q connected but exposed no tools.", spec.Name), nil
						}
						return fmt.Sprintf("enabled MCP server %q tools: %s.", spec.Name, strings.Join(names, ", ")), nil
					}
					return "", err
				}
				b.reg.RemovePrefix(plugin.ToolPrefix(spec.Name))
				names := addTools(b.reg, tools)
				if len(names) == 0 {
					return fmt.Sprintf("MCP server %q connected but exposed no tools.", spec.Name), nil
				}
				return fmt.Sprintf("enabled MCP server %q tools: %s.", spec.Name, strings.Join(names, ", ")), nil
			},
			mcpNames: b.onDemandMCPNames,
		})
	}
}

// ── Phase 9: buildExecutor ───────────────────────────────────────────────────
// Assembles the main agent executor and, when configured, wraps it in a
// two-model Coordinator and/or a ProviderAutoPlanClassifier.

func (b *builder) buildExecutor() error {
	cfg := b.cfg

	execSess := agent.NewSession(b.sysPrompt)
	b.executor = agent.New(b.execProv, b.reg, execSess, agent.Options{
		MaxSteps:             b.maxSteps,
		Temperature:          cfg.Agent.Temperature,
		Pricing:              b.entryPrice,
		Gate:                 b.headlessGate,
		Hooks:                b.hookRunner,
		Jobs:                 b.jm,
		ProjectChecks:        b.projectChecks,
		ContextWindow:        b.entry.ContextWindow,
		SoftCompactRatio:     cfg.Agent.SoftCompactRatio,
		CompactRatio:         cfg.Agent.CompactRatio,
		CompactForceRatio:    cfg.Agent.CompactForceRatio,
		RecentKeep:           cfg.Agent.RecentKeep,
		ArchiveDir:           config.ArchiveDir(),
		KeepPolicy:           b.keepPolicy,
		ReasoningLanguage:    cfg.ReasoningLanguage(),
		CompressionProv:      b.compressionProv,
		VisionProv:           b.visionProv,
		WebExtractProv:       b.webExtractProv,
		PlanModeAllowedTools: cfg.Agent.PlanModeAllowedTools,
		CompressToolOutput:   cfg.CompressToolOutputEnabled(),
		WorkshopThreshold:    workshopThreshold,
		Workshop:             workshopSynthesizer(b.jm, b.execProv, b.reg, b.entry, cfg.Agent),
	}, b.sink)

	b.runner = b.executor
	b.label = b.entry.Model

	// Optional auto-plan classifier.
	if !b.tokenEconomy && !strings.EqualFold(strings.TrimSpace(cfg.Agent.AutoPlan), "off") && cfg.Agent.AutoPlanClassifier != "" {
		cm := cfg.Agent.AutoPlanClassifier
		ce, ok := cfg.ResolveModel(cm)
		if !ok {
			return fmt.Errorf("auto_plan_classifier %q is not a configured provider", cm)
		}
		classifierProv, err := NewProviderWithProxy(ce, b.proxySpec)
		if err != nil {
			return fmt.Errorf("auto_plan_classifier %q: %w", cm, err)
		}
		b.classifier = control.NewBillableProviderAutoPlanClassifier(classifierProv, ce.Price, b.sink)
	}

	// Optional two-model planner/coordinator.
	if pm := cfg.Agent.PlannerModel; pm != "" && !b.tokenEconomy {
		pe, ok := cfg.ResolveModel(pm)
		if !ok {
			return fmt.Errorf("planner_model %q is not a configured provider", pm)
		}
		if pe.Model != b.entry.Model {
			plannerProv, err := NewProviderWithProxy(pe, b.proxySpec)
			if err != nil {
				return fmt.Errorf("planner %q: %w", pm, err)
			}
			plannerSess := agent.NewSession(agent.PlannerPromptWithContext(b.mem.Block()))
			plannerTools := agent.PlannerToolRegistry(b.reg)
			b.runner = agent.NewCoordinator(plannerProv, plannerSess, pe.Price, plannerTools, agent.Options{
				MaxSteps:          cfg.Agent.PlannerMaxSteps,
				MaxStepsKey:       "agent.planner_max_steps",
				Gate:              b.headlessGate,
				ContextWindow:     pe.ContextWindow,
				SoftCompactRatio:  cfg.Agent.SoftCompactRatio,
				CompactRatio:      cfg.Agent.CompactRatio,
				CompactForceRatio: cfg.Agent.CompactForceRatio,
				RecentKeep:        cfg.Agent.RecentKeep,
				ArchiveDir:        config.ArchiveDir(),
				KeepPolicy:        b.keepPolicy,
				ReasoningLanguage: cfg.ReasoningLanguage(),
			}, b.executor, cfg.Agent.Temperature, b.sink, control.NewPlannerGate(b.classifier))
			b.label = b.entry.Model + " + planner " + pe.Model
		}
	}

	return nil
}

// ── Phase 10: buildLearner ───────────────────────────────────────────────────
// Creates the self-improving learner when [learn] is enabled.

func (b *builder) buildLearner() {
	cfg := b.cfg
	if cfg.Learn.Enabled {
		b.lc = learn.New(learn.Config{
			Enabled:         cfg.Learn.Enabled,
			MaxPatterns:     cfg.Learn.MaxPatterns,
			MinConfidence:   cfg.Learn.MinConfidence,
			MaxObservations: cfg.Learn.MaxObservations,
		})
	}
}

// ── Phase 11: assemble ───────────────────────────────────────────────────────
// Assembles the final Controller from all phase outputs, wires mesh and learner,
// and returns it ready to drive.

func (b *builder) assemble() *control.Controller {
	cfg := b.cfg

	ctrlOpts := control.Options{
		Runner:                 b.runner,
		Executor:               b.executor,
		Sink:                   b.sink,
		Policy:                 b.policy,
		Label:                  b.label,
		ModelRef:               b.modelRef,
		SystemPrompt:           b.sysPrompt,
		SessionDir:             b.sessionDir,
		Host:                   b.pluginHost,
		Commands:               b.cmds,
		Skills:                 b.skills,
		AllSkills:              b.allSkills,
		SkillStore:             b.skillStore,
		AllSkillStore:          b.allSkillStore,
		Hooks:                  b.hookRunner,
		Memory:                 b.mem,
		Cleanup:                b.cleanup,
		BalanceURL:             b.entry.BalanceURL,
		BalanceKey:             b.entry.APIKey(),
		BalanceClient:          b.balanceClient,
		Jobs:                   b.jm,
		Registry:               b.reg,
		PluginCtx:              b.ctx,
		WorkspaceRoot:          b.root,
		AutoPlan:               cfg.Agent.AutoPlan,
		ReasoningLanguage:      cfg.ReasoningLanguage(),
		DisableColdResumePrune: !cfg.ColdResumePruneEnabled(),
		Shell:                  b.shell,
		PlanModeAllowedTools:   cfg.Agent.PlanModeAllowedTools,
		ApprovalTimeout:        b.opts.ApprovalTimeout,
		Learner:                b.lc,
		OnRemember: func(rule string) control.RememberResult {
			return rememberPermissionRule(b.root, rule)
		},
	}
	if b.classifier != nil {
		ctrlOpts.Classifier = b.classifier
	}

	if len(cfg.Schedule.Tasks) > 0 {
		schedCfg := scheduler.Config{}
		for _, t := range cfg.Schedule.Tasks {
			schedCfg.Tasks = append(schedCfg.Tasks, scheduler.Task{
				Name:    t.Name,
				Cron:    t.Cron,
				Prompt:  t.Prompt,
				Model:   t.Model,
				Enabled: t.Enabled,
			})
		}
		ctrlOpts.ScheduleConfig = &schedCfg
	}

	ctrl := control.New(ctrlOpts)

	if cfg.Mesh.Enabled && len(cfg.Mesh.Peers) > 0 {
		var meshPeers []mesh.PeerConfig
		for _, p := range cfg.Mesh.Peers {
			meshPeers = append(meshPeers, mesh.PeerConfig{
				Name:     p.Name,
				URL:      p.URL,
				TokenEnv: p.TokenEnv,
				Enabled:  p.Enabled,
			})
		}
		if m := mesh.New(mesh.Config{Enabled: true, Peers: meshPeers}); m != nil {
			ctrl.SetMesh(m)
			b.reg.Add(builtin.ConfineCouncil(m))
		}
	}

	if b.lc != nil {
		ctrl.SetLearner(b.lc)
		slog.Info("boot.learner", "enabled", true)
	}

	return ctrl
}

