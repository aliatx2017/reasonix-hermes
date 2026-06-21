import { lazy, Suspense, useCallback, useEffect, useId, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent, type MouseEvent as ReactMouseEvent, type PointerEvent, type ReactNode } from "react";
import { Check, ChevronDown, ChevronUp, GripVertical, Play, RefreshCw } from "lucide-react";
import { asArray } from "../lib/array";
import { useDeferredClose } from "../lib/useMountTransition";
import { app } from "../lib/bridge";
import { normalizeLangPref, useI18n, useT, type DictKey, type LangPref } from "../lib/i18n";
import { providerIsConfigured, providerRequiresKey } from "../lib/providerModels";
import { useUpdater } from "../lib/useUpdater";
import {
  THEME_STYLES,
  applyTheme,
  getTheme,
  getThemeStyle,
  normalizeThemePreference,
  normalizeThemeStyleForTheme,
  type Theme,
  type ThemeStyle,
} from "../lib/theme";
import { TEXT_SIZES, applyTextSize, getTextSize, type TextSize } from "../lib/textSize";
import {
  applyFontFamily,
  applyMonoFontFamily,
  getFontFamily,
  getMonoFontFamily,
  getCustomFontName,
  getCustomMonoFontName,
  setCustomFontName,
  setCustomMonoFontName,
  type FontFamily,
  type MonoFontFamily,
} from "../lib/fontFamily";
import { getAvailableFontFamilies, getAvailableMonoFontFamilies } from "../lib/fontAvailability";
import { getDisplayMode, onDisplayModeChange, setDisplayMode as setLocalDisplayMode } from "../lib/displayMode";
import { DEFAULT_STATUS_BAR_ITEMS, normalizeStatusBarItems, type StatusBarItemId } from "../lib/statusBarItems";
import {
  comboFromKeyboardEvent,
  detectShortcutPlatform,
  formatShortcutCombo,
  onShortcutsChanged,
  resetCustomShortcuts,
  resolvedShortcutCombo,
  saveCustomShortcut,
  shortcutConflict,
  shortcutDefinitions,
  type ShortcutAction,
} from "../lib/keyboardShortcuts";
import type { HookConfigView, HooksSettingsView, NetworkView, ProviderView, SettingsTab, SettingsView } from "../lib/types";
import { Tooltip } from "./Tooltip";
import { AnchoredPopover } from "./AnchoredPopover";
import { getGenerativePreset, setGenerativePreset, generativeMusic, type GenerativePreset } from "../lib/generative-music";
import { SoundSelect } from "./SoundSelect";
import { getSuccessPreference, setSuccessPreference, getAttentionPreference, setAttentionPreference, playSuccessChime, playAttentionChime, type SoundWavPref } from "../lib/sound";
import { ModalCloseButton } from "./ModalCloseButton";
import { ShortcutComboDisplay } from "./ShortcutComboDisplay";
import { HermesSettings } from "./hermes/HermesSettings";
import { useHermesLiveData } from "./hermes/useHermesLiveData";
import { ModelsSection, settingsModelMeta } from "./ModelsSection";
import { BotsSection, botSettingsMeta, normalizeBotSettings } from "./BotsSection";
import { SettingsSection, SettingsField, ToggleSegment, PROXY_MODES, normalizeProxyMode, proxyModeLabel, type SectionProps, type SettingsInitialFocus } from "./settings-shared";
export type { SettingsInitialFocus } from "./settings-shared";

const SETTINGS_TABS: SettingsTab[] = ["general", "models", "bots", "mcp", "skills", "memory", "hooks", "shortcuts", "permissions", "sandbox", "network", "appearance", "updates", "hermes"];

const MCPServersSettingsPage = lazy(() => import("./CapabilitiesPanel").then((module) => ({ default: module.MCPServersSettingsPage })));
const SkillsSettingsPage = lazy(() => import("./CapabilitiesPanel").then((module) => ({ default: module.SkillsSettingsPage })));
const MemorySettingsPage = lazy(() => import("./MemoryPanel").then((module) => ({ default: module.MemorySettingsPage })));

// SettingsPanel is the desktop settings centre — a centred modal with left
// navigation and a right content area. It hosts all settings pages plus MCP,
// Skills, and Memory management, replacing the old per-feature drawers.
export function SettingsPanel({
  onClose,
  onChanged,
  initialTab,
  initialFocus,
  agentRunning = false,
}: {
  onClose: () => void;
  onChanged: (settings?: SettingsView | null) => void;
  initialTab?: SettingsTab;
  initialFocus?: SettingsInitialFocus;
  agentRunning?: boolean;
}) {
  const t = useT();
  const [s, setS] = useState<SettingsView | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [warning, setWarning] = useState<string | null>(null);
  const [theme, setThemeState] = useState<Theme>(getTheme());
  const [themeStyle, setThemeStyleState] = useState<ThemeStyle>(() => getThemeStyle(getTheme()));
  const [textSize, setTextSizeState] = useState<TextSize>(getTextSize());
  const [fontFamily, setFontFamilyState] = useState<FontFamily>(getFontFamily());
  const [monoFontFamily, setMonoFontFamilyState] = useState<MonoFontFamily>(getMonoFontFamily());
  const [customFontName, setCustomFontNameState] = useState<string>(getCustomFontName());
  const [customMonoFontName, setCustomMonoFontNameState] = useState<string>(getCustomMonoFontName());
  const [tab, setTab] = useState<SettingsTab>(initialTab === "providers" ? "models" : initialTab ?? "general");
  // Play the modal exit animation, then let the parent unmount us.
  const { status, requestClose } = useDeferredClose(onClose, 240);

  const reload = useCallback(async () => {
    const next = normalizeSettingsView(await app.Settings().catch((e) => { console.warn('SettingsPanel: Settings load failed', e); return null; }));
    setS(next);
    return next;
  }, []);
  useEffect(() => {
    void reload();
    if (initialTab) setTab(initialTab === "providers" ? "models" : initialTab);
  }, [initialTab, reload]);
  useEffect(() => {
    if (!s) return;
    const nextTheme = normalizeThemePreference(s.desktopTheme);
    const nextStyle = normalizeThemeStyleForTheme(s.desktopThemeStyle, nextTheme);
    setThemeState(nextTheme);
    setThemeStyleState(nextStyle);
  }, [s?.desktopTheme, s?.desktopThemeStyle]);

  // apply runs a mutation, re-reads settings, and refreshes the topbar/model.
  const apply = useCallback(async (fn: () => Promise<unknown>) => {
    setBusy(true);
    setErr(null);
    setWarning(null);
    try {
      const result = await fn();
      const next = await reload();
      onChanged(next);
      if (typeof result === "string" && result.trim()) {
        setWarning(result.trim());
      }
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
    } finally {
      setBusy(false);
    }
  }, [reload, onChanged]);
  const backgroundApply = useCallback(async (fn: () => Promise<void>) => {
    setErr(null);
    setWarning(null);
    try {
      await fn();
      const next = await reload();
      onChanged(next);
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
    }
  }, [reload, onChanged]);

  // Close on Esc
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !document.querySelector("[data-anchored-popover='active']")) requestClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [requestClose]);

  // The settings-reliant pages (general, models, network, permissions,
  // sandbox, appearance, updates) need SettingsView loaded. MCP, Skills, and Memory
  // load their own data and render regardless.
  const needsSettings = tab === "general" || tab === "models" || tab === "bots" || tab === "network" || tab === "permissions" || tab === "sandbox" || tab === "appearance" || tab === "updates" || tab === "hermes";
  const lazySettingsPageFallback = <div className="empty">{t("settings.loading")}</div>;

  return (
    <div className="management-modal-backdrop settings-modal-backdrop" data-state={status} onClick={(e) => { if (e.target === e.currentTarget) requestClose(); }}>
      <div className="management-modal settings-modal" data-state={status}>
        <header className="management-modal__head settings-modal__head">
          <div className="management-modal__title settings-modal__title">{t("settings.title")}</div>
          <ModalCloseButton label={t("common.close")} onClick={requestClose} />
        </header>

        <div className="settings-center">
          <nav className="settings-center__nav" aria-label={t("settings.title")}>
            {SETTINGS_TABS.map((id) => (
              <button
                key={id}
                className={`settings-center__navitem${tab === id ? " settings-center__navitem--active" : ""}`}
                onClick={() => setTab(id)}
              >
                <span>{settingsTabLabel(id, t)}</span>
                {s && <small>{settingsTabMeta(id, s, t)}</small>}
              </button>
            ))}
          </nav>
          <main className="settings-center__content">
            {needsSettings && err && <div className="banner banner--error">{err}</div>}
            {needsSettings && warning && <div className="banner banner--warning">{warning}</div>}
            {needsSettings && !s ? (
              <div className="empty">{t("settings.loading")}</div>
            ) : (
              <>
                {tab === "general" && s && <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}><GeneralSection s={s} busy={busy} apply={apply} agentRunning={agentRunning} /></SettingsPageShell>}
                {tab === "models" && s && <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}><ModelsSection s={s} busy={busy} apply={apply} backgroundApply={backgroundApply} /></SettingsPageShell>}
                {tab === "bots" && s && <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}><BotsSection s={s} busy={busy} apply={apply} initialFocus={initialFocus} /></SettingsPageShell>}
                {tab === "mcp" && <SettingsPageShell key={tab} s={s} tab={tab} busy={false} apply={apply}><Suspense fallback={lazySettingsPageFallback}><MCPServersSettingsPage /></Suspense></SettingsPageShell>}
                {tab === "skills" && <SettingsPageShell key={tab} s={s} tab={tab} busy={false} apply={apply}><Suspense fallback={lazySettingsPageFallback}><SkillsSettingsPage /></Suspense></SettingsPageShell>}
                {tab === "memory" && <SettingsPageShell key={tab} s={s} tab={tab} busy={false} apply={apply}><Suspense fallback={lazySettingsPageFallback}><MemorySettingsPage /></Suspense></SettingsPageShell>}
                {tab === "hooks" && <SettingsPageShell key={tab} s={s} tab={tab} busy={false} apply={apply}><HooksSection onChanged={onChanged} /></SettingsPageShell>}
                {tab === "shortcuts" && <SettingsPageShell key={tab} s={s} tab={tab} busy={false} apply={apply}><ShortcutsSection /></SettingsPageShell>}
                {tab === "permissions" && s && <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}><PermissionsSection s={s} busy={busy} apply={apply} /></SettingsPageShell>}
                {tab === "sandbox" && s && <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}><SandboxSection s={s} busy={busy} apply={apply} /></SettingsPageShell>}
                {tab === "network" && s && <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}><NetworkSection s={s} busy={busy} apply={apply} /></SettingsPageShell>}
                {tab === "appearance" && s && (
                  <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}>
                    <AppearanceSection
                      theme={theme}
                      themeStyle={themeStyle}
                      textSize={textSize}
                      fontFamily={fontFamily}
                      monoFontFamily={monoFontFamily}
                      customFontName={customFontName}
                      customMonoFontName={customMonoFontName}
                      onTheme={(nextTheme) => {
                        applyTheme(nextTheme, themeStyle, { persist: false });
                        setThemeState(nextTheme);
                        void apply(() => app.SetDesktopAppearance(nextTheme, themeStyle));
                      }}
                      onThemeStyle={(style) => {
                        applyTheme(theme, style, { persist: false });
                        setThemeStyleState(style);
                        void apply(() => app.SetDesktopAppearance(theme, style));
                      }}
                      onTextSize={(size) => {
                        applyTextSize(size);
                        setTextSizeState(size);
                      }}
                      onFontFamily={(font) => {
                        applyFontFamily(font);
                        setFontFamilyState(font);
                      }}
                      onMonoFontFamily={(font) => {
                        applyMonoFontFamily(font);
                        setMonoFontFamilyState(font);
                      }}
                      onCustomFontNameChange={(name) => {
                        setCustomFontNameState(name);
                        setCustomFontName(name);
                        applyFontFamily("custom");
                      }}
                      onCustomMonoFontNameChange={(name) => {
                        setCustomMonoFontNameState(name);
                        setCustomMonoFontName(name);
                        applyMonoFontFamily("custom");
                      }}
                    />
                  </SettingsPageShell>
                )}
                {tab === "updates" && s && (
                  <SettingsPageShell key={tab} s={s} tab={tab} busy={busy} apply={apply}>
                    <UpdatesSection
                      configPath={s.configPath}
                      checkUpdates={s.checkUpdates}
                      telemetry={s.telemetry !== false}
                      metrics={s.metrics !== false}
                      settingsBusy={busy}
                      applySettings={apply}
                    />
                  </SettingsPageShell>
                )}
                {tab === "hermes" && s && <HermesLiveSection key={tab} s={s} apply={apply} />}
              </>
            )}
          </main>
        </div>
      </div>
    </div>
  );
}

function SettingsPageShell({ s: _s, tab, children }: { s: SettingsView | null; tab: SettingsTab; busy: boolean; apply: (fn: () => Promise<unknown>) => Promise<void>; children: ReactNode }) {
  const t = useT();
  const descKey = `settings.pageDesc.${tab}` as keyof typeof import("../locales/en").en;
  const desc = t(descKey as any);
  return (
    <div className={`settings-page settings-page--${settingsPageKind(tab)} settings-page--${tab}`}>
      <div className="settings-page__header">
        <h2 className="settings-page__title">{settingsTabPageTitle(tab, t)}</h2>
        {typeof desc === "string" && desc !== `settings.pageDesc.${tab}` && <p className="settings-page__desc">{desc}</p>}
      </div>
      {children}
    </div>
  );
}

function settingsPageKind(tab: SettingsTab): "form" | "manager" {
  switch (tab) {
    case "models":
    case "mcp":
    case "skills":
    case "memory":
      return "manager";
    default:
      return "form";
  }
}

function settingsTabPageTitle(id: SettingsTab, t: ReturnType<typeof useT>): string {
  switch (id) {
    case "mcp": return t("settings.tab.mcp");
    case "skills": return t("settings.tab.skills");
    case "memory": return t("settings.tab.memory");
    case "shortcuts": return t("settings.tab.shortcuts");
    default: return settingsTabLabel(id, t);
  }
}


function settingsTabLabel(id: SettingsTab, t: ReturnType<typeof useT>): string {
  switch (id) {
    case "general":
      return t("settings.tab.general");
    case "models":
      return t("settings.tab.models");
    case "providers":
      return t("settings.tab.providers");
    case "bots":
      return t("settings.tab.bots");
    case "mcp":
      return t("settings.tab.mcp");
    case "skills":
      return t("settings.tab.skills");
    case "memory":
      return t("settings.tab.memory");
    case "hooks":
      return t("settings.tab.hooks");
    case "shortcuts":
      return t("settings.tab.shortcuts");
    case "network":
      return t("settings.tab.network");
    case "permissions":
      return t("settings.tab.permissions");
    case "sandbox":
      return t("settings.tab.sandbox");
    case "appearance":
      return t("settings.tab.appearance");
    case "updates":
      return t("settings.tab.updates");
    case "hermes":
      return "Hermes";
    default:
      return id;
  }
}

function settingsTabMeta(id: SettingsTab, s: SettingsView, t: ReturnType<typeof useT>): string {
  switch (id) {
    case "models":
      return settingsModelMeta(s, t);
    case "general":
      return `${desktopLayoutStyleLabel(normalizeDesktopLayoutStyle(s.desktopLayoutStyle), t)} · ${closeBehaviorLabel(normalizeCloseBehavior(s.closeBehavior), t)}`;
    case "providers":
      return t("settings.providerCount", { n: s.providers.length });
    case "bots":
      return botSettingsMeta(s.bot, t);
    case "mcp":
      return t("caps.connectorsTab");
    case "skills":
      return t("caps.skillsTab");
    case "memory":
      return t("settings.tabSub.memory");
    case "hooks":
      return t("settings.tabSub.hooks");
    case "shortcuts":
      return t("settings.tabSub.shortcuts");
    case "network":
      return proxyModeLabel(normalizeProxyMode(s.network.proxyMode), t);
    case "permissions":
      return permissionModeLabel(s.permissions.mode, t);
    case "sandbox":
      return sandboxModeLabel(s.sandbox.bash, t);
    case "appearance":
      return t("settings.appearanceMeta");
    case "updates":
      return t("settings.updatesMeta");
    case "hermes":
      return "Hermes dashboard";
    default:
      return id;
  }
}


function ShortcutsSection() {
  const t = useT();
  const [platform] = useState(() => detectShortcutPlatform());
  const [revision, setRevision] = useState(0);
  const [recording, setRecording] = useState<ShortcutAction | null>(null);
  const [conflict, setConflict] = useState<{ action: ShortcutAction; conflictAction: ShortcutAction } | null>(null);

  useEffect(() => onShortcutsChanged(() => setRevision((value) => value + 1)), []);

  const definitions = shortcutDefinitions();
  const commitShortcut = (action: ShortcutAction, event: ReactKeyboardEvent<HTMLButtonElement>) => {
    const combo = comboFromKeyboardEvent(event.nativeEvent);
    if (!combo) return;
    event.preventDefault();
    event.stopPropagation();
    const conflictDefinition = shortcutConflict(action, combo, platform);
    if (conflictDefinition) {
      setConflict({ action, conflictAction: conflictDefinition.action });
      return;
    }
    saveCustomShortcut(action, combo);
    setConflict(null);
    setRecording(null);
    setRevision((value) => value + 1);
  };

  return (
    <SettingsSection
      title={t("settings.shortcutsTitle")}
      description={t("settings.shortcutsHint")}
      actions={
        <button
          className="chip chip--icon"
          type="button"
          title={t("settings.shortcutsResetAll")}
          aria-label={t("settings.shortcutsResetAll")}
          onClick={() => {
            resetCustomShortcuts();
            setConflict(null);
            setRecording(null);
            setRevision((value) => value + 1);
          }}
        >
          <RefreshCw size={14} />
        </button>
      }
    >
      <div className="shortcuts-settings" data-revision={revision}>
        {conflict && (
          <div className="shortcuts-settings__conflict" role="alert">
            {t("settings.shortcutsConflict", {
              action: t(definitions.find((definition) => definition.action === conflict.action)?.labelKey ?? "settings.tab.shortcuts"),
              conflict: t(definitions.find((definition) => definition.action === conflict.conflictAction)?.labelKey ?? "settings.tab.shortcuts"),
            })}
          </div>
        )}
        {definitions.map((definition) => {
          const resolved = resolvedShortcutCombo(definition.action, platform);
          const defaultCombo = definition.defaults[platform];
          const display = formatShortcutCombo(resolved, platform);
          const isCustom = formatShortcutCombo(resolved, platform) !== formatShortcutCombo(defaultCombo, platform);
          const isRecording = recording === definition.action;
          return (
            <div className="shortcuts-settings__row" key={definition.action}>
              <div className="shortcuts-settings__copy">
                <div className="shortcuts-settings__label">{t(definition.labelKey)}</div>
                <div className="shortcuts-settings__desc">{t(definition.descriptionKey)}</div>
              </div>
              <div className="shortcuts-settings__control">
                <button
                  className={`shortcuts-settings__key${isRecording ? " shortcuts-settings__key--recording" : ""}`}
                  type="button"
                  aria-label={isRecording ? t("settings.shortcutsRecording") : display}
                  aria-pressed={isRecording}
                  onClick={() => {
                    setRecording(definition.action);
                    setConflict(null);
                  }}
                  onKeyDown={(event) => isRecording && commitShortcut(definition.action, event)}
                >
                  {isRecording ? t("settings.shortcutsRecording") : <ShortcutComboDisplay combo={resolved} platform={platform} />}
                </button>
                <button
                  className="chip"
                  type="button"
                  disabled={!isCustom}
                  onClick={() => {
                    saveCustomShortcut(definition.action, null);
                    setConflict(null);
                    setRecording(null);
                    setRevision((value) => value + 1);
                  }}
                >
                  {t("settings.shortcutsReset")}
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </SettingsSection>
  );
}

function normalizeNetworkView(network: NetworkView): NetworkView {
  return { ...network, proxyMode: normalizeProxyMode(network.proxyMode) };
}

const PROXY_TYPES = ["http", "https", "socks5", "socks5h"] as const;
const LANGUAGE_PREFS: LangPref[] = ["", "zh", "en"];
const AUTO_PLAN_MODES = ["off", "on"] as const;
const REASONING_PROTOCOLS: readonly string[] = ["", "deepseek", "openai", "none"];

type AutoPlanMode = (typeof AUTO_PLAN_MODES)[number];
function normalizeAutoPlan(mode: string | undefined): AutoPlanMode {
  return mode === "ask" || mode === "on" ? "on" : "off";
}

function normalizeReasoningProtocol(protocol: string | undefined): string {
  return REASONING_PROTOCOLS.includes(protocol ?? "") ? protocol ?? "" : "";
}

function normalizeReasoningLanguage(lang: string | undefined): string {
  const v = String(lang ?? "").trim().toLowerCase();
  return v === "zh" || v === "en" ? v : "auto";
}

function normalizeProviderView(p: ProviderView): ProviderView {
  const visionModels = asArray(p.visionModels);
  const requiresKey = providerRequiresKey(p);
  return {
    ...p,
    builtIn: Boolean(p.builtIn),
    added: Boolean(p.added),
    models: asArray(p.models),
    visionModels,
    visionModelsConfigured: Boolean(p.visionModelsConfigured ?? visionModels.length > 0),
    modelsUrl: p.modelsUrl ?? "",
    reasoningProtocol: normalizeReasoningProtocol(p.reasoningProtocol),
    supportedEfforts: asArray(p.supportedEfforts),
    requiresKey,
    configured: providerIsConfigured({ ...p, requiresKey }),
    keySource: p.keySource ?? "",
    keySourcePath: p.keySourcePath ?? "",
  };
}

function normalizeSettingsView(view: SettingsView | null | undefined): SettingsView | null {
  if (!view) return null;
  const permissions = view.permissions ?? { mode: "ask", allow: [], ask: [], deny: [] };
  const sandbox = view.sandbox ?? { bash: "enforce", network: false, workspaceRoot: "", allowWrite: [], shell: "auto" };
  const network = view.network ?? {
    proxyMode: "auto",
    proxyUrl: "",
    noProxy: "",
    proxy: { type: "socks5", server: "", port: 0, username: "", password: "" },
  };
  const agent = view.agent ?? { temperature: 0, maxSteps: 0, plannerMaxSteps: 0, systemPrompt: "", coldResumePrune: true, reasoningLanguage: "auto" };
  agent.plannerMaxSteps = Number.isFinite(agent.plannerMaxSteps) ? Math.max(0, Math.trunc(agent.plannerMaxSteps)) : 0;
  agent.maxSteps = Number.isFinite(agent.maxSteps) ? Math.max(0, Math.trunc(agent.maxSteps)) : 0;
  agent.reasoningLanguage = normalizeReasoningLanguage(agent.reasoningLanguage);
  return {
    ...view,
    providers: asArray(view.providers).map(normalizeProviderView),
    officialProviders: asArray(view.officialProviders).map(normalizeProviderView),
    providerKinds: asArray(view.providerKinds),
    permissions: {
      ...permissions,
      allow: asArray(permissions.allow),
      ask: asArray(permissions.ask),
      deny: asArray(permissions.deny),
    },
    sandbox: {
      ...sandbox,
      allowWrite: asArray(sandbox.allowWrite),
    },
    network: {
      ...network,
      proxy: network.proxy ?? { type: "socks5", server: "", port: 0, username: "", password: "" },
    },
    agent,
    bot: normalizeBotSettings(view.bot),
    autoPlan: normalizeAutoPlan(view.autoPlan),
    autoApproveTools: Boolean(view.autoApproveTools ?? view.bypass),
    bypass: Boolean(view.autoApproveTools ?? view.bypass),
    desktopLanguage: normalizeLangPref(view.desktopLanguage),
    desktopLayoutStyle: normalizeDesktopLayoutStyle(view.desktopLayoutStyle),
    desktopTheme: normalizeThemePreference(view.desktopTheme),
    desktopThemeStyle: normalizeThemeStyleForTheme(view.desktopThemeStyle, normalizeThemePreference(view.desktopTheme)),
    closeBehavior: normalizeCloseBehavior(view.closeBehavior),
    displayMode: normalizeDisplayMode(view.displayMode),
    statusBarStyle: normalizeStatusBarStyle(view.statusBarStyle),
    statusBarItems: normalizeStatusBarItems(view.statusBarItems),
    checkUpdates: view.checkUpdates !== false,
  };
}

type CloseBehavior = "background" | "quit";

function normalizeCloseBehavior(mode: string | undefined): CloseBehavior {
  return mode === "quit" ? "quit" : "background";
}

type DisplayMode = "standard" | "compact";

function normalizeDisplayMode(mode: string | undefined): DisplayMode {
  return mode === "standard" || mode === "compact" ? mode : "standard";
}

type DesktopLayoutStyle = "classic" | "workbench" | "creation";

function normalizeDesktopLayoutStyle(style: string | undefined): DesktopLayoutStyle {
  if (style === "classic") return "classic";
  if (style === "creation") return "creation";
  return "workbench";
}

function desktopLayoutStyleLabel(style: DesktopLayoutStyle, t: ReturnType<typeof useT>): string {
  return t(`settings.desktopLayoutStyle.${style}`);
}

type StatusBarStyle = "icon" | "text";
type StatusBarDropPlacement = "before" | "after";
type StatusBarDragTarget = {
  id: StatusBarItemId;
  placement: StatusBarDropPlacement;
};

function normalizeStatusBarStyle(style: string | undefined): StatusBarStyle {
  return style === "icon" ? "icon" : "text";
}

function statusBarItemLabel(id: StatusBarItemId, t: ReturnType<typeof useT>): string {
  switch (id) {
    case "model":
      return t("settings.statusBarItem.model");
    case "workspace":
      return t("settings.statusBarItem.workspace");
    case "git_branch":
      return t("settings.statusBarItem.gitBranch");
    case "cache":
      return t("status.cacheLabel");
    case "cache_avg":
      return t("status.cacheAvgLabel");
    case "session_tokens":
      return t("status.sessionTokensLabel");
    case "turn_tokens":
      return t("status.turnTokensLabel");
    case "turn_cost":
      return t("status.turnCostLabel");
    case "session_turns":
      return t("status.sessionTurnsLabel");
    case "context":
      return t("status.ctxLabel");
    case "compact":
      return t("status.compactLabel");
    case "cost":
      return t("status.costLabel");
    case "balance":
      return t("status.balanceLabel");
  }
}

function closeBehaviorLabel(mode: CloseBehavior, t: ReturnType<typeof useT>): string {
  return mode === "quit" ? t("settings.closeBehavior.quit") : t("settings.closeBehavior.background");
}

function permissionModeLabel(mode: string, t: ReturnType<typeof useT>): string {
  switch (mode) {
    case "allow":
      return t("settings.modeAllowShort");
    case "deny":
      return t("settings.modeDenyShort");
    default:
      return t("settings.modeAskShort");
  }
}

function sandboxModeLabel(mode: string, t: ReturnType<typeof useT>): string {
  return mode === "off" ? t("settings.bashOffShort") : t("settings.bashEnforceShort");
}

function GeneralSection({ s, busy, apply, agentRunning }: SectionProps & { agentRunning: boolean }) {
  const { t, setPref } = useI18n();
  const closeBehavior = normalizeCloseBehavior(s.closeBehavior);
  const [displayMode, setDisplayMode] = useState<DisplayMode>(() => normalizeDisplayMode(getDisplayMode()));
  const [statusBarItemsExpanded, setStatusBarItemsExpanded] = useState(false);
  const [draggingStatusBarItem, setDraggingStatusBarItem] = useState<StatusBarItemId | null>(null);
  const [statusBarDragTarget, setStatusBarDragTargetState] = useState<StatusBarDragTarget | null>(null);
  const draggingStatusBarItemRef = useRef<StatusBarItemId | null>(null);
  const statusBarDragTargetRef = useRef<StatusBarDragTarget | null>(null);
  const mouseDragCleanupRef = useRef<(() => void) | null>(null);
  const soundPanelId = useId();
  const statusBarItemsPanelId = useId();
  useEffect(() => onDisplayModeChange((mode) => setDisplayMode(mode)), []);
  useEffect(() => () => mouseDragCleanupRef.current?.(), []);
  const autoPlan = normalizeAutoPlan(s.autoPlan);
  const languagePref = normalizeLangPref(s.desktopLanguage);
  const desktopLayoutStyle = normalizeDesktopLayoutStyle(s.desktopLayoutStyle);
  const [genMusicPreset, setGenMusicPreset] = useState<GenerativePreset>(getGenerativePreset());
  const [soundPref, setSoundPref] = useState<SoundWavPref>(getSuccessPreference());
  const [attentionPref, setAttentionPref] = useState<SoundWavPref>(getAttentionPreference());
  const [soundExpanded, setSoundExpanded] = useState(false);
  const statusBarStyle = normalizeStatusBarStyle(s.statusBarStyle);
  const statusBarItems = normalizeStatusBarItems(s.statusBarItems);
  const soundStatus = summarizeSoundStatus(genMusicPreset, soundPref, attentionPref);
  const visibleStatusItems = new Set<StatusBarItemId>(statusBarItems);
  const orderedStatusItems = [
    ...statusBarItems,
    ...DEFAULT_STATUS_BAR_ITEMS.filter((id) => !visibleStatusItems.has(id)),
  ];
  const applyStatusBarItems = (items: StatusBarItemId[]) => {
    const contentScrollTop = document.querySelector<HTMLElement>(".settings-center__content")?.scrollTop ?? 0;
    const navScrollTop = document.querySelector<HTMLElement>(".settings-center__nav")?.scrollTop ?? 0;
    const active = document.activeElement;
    if (active instanceof HTMLElement && active.closest(".status-bar-items-editor")) active.blur();
    void apply(() => app.SetStatusBarItems(items)).finally(() => {
      window.scrollTo(0, 0);
      requestAnimationFrame(() => {
        window.scrollTo(0, 0);
        const content = document.querySelector<HTMLElement>(".settings-center__content");
        const nav = document.querySelector<HTMLElement>(".settings-center__nav");
        if (content) content.scrollTop = Math.min(contentScrollTop, Math.max(0, content.scrollHeight - content.clientHeight));
        if (nav) nav.scrollTop = navScrollTop;
      });
    });
  };
  const toggleStatusBarItem = (id: StatusBarItemId) => {
    if (visibleStatusItems.has(id)) {
      if (statusBarItems.length <= 1) return;
      applyStatusBarItems(statusBarItems.filter((item) => item !== id));
      return;
    }
    applyStatusBarItems([...statusBarItems, id]);
  };
  const moveStatusBarItem = (id: StatusBarItemId, direction: -1 | 1) => {
    const idx = statusBarItems.indexOf(id);
    const nextIdx = idx + direction;
    if (idx < 0 || nextIdx < 0 || nextIdx >= statusBarItems.length) return;
    const next = [...statusBarItems];
    [next[idx], next[nextIdx]] = [next[nextIdx], next[idx]];
    applyStatusBarItems(next);
  };
  const reorderStatusBarItem = (fromId: StatusBarItemId, toId: StatusBarItemId, placement: StatusBarDropPlacement) => {
    const fromIdx = statusBarItems.indexOf(fromId);
    const toIdx = statusBarItems.indexOf(toId);
    if (fromIdx < 0 || toIdx < 0 || fromIdx === toIdx) return;
    const next = statusBarItems.filter((item) => item !== fromId);
    const insertAt = next.indexOf(toId);
    if (insertAt < 0) return;
    next.splice(placement === "after" ? insertAt + 1 : insertAt, 0, fromId);
    if (next.every((item, index) => item === statusBarItems[index])) return;
    applyStatusBarItems(next);
  };
  const statusBarItemFromPoint = (x: number, y: number): StatusBarDragTarget | null => {
    const row = document.elementFromPoint(x, y)?.closest<HTMLElement>("[data-statusbar-setting-item]");
    const id = row?.dataset.statusbarSettingItem as StatusBarItemId | undefined;
    if (!row || !id || !statusBarItems.includes(id)) return null;
    const rect = row.getBoundingClientRect();
    return { id, placement: y < rect.top + rect.height / 2 ? "before" : "after" };
  };
  const setStatusBarDragTarget = (target: StatusBarDragTarget | null) => {
    const current = statusBarDragTargetRef.current;
    if (current?.id === target?.id && current?.placement === target?.placement) return;
    statusBarDragTargetRef.current = target;
    setStatusBarDragTargetState(target);
  };
  const beginStatusBarDrag = (id: StatusBarItemId, visible: boolean): boolean => {
    if (busy || !visible) return false;
    mouseDragCleanupRef.current?.();
    mouseDragCleanupRef.current = null;
    draggingStatusBarItemRef.current = id;
    statusBarDragTargetRef.current = null;
    setDraggingStatusBarItem(id);
    setStatusBarDragTargetState(null);
    return true;
  };
  const updateStatusBarDrag = (clientX: number, clientY: number) => {
    const draggingId = draggingStatusBarItemRef.current;
    if (!draggingId) return;
    const target = statusBarItemFromPoint(clientX, clientY);
    setStatusBarDragTarget(target && target.id !== draggingId ? target : null);
  };
  const finishStatusBarDrag = (clientX?: number, clientY?: number) => {
    const draggingId = draggingStatusBarItemRef.current;
    let target = statusBarDragTargetRef.current;
    if (draggingId && clientX !== undefined && clientY !== undefined) {
      const pointerTarget = statusBarItemFromPoint(clientX, clientY);
      if (pointerTarget && pointerTarget.id !== draggingId) target = pointerTarget;
    }
    if (draggingId && target) reorderStatusBarItem(draggingId, target.id, target.placement);
    draggingStatusBarItemRef.current = null;
    statusBarDragTargetRef.current = null;
    setDraggingStatusBarItem(null);
    setStatusBarDragTargetState(null);
  };
  const cancelStatusBarDrag = () => {
    mouseDragCleanupRef.current?.();
    mouseDragCleanupRef.current = null;
    draggingStatusBarItemRef.current = null;
    statusBarDragTargetRef.current = null;
    setDraggingStatusBarItem(null);
    setStatusBarDragTargetState(null);
  };
  const startStatusBarPointerDrag = (event: PointerEvent<HTMLElement>, id: StatusBarItemId, visible: boolean) => {
    if (event.button !== 0 || !beginStatusBarDrag(id, visible)) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
  };
  const moveStatusBarPointerDrag = (event: PointerEvent<HTMLElement>) => {
    if (!draggingStatusBarItemRef.current) return;
    event.preventDefault();
    updateStatusBarDrag(event.clientX, event.clientY);
  };
  const endStatusBarPointerDrag = (event: PointerEvent<HTMLElement>) => {
    if (!draggingStatusBarItemRef.current) return;
    event.preventDefault();
    try {
      event.currentTarget.releasePointerCapture(event.pointerId);
    } catch {
      // Pointer capture may already be released by the browser.
    }
    finishStatusBarDrag(event.clientX, event.clientY);
  };
  const cancelStatusBarPointerDrag = (event: PointerEvent<HTMLElement>) => {
    event.preventDefault();
    cancelStatusBarDrag();
  };
  const startStatusBarMouseDrag = (event: ReactMouseEvent<HTMLElement>, id: StatusBarItemId, visible: boolean) => {
    if (event.button !== 0 || !beginStatusBarDrag(id, visible)) return;
    event.preventDefault();
    const handleMove = (moveEvent: MouseEvent) => {
      moveEvent.preventDefault();
      updateStatusBarDrag(moveEvent.clientX, moveEvent.clientY);
    };
    const cleanup = () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
    };
    const handleUp = (upEvent: MouseEvent) => {
      upEvent.preventDefault();
      cleanup();
      mouseDragCleanupRef.current = null;
      finishStatusBarDrag(upEvent.clientX, upEvent.clientY);
    };
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);
    mouseDragCleanupRef.current = cleanup;
  };
  const setLanguage = (next: LangPref) => {
    setPref(next);
    void apply(() => app.SetDesktopLanguage(next));
  };
  return (
    <SettingsSection>
      <SettingsField label={t("settings.language")}>
        <div className="set-seg">
          {LANGUAGE_PREFS.map((pref) => (
            <button
              key={pref || "auto"}
              className={`set-seg__btn${languagePref === pref ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => setLanguage(pref)}
            >
              {pref === "" ? t("settings.langAuto") : pref === "zh" ? "中文" : "English"}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.desktopLayoutStyle")}>
        <div className="set-seg">
          {(["classic", "workbench", "creation"] as const).map((style) => (
            <button
              key={style}
              className={`set-seg__btn${desktopLayoutStyle === style ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => void apply(() => app.SetDesktopLayoutStyle(style))}
            >
              {desktopLayoutStyleLabel(style, t)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.closeBehavior")}>
        <div className="set-seg">
          {(["background", "quit"] as const).map((mode) => (
            <button
              key={mode}
              className={`set-seg__btn${closeBehavior === mode ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => void apply(() => app.SetCloseBehavior(mode))}
            >
              {closeBehaviorLabel(mode, t)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.displayMode")}>
        <div className="set-seg">
          {(["standard", "compact"] as const).map((mode) => (
            <button
              key={mode}
              className={`set-seg__btn${displayMode === mode ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => {
                setLocalDisplayMode(mode);
                void apply(() => app.SetDisplayMode(mode));
              }}
            >
              {t(`settings.displayMode.${mode}`)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.autoPlan")}>
        <div className="set-seg">
          {AUTO_PLAN_MODES.map((mode) => (
            <button
              key={mode}
              className={`set-seg__btn${autoPlan === mode ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => void apply(() => app.SetAutoPlan(mode))}
            >
              {t(`settings.autoPlan.${mode}`)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.sound")} hint={t("settings.soundHint")} stacked>
        <div className={`settings-sound-editor${soundExpanded ? " settings-sound-editor--expanded" : ""}`}>
          <div className="settings-sound-editor__summary">
            <span className={`settings-sound-editor__status settings-sound-editor__status--${soundStatus}`}>
              {t(`settings.soundStatus.${soundStatus}`)}
            </span>
            <Tooltip label={t(soundExpanded ? "settings.soundCollapse" : "settings.soundExpand")}>
              <button
                type="button"
                className="settings-sound-editor__toggle"
                aria-expanded={soundExpanded}
                aria-controls={soundPanelId}
                aria-label={t(soundExpanded ? "settings.soundCollapse" : "settings.soundExpand")}
                onClick={() => setSoundExpanded((open) => !open)}
              >
                {soundExpanded ? <ChevronUp size={15} aria-hidden="true" /> : <ChevronDown size={15} aria-hidden="true" />}
              </button>
            </Tooltip>
          </div>
          {soundExpanded && (
            <div className="settings-sound-editor__list" id={soundPanelId}>
              <div className="settings-sound-row">
                <span className="settings-sound-row__label">{t("settings.generativeMusic")}</span>
                <GenMusicSelect
                  value={genMusicPreset}
                  onChange={(next) => {
                    setGenMusicPreset(next);
                    setGenerativePreset(next);
                    if (next === "off") {
                      generativeMusic.stop();
                    } else {
                      if (generativeMusic.isRunning) {
                        generativeMusic.setPreset(next);
                      } else if (agentRunning) {
                        generativeMusic.start(next);
                      }
                      generativeMusic.playPreview(next);
                    }
                  }}
                  onPreview={() => { if (genMusicPreset !== "off") generativeMusic.playPreview(genMusicPreset); }}
                  previewDisabled={genMusicPreset === "off"}
                />
              </div>
              <div className="settings-sound-row">
                <span className="settings-sound-row__label">{t("settings.notificationSoundSuccess")}</span>
                <SoundSelect
                  value={soundPref}
                  onChange={(next) => {
                    setSoundPref(next);
                    setSuccessPreference(next);
                    playSuccessChime();
                  }}
                  onPreview={playSuccessChime}
                  previewDisabled={soundPref === "off"}
                />
              </div>
              <div className="settings-sound-row">
                <span className="settings-sound-row__label">{t("settings.notificationSoundAttention")}</span>
                <SoundSelect
                  value={attentionPref}
                  onChange={(next) => {
                    setAttentionPref(next);
                    setAttentionPreference(next);
                    playAttentionChime();
                  }}
                  onPreview={playAttentionChime}
                  previewDisabled={attentionPref === "off"}
                />
              </div>
            </div>
          )}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.statusBarStyle")}>
        <div className="set-seg">
          {(["icon", "text"] as const).map((style) => (
            <button
              key={style}
              className={`set-seg__btn${statusBarStyle === style ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => void apply(() => app.SetStatusBarStyle(style))}
            >
              {t(`settings.statusBarStyle.${style}`)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.statusBarItems")} hint={t("settings.statusBarItemsHint")} stacked>
        <div className={`status-bar-items-editor${statusBarItemsExpanded ? " status-bar-items-editor--expanded" : ""}`}>
          <div className="status-bar-items-editor__summary">
            <span className="status-bar-items-editor__summary-text">
              {t("settings.statusBarItemsSummary", { visible: statusBarItems.length, total: DEFAULT_STATUS_BAR_ITEMS.length })}
            </span>
            <Tooltip label={t(statusBarItemsExpanded ? "settings.statusBarItemsCollapse" : "settings.statusBarItemsExpand")}>
              <button
                type="button"
                className="status-bar-items-editor__toggle"
                aria-expanded={statusBarItemsExpanded}
                aria-controls={statusBarItemsPanelId}
                aria-label={t(statusBarItemsExpanded ? "settings.statusBarItemsCollapse" : "settings.statusBarItemsExpand")}
                onClick={() => setStatusBarItemsExpanded((open) => !open)}
              >
                {statusBarItemsExpanded ? <ChevronUp size={15} aria-hidden="true" /> : <ChevronDown size={15} aria-hidden="true" />}
              </button>
            </Tooltip>
          </div>
          {statusBarItemsExpanded && (
            <div className="status-bar-items-editor__list" id={statusBarItemsPanelId}>
              {orderedStatusItems.map((id) => {
                const label = statusBarItemLabel(id, t);
                const visible = visibleStatusItems.has(id);
                const visibleIndex = statusBarItems.indexOf(id);
                const disableHide = visible && statusBarItems.length <= 1;
                const dragLabel = t("settings.statusBarItem.drag", { label });
                const moveUpLabel = t("settings.statusBarItem.moveUp", { label });
                const moveDownLabel = t("settings.statusBarItem.moveDown", { label });
                const dropPlacement = statusBarDragTarget?.id === id ? statusBarDragTarget.placement : null;
                return (
                  <div
                    className={[
                      "status-bar-item-row",
                      visible ? "" : "status-bar-item-row--hidden",
                      draggingStatusBarItem === id ? "status-bar-item-row--dragging" : "",
                      dropPlacement ? "status-bar-item-row--drag-over" : "",
                      dropPlacement === "before" ? "status-bar-item-row--drop-before" : "",
                      dropPlacement === "after" ? "status-bar-item-row--drop-after" : "",
                    ].filter(Boolean).join(" ")}
                    data-statusbar-setting-item={id}
                    key={id}
                  >
                    <Tooltip label={dragLabel}>
                      <button
                        type="button"
                        className="status-bar-item-row__drag"
                        disabled={!visible || busy}
                        aria-label={dragLabel}
                        title={dragLabel}
                        onPointerDown={(event) => startStatusBarPointerDrag(event, id, visible)}
                        onPointerMove={moveStatusBarPointerDrag}
                        onPointerUp={endStatusBarPointerDrag}
                        onPointerCancel={cancelStatusBarPointerDrag}
                        onMouseDown={(event) => startStatusBarMouseDrag(event, id, visible)}
                      >
                        <GripVertical size={14} aria-hidden="true" />
                      </button>
                    </Tooltip>
                    <label className="status-bar-item-row__toggle">
                      <input
                        type="checkbox"
                        checked={visible}
                        disabled={busy || disableHide}
                        onChange={() => toggleStatusBarItem(id)}
                      />
                      <span className="status-bar-item-row__check" aria-hidden="true">
                        {visible && <Check size={12} />}
                      </span>
                      <span className="status-bar-item-row__label">{label}</span>
                    </label>
                    <div className="status-bar-item-row__actions">
                      <Tooltip label={moveUpLabel}>
                        <button
                          type="button"
                          className="status-bar-item-row__order"
                          disabled={busy || !visible || visibleIndex <= 0}
                          onClick={() => moveStatusBarItem(id, -1)}
                          aria-label={moveUpLabel}
                        >
                          <ChevronUp size={14} aria-hidden="true" />
                        </button>
                      </Tooltip>
                      <Tooltip label={moveDownLabel}>
                        <button
                          type="button"
                          className="status-bar-item-row__order"
                          disabled={busy || !visible || visibleIndex < 0 || visibleIndex >= statusBarItems.length - 1}
                          onClick={() => moveStatusBarItem(id, 1)}
                          aria-label={moveDownLabel}
                        >
                          <ChevronDown size={14} aria-hidden="true" />
                        </button>
                      </Tooltip>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </SettingsField>
    </SettingsSection>
  );
}

const GENRE_OPTIONS: { value: GenerativePreset; labelKey: DictKey }[] = [
  { value: "off", labelKey: "settings.generativeMusic.off" },
  { value: "ethereal", labelKey: "settings.generativeMusic.presets.ethereal" },
  { value: "classic", labelKey: "settings.generativeMusic.presets.classic" },
  { value: "digital", labelKey: "settings.generativeMusic.presets.digital" },
  { value: "retro", labelKey: "settings.generativeMusic.presets.retro" },
];

function summarizeSoundStatus(
  music: GenerativePreset,
  success: SoundWavPref,
  attention: SoundWavPref,
): "allOff" | "enabled" | "custom" {
  const enabledCount = [music !== "off", success !== "off", attention !== "off"].filter(Boolean).length;
  if (enabledCount === 0) return "allOff";
  if (enabledCount === 1) return "enabled";
  return "custom";
}

function GenMusicSelect({
  value,
  onChange,
  onPreview,
  previewDisabled,
}: {
  value: GenerativePreset;
  onChange: (v: GenerativePreset) => void;
  onPreview: () => void;
  previewDisabled?: boolean;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const selected = GENRE_OPTIONS.find((o) => o.value === value) ?? GENRE_OPTIONS[0];

  return (
    <div className="sound-select">
      <button
        ref={triggerRef}
        className="sound-select__trigger"
        type="button"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="sound-select__label">{t(selected.labelKey)}</span>
        <ChevronDown
          size={16}
          className={`sound-select__chev${open ? " sound-select__chev--open" : ""}`}
        />
      </button>
      {!previewDisabled && (
        <button className="chip chip--icon" type="button" title={t("settings.generativeMusicPreview")} aria-label={t("settings.generativeMusicPreview")} onClick={onPreview}>
          <Play size={13} aria-hidden="true" />
        </button>
      )}
      <AnchoredPopover
        open={open}
        anchorRef={triggerRef}
        onClose={() => setOpen(false)}
        className="sound-select__menu"
        placement="bottom"
      >
        <div className="sound-select__list" role="listbox">
          {GENRE_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              className={`sound-select__option${opt.value === value ? " sound-select__option--selected" : ""}`}
              role="option"
              aria-selected={opt.value === value}
              type="button"
              onClick={() => {
                onChange(opt.value);
                setOpen(false);
              }}
            >
              <span>{t(opt.labelKey)}</span>
              {opt.value === value && <Check size={14} className="sound-select__check" />}
            </button>
          ))}
        </div>
      </AnchoredPopover>
    </div>
  );
}

function NetworkSection({ s, busy, apply }: SectionProps) {
  const t = useT();
  const savedNetwork = normalizeNetworkView(s.network);
  const [draft, setDraft] = useState<NetworkView>(savedNetwork);
  useEffect(() => setDraft(normalizeNetworkView(s.network)), [s.network]);
  const dirty = JSON.stringify(draft) !== JSON.stringify(savedNetwork);
  const setProxy = (next: Partial<NetworkView["proxy"]>) => {
    setDraft({ ...draft, proxy: { ...draft.proxy, ...next } });
  };

  return (
    <SettingsSection
      title={t("settings.tab.network")}
      actions={
        <button
          className="btn btn--primary btn--small"
          disabled={busy || !dirty}
          onClick={() => void apply(() => app.SetNetwork(draft))}
        >
          {t("settings.saveNetwork")}
        </button>
      }
    >
      <SettingsField label={t("settings.proxyMode")}>
        <div className="set-seg">
          {PROXY_MODES.map((mode) => (
            <button
              key={mode}
              className={`set-seg__btn${draft.proxyMode === mode ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => setDraft({ ...draft, proxyMode: mode })}
            >
              {proxyModeLabel(mode, t)}
            </button>
          ))}
        </div>
      </SettingsField>

      {draft.proxyMode === "custom" && (
        <>
          <SettingsField label={t("settings.proxyType")}>
            <div className="set-seg">
              {PROXY_TYPES.map((typ) => (
                <button
                  key={typ}
                  className={`set-seg__btn${draft.proxy.type === typ ? " set-seg__btn--on" : ""}`}
                  disabled={busy}
                  onClick={() => setProxy({ type: typ })}
                >
                  {typ.toUpperCase()}
                </button>
              ))}
            </div>
          </SettingsField>
          <SettingsField label={t("settings.proxyServer")}>
            <div className="settings-inline-controls">
            <input
              className="mem-input set-grow"
              placeholder="127.0.0.1"
              value={draft.proxy.server}
              disabled={busy || !!draft.proxyUrl.trim()}
              onChange={(e) => setProxy({ server: e.target.value })}
            />
            <label className="set-label">{t("settings.proxyPort")}</label>
            <input
              className="mem-input set-narrow"
              placeholder="7890"
              value={draft.proxy.port ? String(draft.proxy.port) : ""}
              disabled={busy || !!draft.proxyUrl.trim()}
              inputMode="numeric"
              onChange={(e) => setProxy({ port: Number(e.target.value) || 0 })}
            />
            </div>
          </SettingsField>
          <SettingsField label={t("settings.proxyUsername")}>
            <div className="settings-inline-controls">
            <input
              className="mem-input set-grow"
              value={draft.proxy.username}
              disabled={busy || !!draft.proxyUrl.trim()}
              onChange={(e) => setProxy({ username: e.target.value })}
            />
            <label className="set-label">{t("settings.proxyPassword")}</label>
            <input
              className="mem-input set-grow"
              type="password"
              value={draft.proxy.password}
              disabled={busy || !!draft.proxyUrl.trim()}
              onChange={(e) => setProxy({ password: e.target.value })}
            />
            </div>
          </SettingsField>
          <SettingsField label={t("settings.proxyUrl")} hint={t("settings.proxyUrlHint")}>
              <input
                className="mem-input set-grow"
                placeholder="socks5://127.0.0.1:7890"
                value={draft.proxyUrl}
                disabled={busy}
                onChange={(e) => setDraft({ ...draft, proxyUrl: e.target.value })}
              />
          </SettingsField>
          <SettingsField label={t("settings.noProxy")}>
            <input
              className="mem-input set-grow"
              placeholder="localhost,127.0.0.1,.local"
              value={draft.noProxy}
              disabled={busy}
              onChange={(e) => setDraft({ ...draft, noProxy: e.target.value })}
            />
          </SettingsField>
        </>
      )}
    </SettingsSection>
  );
}

function PermissionsSection({ s, busy, apply }: SectionProps) {
  const t = useT();
  return (
    <>
    <SettingsSection title={t("settings.permissions")} description={t("settings.permissionsModeHint")}>
      <SettingsField label={t("settings.writerMode")}>
        <select
          className="mem-select set-grow"
          value={s.permissions.mode}
          disabled={busy}
          onChange={(e) => void apply(() => app.SetPermissionMode(e.target.value))}
        >
          <option value="ask">{t("settings.modeAsk")}</option>
          <option value="allow">{t("settings.modeAllow")}</option>
          <option value="deny">{t("settings.modeDeny")}</option>
        </select>
      </SettingsField>
    </SettingsSection>
    <SettingsSection title={t("settings.permissionRules")} description={t("settings.ruleForm")}>
      <div className="set-rules-grid">
        {(["deny", "ask", "allow"] as const).map((list) => (
          <RuleList
            key={list}
            list={list}
            rules={s.permissions[list]}
            busy={busy}
            onAdd={(rule) => apply(() => app.AddPermissionRule(list, rule))}
            onRemove={(rule) => apply(() => app.RemovePermissionRule(list, rule))}
          />
        ))}
      </div>
    </SettingsSection>
    </>
  );
}

function RuleList({
  list,
  rules,
  busy,
  onAdd,
  onRemove,
}: {
  list: string;
  rules: string[];
  busy: boolean;
  onAdd: (rule: string) => Promise<void>;
  onRemove: (rule: string) => Promise<void>;
}) {
  const t = useT();
  const [draft, setDraft] = useState("");
  const add = () => {
    const r = draft.trim();
    if (r) {
      void onAdd(r);
      setDraft("");
    }
  };
  return (
    <div className="set-rules">
      <div className="set-rules__head">
        <div className="set-rules__label">{ruleListLabel(list, t)}</div>
        {ruleListHint(list, t) && <div className="set-rules__hint">{ruleListHint(list, t)}</div>}
      </div>
      <div className="set-rules__chips">
        {rules.length === 0 && <span className="mem-empty">{t("common.none")}</span>}
        {rules.map((r) => (
          <span className="set-rule" key={r}>
            {r}
            <Tooltip label={t("common.delete")}>
              <button className="set-rule__x" disabled={busy} onClick={() => void onRemove(r)}>
                ✕
              </button>
            </Tooltip>
          </span>
        ))}
      </div>
      <div className="set-rules__add">
        <input
          className="mem-input"
          placeholder={t("settings.addRule", { list })}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") add();
          }}
        />
        <button className="btn btn--small" disabled={busy || !draft.trim()} onClick={add}>
          {t("common.add")}
        </button>
      </div>
    </div>
  );
}

function ruleListLabel(list: string, t: ReturnType<typeof useT>): string {
  switch (list) {
    case "deny":
      return t("settings.ruleDeny");
    case "ask":
      return t("settings.ruleAsk");
    case "allow":
      return t("settings.ruleAllow");
    case "allow_write":
      return t("settings.ruleAllowWrite");
    default:
      return list;
  }
}

function ruleListHint(list: string, t: ReturnType<typeof useT>): string {
  switch (list) {
    case "deny":
      return t("settings.ruleDenyHint");
    case "ask":
      return t("settings.ruleAskHint");
    case "allow":
      return t("settings.ruleAllowHint");
    default:
      return "";
  }
}

type HookScope = "global" | "project";

function HooksSection({ onChanged }: { onChanged: (settings?: SettingsView | null) => void }) {
  const t = useT();
  const [scope, setScope] = useState<HookScope>("global");
  const [view, setView] = useState<HooksSettingsView | null>(null);
  const [jsonText, setJsonText] = useState("");
  const [jsonMessage, setJsonMessage] = useState<string | null>(null);
  const [jsonError, setJsonError] = useState<string | null>(null);
  const [pathMessage, setPathMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const load = useCallback(async (nextScope: HookScope) => {
    setBusy(true);
    setErr(null);
    try {
      const next = normalizeHooksSettingsView(await app.HooksSettings(nextScope), nextScope);
      setView(next);
      setJsonText(formatHooksJSON(next.hooks, next.events));
      setJsonMessage(null);
      setJsonError(null);
      setPathMessage(null);
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
      setView(null);
      setJsonText("");
      setJsonMessage(null);
      setJsonError(null);
      setPathMessage(null);
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    void load(scope);
  }, [load, scope]);

  const parseHooksEditorJSON = (raw = jsonText): { hooks: HookConfigView[]; text: string } | null => {
    try {
      const hooks = parseHooksJSON(raw, view?.events ?? []);
      const text = formatHooksJSON(hooks, view?.events ?? []);
      setJsonText(text);
      setJsonError(null);
      return { hooks, text };
    } catch (e) {
      setJsonError(t("settings.hooksJsonInvalid", { error: String((e as Error)?.message ?? e) }));
      setJsonMessage(null);
      return null;
    }
  };
  const copyHooksJSON = async () => {
    const parsed = parseHooksEditorJSON();
    if (!parsed) return;
    try {
      await navigator.clipboard?.writeText(parsed.text);
      setJsonMessage(t("settings.hooksJsonCopied"));
    } catch {
      setJsonMessage(t("settings.hooksJsonClipboardUnavailable"));
    }
  };
  const formatHooksEditorJSON = (raw = jsonText) => {
    const parsed = parseHooksEditorJSON(raw);
    if (parsed) setJsonMessage(t("settings.hooksJsonFormatted"));
  };
  const pasteHooksJSON = async () => {
    try {
      const raw = await navigator.clipboard?.readText();
      if (!raw) throw new Error(t("settings.hooksJsonClipboardEmpty"));
      setJsonText(raw);
      formatHooksEditorJSON(raw);
    } catch (e) {
      setJsonError(t("settings.hooksJsonPasteFailed", { error: String((e as Error)?.message ?? e) }));
      setJsonMessage(null);
    }
  };
  const copyHooksPath = async () => {
    const path = view?.path?.trim();
    if (!path) {
      setPathMessage(t("settings.hooksPathUnavailable"));
      return;
    }
    try {
      await navigator.clipboard?.writeText(path);
      setPathMessage(t("settings.hooksPathCopied"));
    } catch {
      setPathMessage(t("settings.hooksJsonClipboardUnavailable"));
    }
  };
  const save = async () => {
    setBusy(true);
    setErr(null);
    try {
      const parsed = parseHooksEditorJSON();
      if (!parsed) return;
      await app.SaveHooksSettingsForRoot(scope, view?.projectRoot?.trim() ?? "", parsed.hooks);
      await load(scope);
      onChanged();
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
    } finally {
      setBusy(false);
    }
  };
  const trustProject = async () => {
    const projectRoot = view?.projectRoot?.trim() ?? "";
    if (!projectRoot) {
      setErr(t("settings.hooksProjectRootUnavailable"));
      return;
    }
    setBusy(true);
    setErr(null);
    try {
      await app.TrustProjectHooksForRoot(projectRoot);
      await load("project");
      onChanged();
    } catch (e) {
      setErr(String((e as Error)?.message ?? e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      {err && <div className="banner banner--error">{err}</div>}
      <SettingsSection title={t("settings.hooksScopeSection")} description={t("settings.hooksScopeHint")}>
        <SettingsField label={t("settings.hooksScopeField")}>
          <select className="mem-select set-grow" value={scope} disabled={busy} onChange={(e) => setScope(e.target.value === "project" ? "project" : "global")}>
            <option value="global">{t("settings.hooksGlobal")}</option>
            <option value="project">{t("settings.hooksProject")}</option>
          </select>
        </SettingsField>
        <SettingsField label={t("settings.hooksPath")} hint={scope === "project" ? t("settings.hooksPathProjectHint") : t("settings.hooksPathGlobalHint")}>
          <div className="hooks-path-stack">
            <div className={`hooks-path-display${view?.path ? "" : " hooks-path-display--empty"}`}>
              <code className="hooks-path-display__value" title={view?.path || t("settings.hooksPathUnavailable")}>
                {view?.path || t("settings.hooksPathUnavailable")}
              </code>
              <button className="btn btn--small" disabled={busy || !view?.path} onClick={() => void copyHooksPath()}>{t("settings.hooksPathCopy")}</button>
            </div>
            {pathMessage && <div className="hooks-path-display__message">{pathMessage}</div>}
          </div>
        </SettingsField>
        {scope === "project" && (
          <SettingsField label={t("settings.hooksTrust")} hint={t("settings.hooksTrustHint")}>
            <div className="hooks-trust-stack">
              <div className="hooks-trust-row">
                <span className={`set-rule${view?.trusted ? "" : " set-rule--warn"}`}>{view?.trusted ? t("settings.hooksTrusted") : t("settings.hooksUntrusted")}</span>
                <button className="btn btn--small" disabled={busy || view?.trusted || !view?.projectRoot} onClick={() => void trustProject()}>{t("settings.hooksTrustProject")}</button>
              </div>
              <code className={`hooks-trust-root${view?.projectRoot ? "" : " hooks-trust-root--empty"}`} title={view?.projectRoot || t("settings.hooksProjectRootUnavailable")}>
                {view?.projectRoot || t("settings.hooksProjectRootUnavailable")}
              </code>
            </div>
          </SettingsField>
        )}
      </SettingsSection>

      <SettingsSection
        title={t("settings.hooks")}
        description={scope === "project" ? t("settings.hooksProjectHint") : t("settings.hooksGlobalHint")}
        actions={(
          <button className="btn btn--small btn--primary" disabled={busy} onClick={() => void save()}>{t("common.save")}</button>
        )}
      >
        {view && (
          <div className="hooks-json-panel">
            <div className="hooks-json-panel__head">
              <div>
                <div className="set-rules__label">{t("settings.hooksJsonTitle")}</div>
                <div className="set-rules__hint">{t("settings.hooksJsonHint")}</div>
              </div>
              <div className="hooks-json-panel__actions">
                <button className="btn btn--small" disabled={busy} onClick={() => void copyHooksJSON()}>{t("settings.hooksJsonCopy")}</button>
                <button className="btn btn--small" disabled={busy} onClick={() => void pasteHooksJSON()}>{t("settings.hooksJsonPaste")}</button>
                <button className="btn btn--small" disabled={busy || !jsonText.trim()} onClick={() => formatHooksEditorJSON()}>{t("settings.hooksJsonApply")}</button>
              </div>
            </div>
            <textarea
              className="mem-textarea hooks-json-panel__textarea"
              value={jsonText}
              disabled={busy}
              spellCheck={false}
              onChange={(e) => {
                setJsonText(e.target.value);
                setJsonMessage(null);
                setJsonError(null);
              }}
            />
            {jsonError && <div className="hooks-json-panel__message hooks-json-panel__message--error">{jsonError}</div>}
            {jsonMessage && <div className="hooks-json-panel__message">{jsonMessage}</div>}
          </div>
        )}
        {!view && <div className="empty">{t("settings.loading")}</div>}
      </SettingsSection>
    </>
  );
}

function normalizeHooksSettingsView(view: HooksSettingsView, scope: HookScope): HooksSettingsView {
  const events = asArray(view?.events).filter(Boolean);
  return {
    scope: view?.scope === "project" ? "project" : scope,
    path: view?.path ?? "",
    projectRoot: view?.projectRoot ?? "",
    trusted: !!view?.trusted,
    events,
    hooks: asArray(view?.hooks).map(normalizeHookConfig).filter((h) => h.event),
  };
}

function formatHooksJSON(hooks: HookConfigView[], eventOrder: string[]): string {
  const grouped: Record<string, Array<Record<string, string | number>>> = {};
  const events = new Set(eventOrder);
  for (const hook of hooks.map(normalizeHookConfig).filter((h) => h.event)) {
    events.add(hook.event);
    const entry: Record<string, string | number> = { command: hook.command };
    if (hook.match) entry.match = hook.match;
    if (hook.description) entry.description = hook.description;
    if ((hook.timeout ?? 0) > 0) entry.timeout = hook.timeout ?? 0;
    if (hook.cwd) entry.cwd = hook.cwd;
    (grouped[hook.event] ||= []).push(entry);
  }
  const ordered: typeof grouped = {};
  for (const event of [...eventOrder, ...Array.from(events).sort()]) {
    if (grouped[event]?.length && !ordered[event]) ordered[event] = grouped[event];
  }
  return JSON.stringify({ hooks: ordered }, null, 2);
}

function parseHooksJSON(raw: string, validEvents: string[]): HookConfigView[] {
  const trimmed = raw.trim();
  if (!trimmed) return [];
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (e) {
    throw new Error(String((e as Error)?.message ?? e));
  }
  if (Array.isArray(parsed)) {
    return parsed.map((item) => normalizeHookConfig(parseHookArrayItem(item, validEvents))).filter((h) => h.event);
  }
  if (!parsed || typeof parsed !== "object") {
    throw new Error("expected an object or array");
  }
  const obj = parsed as Record<string, unknown>;
  const hooksValue = obj.hooks && typeof obj.hooks === "object" && !Array.isArray(obj.hooks) ? obj.hooks : obj;
  return flattenHooksMap(hooksValue as Record<string, unknown>, validEvents);
}

function parseHookArrayItem(item: unknown, validEvents: string[]): HookConfigView {
  if (!item || typeof item !== "object" || Array.isArray(item)) throw new Error("hook item must be an object");
  const obj = item as Record<string, unknown>;
  const event = stringField(obj, "event") || "PreToolUse";
  if (validEvents.length > 0 && !validEvents.includes(event)) throw new Error(`unknown hook event ${event}`);
  return {
    event,
    match: stringField(obj, "match"),
    command: stringField(obj, "command"),
    description: stringField(obj, "description"),
    timeout: numberField(obj, "timeout"),
    cwd: stringField(obj, "cwd"),
  };
}

function flattenHooksMap(hooks: Record<string, unknown>, validEvents: string[]): HookConfigView[] {
  const valid = new Set(validEvents);
  const out: HookConfigView[] = [];
  for (const [event, value] of Object.entries(hooks)) {
    if (valid.size > 0 && !valid.has(event)) throw new Error(`unknown hook event ${event}`);
    const items = Array.isArray(value) ? value : [value];
    for (const item of items) {
      if (!item || typeof item !== "object" || Array.isArray(item)) throw new Error(`hook ${event} item must be an object`);
      const obj = item as Record<string, unknown>;
      out.push(normalizeHookConfig({
        event,
        match: stringField(obj, "match"),
        command: stringField(obj, "command"),
        description: stringField(obj, "description"),
        timeout: numberField(obj, "timeout"),
        cwd: stringField(obj, "cwd"),
      }));
    }
  }
  return out.filter((h) => h.event);
}

function stringField(obj: Record<string, unknown>, key: string): string {
  const value = obj[key];
  return typeof value === "string" ? value : "";
}

function numberField(obj: Record<string, unknown>, key: string): number {
  const value = obj[key];
  return typeof value === "number" && Number.isFinite(value) ? Math.floor(value) : 0;
}

function normalizeHookConfig(h: HookConfigView): HookConfigView {
  return {
    event: h.event || "PreToolUse",
    match: h.match ?? "",
    command: h.command ?? "",
    description: h.description ?? "",
    timeout: h.timeout && h.timeout > 0 ? Math.floor(h.timeout) : 0,
    cwd: h.cwd ?? "",
  };
}

function SandboxSection({ s, busy, apply }: SectionProps) {
  const t = useT();
  const sb = s.sandbox;
  const [root, setRoot] = useState(sb.workspaceRoot);
  const set = (next: Partial<typeof sb>) =>
    apply(() => app.SetSandbox(next.bash ?? sb.bash, next.network ?? sb.network, next.workspaceRoot ?? sb.workspaceRoot, next.allowWrite ?? sb.allowWrite, next.shell ?? sb.shell));

  return (
    <SettingsSection title={t("settings.sandboxTitle")}>
      <SettingsField label={t("settings.shellInterpreter")}>
        <select className="mem-select set-grow" value={sb.shell || "auto"} disabled={busy} onChange={(e) => void set({ shell: e.target.value })}>
          <option value="auto">{t("settings.shellAuto")}</option>
          <option value="bash">{t("settings.shellBash")}</option>
          <option value="powershell">{t("settings.shellPowershell")}</option>
          <option value="pwsh">{t("settings.shellPwsh")}</option>
        </select>
      </SettingsField>
      <SettingsField label={t("settings.bashSandbox")}>
        <select className="mem-select set-grow" value={sb.bash} disabled={busy} onChange={(e) => void set({ bash: e.target.value })}>
          <option value="enforce">{t("settings.bashEnforce")}</option>
          <option value="off">{t("settings.bashOff")}</option>
        </select>
      </SettingsField>
      <SettingsField label={t("settings.allowNetwork")}>
        <label className="set-check set-check--inline">
          <input type="checkbox" checked={sb.network} disabled={busy} onChange={(e) => void set({ network: e.target.checked })} />
          {t("settings.allowNetwork")}
        </label>
      </SettingsField>
      <SettingsField label={t("settings.workspaceRoot")}>
        <input
          className="mem-input set-grow"
          placeholder={t("settings.workspaceDefault")}
          value={root}
          disabled={busy}
          onChange={(e) => setRoot(e.target.value)}
          onBlur={() => root !== sb.workspaceRoot && void set({ workspaceRoot: root })}
        />
      </SettingsField>
      <RuleList
        list="allow_write"
        rules={sb.allowWrite}
        busy={busy}
        onAdd={(d) => set({ allowWrite: [...sb.allowWrite, d] })}
        onRemove={(d) => set({ allowWrite: sb.allowWrite.filter((x) => x !== d) })}
      />
    </SettingsSection>
  );
}

// Visual-style metadata for the appearance theme cards. The two surface
// swatches + accent are read from CSS variables at render time so they always
// reflect the live token values for the currently-resolved light/dark mode.
const THEME_STYLE_META: Record<ThemeStyle, { name: string; zh: DictKey; note: DictKey; desc: DictKey }> = {
  graphite: { name: "Graphite", zh: "settings.style.graphite.zh", note: "settings.style.graphite.note", desc: "settings.style.graphite.desc" },
  aurora: { name: "Aurora", zh: "settings.style.aurora.zh", note: "settings.style.aurora.note", desc: "settings.style.aurora.desc" },
  slate: { name: "Slate", zh: "settings.style.slate.zh", note: "settings.style.slate.note", desc: "settings.style.slate.desc" },
  carbon: { name: "Carbon", zh: "settings.style.carbon.zh", note: "settings.style.carbon.note", desc: "settings.style.carbon.desc" },
  nocturne: { name: "Nocturne", zh: "settings.style.nocturne.zh", note: "settings.style.nocturne.note", desc: "settings.style.nocturne.desc" },
  amber: { name: "Amber", zh: "settings.style.amber.zh", note: "settings.style.amber.note", desc: "settings.style.amber.desc" },
  hermes: { name: "Hermes", zh: "settings.style.hermes.zh" as DictKey, note: "settings.style.hermes.note" as DictKey, desc: "settings.style.hermes.desc" as DictKey },
};

function AppearanceSection({
  theme,
  themeStyle,
  textSize,
  fontFamily,
  monoFontFamily,
  customFontName,
  customMonoFontName,
  onTheme,
  onThemeStyle,
  onTextSize,
  onFontFamily,
  onMonoFontFamily,
  onCustomFontNameChange,
  onCustomMonoFontNameChange,
}: {
  theme: Theme;
  themeStyle: ThemeStyle;
  textSize: TextSize;
  fontFamily: FontFamily;
  monoFontFamily: MonoFontFamily;
  customFontName: string;
  customMonoFontName: string;
  onTheme: (t: Theme) => void;
  onThemeStyle: (style: ThemeStyle) => void;
  onTextSize: (size: TextSize) => void;
  onFontFamily: (font: FontFamily) => void;
  onMonoFontFamily: (font: MonoFontFamily) => void;
  onCustomFontNameChange: (name: string) => void;
  onCustomMonoFontNameChange: (name: string) => void;
}) {
  const t = useT();
  const themeOptions: Theme[] = ["auto", "light", "dark"];
  const availableFontFamilies = useMemo(() => getAvailableFontFamilies(fontFamily), [fontFamily]);
  const availableMonoFontFamilies = useMemo(() => getAvailableMonoFontFamilies(monoFontFamily), [monoFontFamily]);
  return (
    <SettingsSection title={t("settings.appearance")}>
      <SettingsField label={t("settings.theme")}>
        <div className="set-seg">
          {themeOptions.map((opt) => (
            <button
              key={opt}
              className={`set-seg__btn${theme === opt ? " set-seg__btn--on" : ""}`}
              onClick={() => onTheme(opt)}
            >
              {themeName(opt, t)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.themeStyle")} stacked>
        <div className="theme-card-grid">
          {THEME_STYLES.map((opt) => {
            const meta = THEME_STYLE_META[opt];
            const selected = themeStyle === opt;
            return (
              <button
                key={opt}
                type="button"
                role="radio"
                aria-checked={selected}
                className={`theme-card${selected ? " theme-card--on" : ""}`}
                onClick={() => onThemeStyle(opt)}
              >
                <span className="theme-card__head">
                  <span className="theme-card__name">
                    {meta.name} <span className="theme-card__zh">{t(meta.zh)}</span>
                  </span>
                  <span className="theme-card__tag">{t(meta.note)}</span>
                </span>
                <span className="theme-card__swatches" data-theme-style-card={opt}>
                  <span className="theme-card__swatch theme-card__swatch--bg" />
                  <span className="theme-card__swatch theme-card__swatch--surface" />
                  <span className="theme-card__swatch theme-card__swatch--accent" />
                </span>
                <span className="theme-card__desc">{t(meta.desc)}</span>
                <span className="theme-card__check" aria-hidden="true">
                  <Check size={13} strokeWidth={3} />
                </span>
              </button>
            );
          })}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.textSize")}>
        <div className="set-seg">
          {TEXT_SIZES.map((size) => (
            <button
              key={size}
              className={`set-seg__btn${textSize === size ? " set-seg__btn--on" : ""}`}
              onClick={() => onTextSize(size)}
            >
              {textSizeName(size, t)}
            </button>
          ))}
        </div>
      </SettingsField>
      <SettingsField label={t("settings.fontFamily")}>
        <div className="set-seg">
          {availableFontFamilies.map((font) => (
            <button
              key={font}
              className={`set-seg__btn${fontFamily === font ? " set-seg__btn--on" : ""}`}
              onClick={() => onFontFamily(font)}
            >
              {fontFamilyName(font, t)}
            </button>
          ))}
        </div>
      </SettingsField>
      {fontFamily === "custom" && (
        <SettingsField label={t("settings.fontFamilyCustomName")}>
          <textarea
            className="mem-input"
            style={{ width: "100%", resize: "vertical" }}
            rows={2}
            placeholder={t("settings.fontFamilyCustomPlaceholder")}
            value={customFontName}
            onChange={(e) => onCustomFontNameChange(e.target.value)}
          />
        </SettingsField>
      )}
      <SettingsField label={t("settings.monoFontFamily")}>
        <div className="set-seg">
          {availableMonoFontFamilies.map((font) => (
            <button
              key={font}
              className={`set-seg__btn${monoFontFamily === font ? " set-seg__btn--on" : ""}`}
              onClick={() => onMonoFontFamily(font)}
            >
              {monoFontFamilyName(font, t)}
            </button>
          ))}
        </div>
      </SettingsField>
      {monoFontFamily === "custom" && (
        <SettingsField label={t("settings.monoFontFamilyCustomName")}>
          <textarea
            className="mem-input"
            style={{ width: "100%", resize: "vertical" }}
            rows={2}
            placeholder={t("settings.monoFontFamilyCustomPlaceholder")}
            value={customMonoFontName}
            onChange={(e) => onCustomMonoFontNameChange(e.target.value)}
          />
        </SettingsField>
      )}
    </SettingsSection>
  );
}

function themeName(theme: Theme, t: ReturnType<typeof useT>): string {
  switch (theme) {
    case "auto":
      return t("settings.themeAuto");
    case "light":
      return t("settings.themeLight");
    case "dark":
      return t("settings.themeDark");
  }
}

function textSizeName(size: TextSize, t: ReturnType<typeof useT>): string {
  switch (size) {
    case "small":
      return t("settings.textSizeSmall");
    case "default":
      return t("settings.textSizeDefault");
    case "large":
      return t("settings.textSizeLarge");
    case "xlarge":
      return t("settings.textSizeXLarge");
    case "xxlarge":
      return t("settings.textSizeXXLarge");
  }
}

function fontFamilyName(font: FontFamily, t: ReturnType<typeof useT>): string {
  switch (font) {
    case "system":
      return t("settings.fontFamilySystem");
    case "yahei":
      return t("settings.fontFamilyYaHei");
    case "pingfang":
      return t("settings.fontFamilyPingFang");
    case "noto":
      return t("settings.fontFamilyNoto");
    case "custom":
      return t("settings.fontFamilyCustom");
  }
}

function monoFontFamilyName(font: MonoFontFamily, t: ReturnType<typeof useT>): string {
  switch (font) {
    case "system":
      return t("settings.monoFontFamilySystem");
    case "cascadia":
      return t("settings.monoFontFamilyCascadia");
    case "jetbrains":
      return t("settings.monoFontFamilyJetBrains");
    case "sfmono":
      return t("settings.monoFontFamilySFMono");
    case "custom":
      return t("settings.monoFontFamilyCustom");
  }
}

const MB = 1024 * 1024;
const mb = (n: number) => (n / MB).toFixed(1);

// UpdatesSection is the manual side of the auto-updater: it shows the startup
// check preference, running version, and a Check button, then the same state
// machine the top banner uses (useUpdater) — available → download → install, with
// progress and errors inline.
function UpdatesSection({
  configPath,
  checkUpdates,
  telemetry,
  metrics,
  settingsBusy,
  applySettings,
}: {
  configPath: string;
  checkUpdates: boolean;
  telemetry: boolean;
  metrics: boolean;
  settingsBusy: boolean;
  applySettings: (fn: () => Promise<void>) => Promise<void>;
}) {
  const t = useT();
  const { status, check, download: downloadUpdate, install: installUpdate } = useUpdater();
  const [version, setVersion] = useState("");
  useEffect(() => {
    app.Version().then(setVersion).catch((e) => { console.warn('SettingsPanel: version fetch failed', e) });
  }, []);

  const updaterBusy =
    status.kind === "checking" || status.kind === "downloading" || status.kind === "verifying" || status.kind === "installing";

  return (
    <SettingsSection title={t("updater.title")}>
      <SettingsField
        className="settings-field--wide-copy"
        label={t("updater.autoCheckLabel")}
        hint={t("updater.autoCheckHint")}
      >
        <ToggleSegment
          value={checkUpdates}
          disabled={settingsBusy}
          onChange={(enabled) => void applySettings(() => app.SetDesktopCheckUpdates(enabled))}
        />
      </SettingsField>
      <SettingsField
        className="settings-field--wide-copy"
        label={t("settings.telemetryLabel")}
        hint={t("settings.telemetryHint")}
      >
        <ToggleSegment
          value={telemetry}
          disabled={settingsBusy}
          onChange={(enabled) => void applySettings(() => app.SetDesktopTelemetry(enabled))}
        />
      </SettingsField>
      <SettingsField
        className="settings-field--wide-copy"
        label={t("settings.metricsLabel")}
        hint={t("settings.metricsHint")}
      >
        <ToggleSegment
          value={metrics}
          disabled={settingsBusy}
          onChange={(enabled) => void applySettings(() => app.SetDesktopMetrics(enabled))}
        />
      </SettingsField>
      <SettingsField label={t("updater.currentVersion", { v: version || "…" })}>
        <button className="btn btn--small" disabled={updaterBusy} onClick={() => void check()}>
          {status.kind === "checking" ? t("updater.checking") : t("updater.checkButton")}
        </button>
      </SettingsField>
      {status.kind === "available" && (
        <div className="mem-hint">{t("updater.channelLabel", { channel: status.info.channel || "stable" })}</div>
      )}
      {status.kind === "upToDate" && <div className="mem-hint">{t("updater.upToDate")}</div>}
      {status.kind === "available" && (
        <>
          <SettingsField label={t("updater.available", { v: status.info.latest })}>
            <button className="btn btn--primary btn--small" onClick={() => downloadUpdate(status.info)}>
              {status.info.canSelfUpdate ? t("updater.downloadUpdate") : t("updater.goToDownload")}
            </button>
          </SettingsField>
          {!status.info.canSelfUpdate && <div className="mem-hint">{status.info.manualReason || t("updater.macHint")}</div>}
        </>
      )}
      {status.kind === "downloading" && (
        <div className="mem-hint">
          {t("updater.downloading", {
            done: mb(status.received),
            total: mb(status.total),
            pct: status.total > 0 ? Math.round((status.received / status.total) * 100) : 0,
          })}
        </div>
      )}
      {status.kind === "verifying" && <div className="mem-hint">{t("updater.verifying")}</div>}
      {status.kind === "downloaded" && (
        <SettingsField label={t("updater.downloaded", { v: status.info.latest })}>
          <button className="btn btn--primary btn--small" onClick={installUpdate}>
            {t("updater.restartInstall")}
          </button>
        </SettingsField>
      )}
      {status.kind === "installing" && <div className="mem-hint">{t("updater.installing")}</div>}
      {status.kind === "done" && <div className="mem-hint">{t("updater.done")}</div>}
      {status.kind === "error" && <div className="banner banner--error">{t("updater.failed", { msg: status.message })}</div>}
      {configPath && (
        <Tooltip label={configPath} fill block className="mem-hint settings-config-path">
          {t("settings.config", { path: configPath })}
        </Tooltip>
      )}
    </SettingsSection>
  );
}

// ── HermesLiveSection ──────────────────────────────────────────────────────

function HermesLiveSection({
  s,
  apply,
}: { s: SettingsView; apply: (fn: () => Promise<unknown>) => Promise<void> }) {
  const live = useHermesLiveData(undefined, true);
  return (
    <SettingsPageShell
      s={s}
      tab="hermes"
      busy={false}
      apply={async (fn) => {
        await apply(fn);
      }}
    >
      <HermesSettings
        s={s}
        onHotbarChange={() => {}}
        onProfileSelect={() => {}}
        cache={live.cache}
        bot={live.bot}
        goal={live.goal}
        cost={live.cost}
        tokens={live.tokens}
        compress={live.compress}
        schedule={live.schedule}
        collab={live.collab}
        council={live.council}
      />
    </SettingsPageShell>
  );
}
