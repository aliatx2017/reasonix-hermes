import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { CheckCircle2, ChevronDown, Clipboard, KeyRound, Loader2, QrCode, RefreshCw, Send } from "lucide-react";
import { asArray } from "../lib/array";
import { app } from "../lib/bridge";
import { useT, type DictKey } from "../lib/i18n";
import type { BotAllowlistView, BotConnectionDiagnostic, BotConnectionView, BotInstallStartResult, BotSettingsView } from "../lib/types";
import { InlineConfirmButton } from "./InlineConfirmButton";
import { allRefs, toRef, uniqueStrings, ToggleSegment, SettingsField, type SectionProps, type SettingsInitialFocus } from "./settings-shared";
import { ModelPicker, settingsModelMeta } from "./ModelsSection";

const QRCodeSVG = lazy(() => import("qrcode.react").then((module) => ({ default: module.QRCodeSVG })));

// ── Bot settings helpers ──────────────────────────────────────────────────

export function botSettingsMeta(bot: BotSettingsView, t: ReturnType<typeof useT>): string {
  const normalized = normalizeBotSettings(bot);
  const connections = normalized.connections.length + (qqBotAdded(normalized.qq) ? 1 : 0);
  if (connections === 0) return t("settings.botNoConnections");
  if (!normalized.enabled) return t("settings.botDisabledWithConnections", { n: connections });
  return t("settings.botConnectionCount", { n: connections });
}

export const BOT_TOOL_APPROVAL_MODES = ["", "ask", "auto", "yolo"] as const;
export type BotConnectionToolApprovalMode = (typeof BOT_TOOL_APPROVAL_MODES)[number];

export function defaultBotSettings(): BotSettingsView {
  return {
    enabled: false,
    model: "",
    toolApprovalMode: "ask",
    maxSteps: 0,
    debounceMs: 1500,
    allowlist: {
      enabled: true,
      allowAll: false,
      qqUsers: [],
      feishuUsers: [],
      weixinUsers: [],
      qqGroups: [],
      feishuGroups: [],
      weixinGroups: [],
    },
    qq: { enabled: false, appId: "", appSecretEnv: "QQ_BOT_APP_SECRET", secretSet: false, sandbox: false },
    feishu: {
      enabled: false,
      domain: "feishu",
      appId: "",
      appSecretEnv: "FEISHU_BOT_APP_SECRET",
      secretSet: false,
      verificationToken: "",
      mode: "webhook",
      webhookPort: 8080,
      requireMention: true,
    },
    weixin: {
      enabled: false,
      accountId: "default",
      tokenEnv: "WEIXIN_BOT_TOKEN",
      tokenSet: false,
      apiBase: "https://ilinkai.weixin.qq.com",
    },
    connections: [],
  };
}


export function normalizeBotSettings(bot: BotSettingsView | null | undefined): BotSettingsView {
  const fallback = defaultBotSettings();
  const allowlist = bot?.allowlist ?? fallback.allowlist;
  const mode = bot?.feishu?.mode === "websocket" ? "websocket" : "webhook";
  return {
    ...fallback,
    ...bot,
    toolApprovalMode: normalizeBotToolApprovalMode(bot?.toolApprovalMode),
    maxSteps: Math.max(0, Number(bot?.maxSteps ?? fallback.maxSteps) || 0),
    debounceMs: Number(bot?.debounceMs) || fallback.debounceMs,
    allowlist: {
      ...fallback.allowlist,
      ...allowlist,
      qqUsers: asArray(allowlist.qqUsers),
      feishuUsers: asArray(allowlist.feishuUsers),
      weixinUsers: asArray(allowlist.weixinUsers),
      qqGroups: asArray(allowlist.qqGroups),
      feishuGroups: asArray(allowlist.feishuGroups),
      weixinGroups: asArray(allowlist.weixinGroups),
    },
    qq: { ...fallback.qq, ...bot?.qq },
    feishu: { ...fallback.feishu, ...bot?.feishu, domain: bot?.feishu?.domain === "lark" ? "lark" : "feishu", mode },
    weixin: { ...fallback.weixin, ...bot?.weixin },
    connections: asArray(bot?.connections).map(normalizeBotConnection),
  };
}

export function normalizeBotConnection(raw: any) {
  const credential = raw?.credential ?? {};
  const workspaceRoot = String(raw?.workspaceRoot ?? "").trim();
  return {
    id: String(raw?.id ?? "").trim(),
    provider: String(raw?.provider ?? "").trim(),
    domain: String(raw?.domain ?? "").trim(),
    label: String(raw?.label ?? "").trim(),
    enabled: raw?.enabled !== false,
    status: String(raw?.status ?? "disconnected").trim(),
    model: String(raw?.model ?? "").trim(),
    toolApprovalMode: normalizeBotToolApprovalMode(raw?.toolApprovalMode, true),
    workspaceRoot,
    credential: {
      appId: String(credential.appId ?? "").trim(),
      appSecretEnv: String(credential.appSecretEnv ?? "").trim(),
      accountId: String(credential.accountId ?? "").trim(),
      tokenEnv: String(credential.tokenEnv ?? "").trim(),
      secretSet: Boolean(credential.secretSet),
    },
    sessionMappings: asArray(raw?.sessionMappings).map((item: any) => ({
      remoteId: String(item?.remoteId ?? "").trim(),
      sessionId: String(item?.sessionId ?? "").trim(),
      sessionSource: String(item?.sessionSource ?? "").trim(),
      chatType: String(item?.chatType ?? "").trim(),
      userId: String(item?.userId ?? "").trim(),
      threadId: String(item?.threadId ?? "").trim(),
      scope: normalizeBotMappingScope(item?.scope, item?.workspaceRoot ?? workspaceRoot),
      workspaceRoot: normalizeBotMappingScope(item?.scope, item?.workspaceRoot ?? workspaceRoot) === "project"
        ? String(item?.workspaceRoot ?? workspaceRoot).trim()
        : "",
      updatedAt: String(item?.updatedAt ?? "").trim(),
    })),
    lastError: String(raw?.lastError ?? "").trim(),
    createdAt: String(raw?.createdAt ?? "").trim(),
    updatedAt: String(raw?.updatedAt ?? "").trim(),
  };
}


export function normalizeBotToolApprovalMode(mode: unknown, allowEmpty = false): "ask" | "auto" | "yolo" | "" {
  const raw = String(mode ?? "").trim().toLowerCase();
  if (raw === "") return allowEmpty ? "" : "ask";
  if (raw === "ask") return "ask";
  if (raw === "auto") return "auto";
  if (raw === "yolo" || raw === "full" || raw === "full-access" || raw === "bypass") return "yolo";
  return allowEmpty ? "" : "ask";
}

export function normalizeBotMappingScope(scope: unknown, workspaceRoot: unknown): "global" | "project" {
  if (String(scope ?? "").trim() === "project") return "project";
  return String(workspaceRoot ?? "").trim() ? "project" : "global";
}

type BotInstallTarget = "qq" | "feishu" | "lark" | "weixin";
export type BotOfficialInstallTarget = Exclude<BotInstallTarget, "qq">;

const BOT_ALLOWLIST_TEXT_KEYS = ["qqUsers", "feishuUsers", "weixinUsers", "qqGroups", "feishuGroups", "weixinGroups"] as const;
function botAllowlistTextValues(allowlist: BotAllowlistView): Record<BotAllowlistTextKey, string> {
  return {
    qqUsers: allowlist.qqUsers.join("\n"),
    feishuUsers: allowlist.feishuUsers.join("\n"),
    weixinUsers: allowlist.weixinUsers.join("\n"),
    qqGroups: allowlist.qqGroups.join("\n"),
    feishuGroups: allowlist.feishuGroups.join("\n"),
    weixinGroups: allowlist.weixinGroups.join("\n"),
  };
}

function parseBotListInput(value: string): string[] {
  return uniqueStrings(value
    .split(/[\n,，]+/)
    .map((entry) => entry.trim())
    .filter(Boolean));
}

type BotAllowlistTextKey = typeof BOT_ALLOWLIST_TEXT_KEYS[number];
type BotInstallState = {
  target: BotInstallTarget | "";
  result: BotInstallStartResult | null;
  status: "idle" | "starting" | "showing" | "connected" | "error";
  timeLeft: number;
  message: string;
};
const BOT_INSTALL_TARGETS: BotInstallTarget[] = ["qq", "feishu", "lark", "weixin"];
const BOT_INSTALL_DEFAULT_TIMEOUT_SECONDS = 300;
const BOT_INSTALL_MIN_POLL_SECONDS = 3;
const DEFAULT_QQ_SECRET_ENV = "QQ_BOT_APP_SECRET";
const QQ_CONNECTION_ID = "__qq_bot__";

type BotConnectionListItem =
  | { kind: "qq" }
  | { kind: "connection"; connection: BotConnectionView };

export type BotsSectionProps = SectionProps & { initialFocus?: SettingsInitialFocus };

export function BotsSection({ s, busy, apply, initialFocus }: BotsSectionProps) {
  const t = useT();
  const savedBot = normalizeBotSettings(s.bot);
  const [draft, setDraft] = useState<BotSettingsView>(savedBot);
  const [allowlistText, setAllowlistText] = useState<Record<BotAllowlistTextKey, string>>(() => botAllowlistTextValues(savedBot.allowlist));
  const [allowlistFocused, setAllowlistFocused] = useState(false);
  const [allowlistOpen, setAllowlistOpen] = useState(false);
  const [installTarget, setInstallTarget] = useState<BotInstallTarget>("qq");
  const [install, setInstall] = useState<BotInstallState>({ target: "qq", result: null, status: "idle", timeLeft: 0, message: "" });
  const [diagnostics, setDiagnostics] = useState<Record<string, BotConnectionDiagnostic | string>>({});
  const [testTargets, setTestTargets] = useState<Record<string, string>>({});
  const [connectionSecrets, setConnectionSecrets] = useState<Record<string, string>>({});
  const [qqSecretValue, setQQSecretValue] = useState("");
  const [expandedConnectionId, setExpandedConnectionId] = useState("");
  const installRef = useRef(install);
  const installPollTimerRef = useRef<number | null>(null);
  const installCountdownTimerRef = useRef<number | null>(null);
  const installRequestInFlightRef = useRef(false);
  const installAttemptRef = useRef(0);
  const allowlistRef = useRef<HTMLDetailsElement | null>(null);
  const initialFocusHandledRef = useRef("");
  const pendingAllowlistFocusRef = useRef(false);
  const refs = allRefs(s);

  useEffect(() => {
    const nextBot = normalizeBotSettings(s.bot);
    setDraft(nextBot);
    setAllowlistText(botAllowlistTextValues(nextBot.allowlist));
    setConnectionSecrets({});
    setQQSecretValue("");
    setTestTargets({});
  }, [s.bot]);
  useEffect(() => {
    if (initialFocus?.target !== "bot-allowlist") return;
    const focusKey = `${initialFocus.target}:${initialFocus.connectionId ?? ""}`;
    if (initialFocusHandledRef.current === focusKey) return;
    let focusConnectionId = "";
    if (initialFocus.connectionId === QQ_CONNECTION_ID && qqBotAdded(draft.qq)) {
      focusConnectionId = QQ_CONNECTION_ID;
    } else if (initialFocus.connectionId && draft.connections.some((connection) => connection.id === initialFocus.connectionId)) {
      focusConnectionId = initialFocus.connectionId;
    } else {
      focusConnectionId = draft.connections[0]?.id ?? "";
    }
    if (!focusConnectionId) return;
    initialFocusHandledRef.current = focusKey;
    pendingAllowlistFocusRef.current = true;
    setExpandedConnectionId(focusConnectionId);
    setAllowlistOpen(false);
  }, [draft.connections, draft.qq, initialFocus]);
  useEffect(() => {
    setAllowlistOpen(false);
  }, [expandedConnectionId]);
  useEffect(() => {
    installRef.current = install;
  }, [install]);
  useEffect(() => {
    installAttemptRef.current += 1;
    installRequestInFlightRef.current = false;
    clearInstallTimers();
    setInstall({ target: installTarget, result: null, status: "idle", timeLeft: 0, message: "" });
  }, [installTarget]);
  useEffect(() => () => {
    installAttemptRef.current += 1;
    clearInstallTimers();
  }, []);

  const setConnections = (mapper: (connections: BotConnectionView[]) => BotConnectionView[]) =>
    setDraft((prev) => ({ ...prev, connections: mapper(prev.connections) }));
  const persistBotDraft = async (nextDraft: BotSettingsView) => {
    const nextBot = botDraftWithDerivedGatewayState(nextDraft);
    setDraft(nextBot);
    await apply(async () => {
      await app.SetBotSettings(nextBot);
    });
  };
  const persistConnections = (mapper: (connections: BotConnectionView[]) => BotConnectionView[]) =>
    persistBotDraft({ ...draft, connections: mapper(draft.connections) });
  const updateConnection = (id: string, patch: Partial<BotConnectionView>) =>
    setConnections((items) => items.map((item) => item.id === id ? { ...item, ...patch } : item));
  const persistConnection = (id: string, patch: Partial<BotConnectionView>) =>
    persistConnections((items) => items.map((item) => item.id === id ? { ...item, ...patch } : item));
  const updateConnectionCredential = (id: string, patch: Partial<BotConnectionView["credential"]>) =>
    setConnections((items) => items.map((item) => item.id === id ? { ...item, credential: { ...item.credential, ...patch } } : item));
  const persistConnectionCredential = (id: string, patch: Partial<BotConnectionView["credential"]>) =>
    persistConnections((items) => items.map((item) => item.id === id ? { ...item, credential: { ...item.credential, ...patch } } : item));
  const updateAllowlist = (patch: Partial<BotAllowlistView>) =>
    setDraft((prev) => ({ ...prev, allowlist: { ...prev.allowlist, ...patch } }));
  const persistAllowlist = (patch: Partial<BotAllowlistView>) =>
    persistBotDraft({ ...draft, allowlist: { ...draft.allowlist, ...patch } });
  const persistAllowlistText = (key: BotAllowlistTextKey, value: string) => {
    const entries = parseBotListInput(value);
    setAllowlistText((prev) => ({ ...prev, [key]: entries.join("\n") }));
    void persistAllowlist({ [key]: entries } as Partial<BotAllowlistView>);
  };
  const updateQQ = (patch: Partial<BotSettingsView["qq"]>) =>
    setDraft((prev) => ({ ...prev, qq: { ...prev.qq, ...patch } }));
  const persistQQ = (patch: Partial<BotSettingsView["qq"]>) =>
    persistBotDraft({ ...draft, qq: { ...draft.qq, ...patch } });
  const removeConnection = async (connection: BotConnectionView) => {
    const nextDraft = botDraftWithDerivedGatewayState({
      ...draft,
      connections: draft.connections.filter((item) => item.id !== connection.id),
    });
    await apply(async () => {
      await app.SetBotSettings(nextDraft);
    });
  };
  const installQrURL = install.result?.url ?? "";
  const installQrIsImage = installQrURL.startsWith("data:image/");
  const isQQInstallTarget = installTarget === "qq";
  const selectedInstallConnection = isQQInstallTarget ? undefined : draft.connections.find((connection) => botInstallTargetMatchesConnection(installTarget, connection));
  const selectedInstallLabel = botTargetLabel(installTarget, t);
  const installUserCode = install.result?.userCode && installTarget !== "weixin" ? formatInstallUserCode(install.result.userCode) : "";
  const qqSecretEnv = draft.qq.appSecretEnv.trim() || DEFAULT_QQ_SECRET_ENV;
  const qqConfigured = draft.qq.enabled && draft.qq.appId.trim() && qqSecretEnv && draft.qq.secretSet;
  const qqCanEnableAccess = qqAccessReady(draft.allowlist);
  const qqCanSaveAndEnable = Boolean(draft.qq.appId.trim() && qqSecretEnv && (draft.qq.secretSet || qqSecretValue.trim()) && qqCanEnableAccess);
  const qqAdded = qqBotAdded(draft.qq);
  const nativeRuntimeAvailable = typeof window !== "undefined" && Boolean(window.runtime);
  const browserPreviewBotConfigured = !nativeRuntimeAvailable && (qqAdded || draft.connections.length > 0);
  const qqOnline = qqConfigured && nativeRuntimeAvailable;
  const connectionItems: BotConnectionListItem[] = [
    ...(qqAdded ? [{ kind: "qq" as const }] : []),
    ...draft.connections.map((connection) => ({ kind: "connection" as const, connection })),
  ];

  const saveBot = () => app.SetBotSettings(botDraftWithDerivedGatewayState(draft));
  function clearInstallTimers() {
    if (installPollTimerRef.current !== null) {
      window.clearTimeout(installPollTimerRef.current);
      installPollTimerRef.current = null;
    }
    if (installCountdownTimerRef.current !== null) {
      window.clearInterval(installCountdownTimerRef.current);
      installCountdownTimerRef.current = null;
    }
  }
  function beginInstallCountdown(attempt: number) {
    if (installCountdownTimerRef.current !== null) {
      window.clearInterval(installCountdownTimerRef.current);
    }
    installCountdownTimerRef.current = window.setInterval(() => {
      setInstall((prev) => {
        if (installAttemptRef.current !== attempt || prev.status !== "showing") return prev;
        return { ...prev, timeLeft: Math.max(0, prev.timeLeft - 1) };
      });
    }, 1000);
  }
  function scheduleInstallPoll(attempt: number, interval: number) {
    if (installPollTimerRef.current !== null) {
      window.clearTimeout(installPollTimerRef.current);
    }
    installPollTimerRef.current = window.setTimeout(() => void pollInstall(attempt), Math.max(interval || BOT_INSTALL_MIN_POLL_SECONDS, BOT_INSTALL_MIN_POLL_SECONDS) * 1000);
  }
  const startInstall = async (target: BotOfficialInstallTarget) => {
    if (installRequestInFlightRef.current) return;
    const existing = draft.connections.find((connection) => botInstallTargetMatchesConnection(target, connection));
    if (existing) {
      installAttemptRef.current += 1;
      clearInstallTimers();
      setInstall({ target, result: null, status: "connected", timeLeft: 0, message: t("settings.botInstallAlreadyConnected", { provider: botTargetLabel(target, t) }) });
      return;
    }
    clearInstallTimers();
    const attempt = installAttemptRef.current + 1;
    installAttemptRef.current = attempt;
    installRequestInFlightRef.current = true;
    setInstall({ target, result: null, status: "starting", timeLeft: 0, message: t("settings.botInstallStarting") });
    const provider = target === "weixin" ? "weixin" : "feishu";
    const domain = target === "lark" ? "lark" : target === "weixin" ? "weixin" : "feishu";
    try {
      const result = await app.StartBotConnectionInstall(provider, domain);
      if (installAttemptRef.current !== attempt) return;
      if (!result.ok) {
        setInstall({ target, result, status: "error", timeLeft: 0, message: result.message || t("settings.botInstallFailed") });
        return;
      }
      const timeLeft = result.expireIn > 0 ? result.expireIn : BOT_INSTALL_DEFAULT_TIMEOUT_SECONDS;
      setInstall({ target, result, status: "showing", timeLeft, message: result.message || t("settings.botInstallScanHint") });
      beginInstallCountdown(attempt);
      scheduleInstallPoll(attempt, result.interval);
    } catch (err) {
      if (installAttemptRef.current === attempt) {
        setInstall({ target, result: null, status: "error", timeLeft: 0, message: err instanceof Error ? err.message : t("settings.botInstallFailed") });
      }
    } finally {
      if (installAttemptRef.current === attempt) {
        installRequestInFlightRef.current = false;
      }
    }
  };
  const pollInstall = async (attempt = installAttemptRef.current) => {
    const current = installRef.current;
    if (installAttemptRef.current !== attempt || current.status !== "showing" || !current.result?.installId || !current.target) return;
    const poll = await app.PollBotConnectionInstall(current.result.installId);
    if (installAttemptRef.current !== attempt) return;
    if (poll.done) {
      clearInstallTimers();
      setDraft((prev) => ({
        ...prev,
        enabled: true,
        connections: [...prev.connections.filter((c) => c.id !== poll.connection.id), poll.connection],
      }));
      setInstall((prev) => ({ ...prev, status: "connected", timeLeft: 0, message: poll.message || t("settings.botInstallConnected") }));
      return;
    }
    if (poll.error) {
      clearInstallTimers();
      setInstall((prev) => ({ ...prev, status: "error", timeLeft: 0, message: poll.error }));
      return;
    }
    setInstall((prev) => ({ ...prev, message: poll.message || t("settings.botInstallWaiting") }));
    scheduleInstallPoll(attempt, current.result.interval);
  };
  useEffect(() => {
    if (install.status !== "showing" || install.timeLeft > 0) return;
    installAttemptRef.current += 1;
    clearInstallTimers();
    setInstall((prev) => prev.status === "showing" ? { ...prev, status: "error", message: t("settings.botInstallExpired") } : prev);
  }, [install.status, install.timeLeft]);
  const diagnoseConnection = async (id: string) => {
    const diag = await app.DiagnoseBotConnection(id);
    setDiagnostics((prev) => ({ ...prev, [id]: diag }));
    return diag;
  };
  const testConnection = async (connection: BotConnectionView) => {
    const target = (testTargets[connection.id] ?? firstConnectionRemote(connection)).trim();
    const diag = await app.TestBotConnection(connection.id, target);
    setDiagnostics((prev) => ({ ...prev, [connection.id]: diag }));
    if (diag.messageId && target) {
      const updatedAt = new Date().toISOString();
      await persistConnections((items) => items.map((item) => {
        if (item.id !== connection.id) return item;
        const scope = connection.workspaceRoot ? "project" : "global";
        const matchesTestMapping = (mapping: BotConnectionView["sessionMappings"][number]) =>
          mapping.remoteId === target &&
          !mapping.chatType.trim() &&
          !mapping.userId.trim() &&
          !mapping.threadId.trim();
        const sessionMappings = [
          ...item.sessionMappings.filter((mapping) => !matchesTestMapping(mapping)),
          { remoteId: target, sessionId: "", sessionSource: "", chatType: "", userId: "", threadId: "", scope, workspaceRoot: scope === "project" ? connection.workspaceRoot : "", updatedAt },
        ];
        return { ...item, sessionMappings, updatedAt };
      }));
    }
  };
  const ensureReportableDiagnostic = async (connection: BotConnectionView) => {
    return diagnoseConnection(connection.id);
  };
  const copyConnectionDiagnostic = async (connection: BotConnectionView) => {
    const diag = await ensureReportableDiagnostic(connection);
    if (!diag.reportDetail) return;
    try {
      await navigator.clipboard.writeText(diag.reportDetail);
      setDiagnostics((prev) => ({ ...prev, [connection.id]: { ...diag, message: t("settings.botDiagnosticCopied") } }));
    } catch (err) {
      setDiagnostics((prev) => ({
        ...prev,
        [connection.id]: { ...diag, status: "error", message: err instanceof Error ? err.message : t("settings.botDiagnosticCopyFailed") },
      }));
    }
  };
  const reportConnectionDiagnostic = async (connection: BotConnectionView) => {
    const diag = await ensureReportableDiagnostic(connection);
    if (!diag.reportDetail) return;
    try {
      await app.ReportCrash(diag.reportKind || "bot", diag.reportDetail);
      setDiagnostics((prev) => ({ ...prev, [connection.id]: { ...diag, status: "ok", message: t("settings.botDiagnosticReportSent") } }));
    } catch (err) {
      setDiagnostics((prev) => ({
        ...prev,
        [connection.id]: { ...diag, status: "error", message: err instanceof Error ? err.message : t("settings.botDiagnosticReportFailed") },
      }));
    }
  };
  const saveConnectionSecret = async (connection: BotConnectionView) => {
    const env = botConnectionSecretEnv(connection).trim();
    const value = (connectionSecrets[connection.id] ?? "").trim();
    if (!env || !value) return;
    await apply(async () => {
      await saveBot();
      await app.SetBotSecret(env, value);
    });
    setConnectionSecrets((prev) => ({ ...prev, [connection.id]: "" }));
  };
  const clearConnectionSecret = async (connection: BotConnectionView) => {
    const env = botConnectionSecretEnv(connection).trim();
    if (!env) return;
    await apply(async () => {
      await saveBot();
      await app.ClearBotSecret(env);
    });
  };
  const clearQQSecret = async () => {
    const env = draft.qq.appSecretEnv.trim() || DEFAULT_QQ_SECRET_ENV;
    if (!env) return;
    await apply(async () => {
      await saveBot();
      await app.ClearBotSecret(env);
    });
    setQQSecretValue("");
  };
  const focusQQAccessSettings = () => {
    pendingAllowlistFocusRef.current = true;
    setExpandedConnectionId(QQ_CONNECTION_ID);
    setAllowlistOpen(true);
    setAllowlistFocused(true);
    setDiagnostics((prev) => ({ ...prev, [QQ_CONNECTION_ID]: t("settings.botQQAccessRequired") }));
  };
  const saveQQAndEnable = async () => {
    if (!qqCanEnableAccess) {
      focusQQAccessSettings();
      return;
    }
    const env = draft.qq.appSecretEnv.trim() || DEFAULT_QQ_SECRET_ENV;
    const secret = qqSecretValue.trim();
    const nextDraft = botDraftWithDerivedGatewayState({
      ...draft,
      qq: {
        ...draft.qq,
        enabled: true,
        appId: draft.qq.appId.trim(),
        appSecretEnv: env,
        secretSet: draft.qq.secretSet || Boolean(secret),
      },
    });
    await apply(async () => {
      await app.SetBotSettings(nextDraft);
      if (secret) await app.SetBotSecret(env, secret);
    });
    setDraft(nextDraft);
    setQQSecretValue("");
  };
  const removeQQBot = async () => {
    const env = draft.qq.appSecretEnv.trim() || DEFAULT_QQ_SECRET_ENV;
    const nextDraft = botDraftWithDerivedGatewayState({
      ...draft,
      qq: { enabled: false, appId: "", appSecretEnv: DEFAULT_QQ_SECRET_ENV, secretSet: false, sandbox: false },
    });
    await apply(async () => {
      await app.SetBotSettings(nextDraft);
      if (draft.qq.secretSet) await app.ClearBotSecret(env);
    });
    setDraft(nextDraft);
    setQQSecretValue("");
    setExpandedConnectionId("");
  };
  const onlineConnections = (qqOnline ? 1 : 0) + draft.connections.filter((connection) => connection.enabled && connection.status === "connected").length;
  const selectedQQ = qqAdded && expandedConnectionId === QQ_CONNECTION_ID;
  const selectedConnection = selectedQQ ? null : draft.connections.find((connection) => connection.id === expandedConnectionId) ?? null;
  const selectedDiagnostic = selectedConnection ? diagnostics[selectedConnection.id] : undefined;
  const selectedDiagnosticDetail = diagnosticReportDetail(selectedDiagnostic);
  const selectedConnectionRemote = selectedConnection ? firstConnectionRemote(selectedConnection) : "";
  const selectedConnectionToolApprovalMode = selectedConnection ? normalizeBotToolApprovalMode(selectedConnection.toolApprovalMode, true) : "";
  const selectedAllowlistTargetReady = selectedQQ || Boolean(selectedConnection);
  useEffect(() => {
    if (!pendingAllowlistFocusRef.current || !selectedAllowlistTargetReady) return;
    setAllowlistOpen(true);
    const scrollTimer = window.setTimeout(() => {
      if (!allowlistRef.current) return;
      pendingAllowlistFocusRef.current = false;
      allowlistRef.current.scrollIntoView({ block: "center", behavior: "smooth" });
      setAllowlistFocused(true);
    }, 80);
    const clearTimer = window.setTimeout(() => setAllowlistFocused(false), 2100);
    return () => {
      window.clearTimeout(scrollTimer);
      window.clearTimeout(clearTimer);
    };
  }, [selectedAllowlistTargetReady]);

  return (
    <div className="bot-phone-connect">
        <div className="bot-connection-list">
          <div className="bot-connection-list__head">
            <div className="bot-connection-list__title">
              <strong>{t("settings.botConnectedBots")}</strong>
              <span>{t("settings.botConnectedBotsSummary", { online: onlineConnections, total: connectionItems.length })}</span>
            </div>
          </div>
          {browserPreviewBotConfigured ? (
            <div className="bot-connection-warning">{t("settings.botBrowserPreviewWarning")}</div>
          ) : null}
          {connectionItems.length === 0 ? (
            <div className="bot-connection-empty">{t("settings.botConnectionsEmpty")}</div>
          ) : (
            <div className="bot-connection-table" role="table" aria-label={t("settings.botConnectedBots")}>
              <div className="bot-connection-table__header" role="row">
                <span>{t("settings.botConnectionColumnChannel")}</span>
                <span>{t("settings.botConnectionColumnName")}</span>
                <span>{t("settings.botConnectionColumnStatus")}</span>
                <span>{t("settings.botConnectionColumnActions")}</span>
              </div>
              {connectionItems.map((item) => {
                if (item.kind === "qq") {
                  const appID = draft.qq.appId.trim();
                  const qqDiagMessage = diagnosticMessage(diagnostics[QQ_CONNECTION_ID]);
                  const statusText = qqOnline
                    ? t("settings.botConnectionConnected")
                    : qqConfigured
                      ? t("settings.botConnectionConfigured")
                      : draft.qq.secretSet
                      ? t("settings.botConnectionDisconnected")
                      : t("settings.botSecretMissing");
                  return (
                    <div key={QQ_CONNECTION_ID} className="bot-connection-row" role="rowgroup">
                      <div className="bot-connection-row__grid" role="row">
                      <div className="bot-connection-row__channel" role="cell">
                        <span>QQ</span>
                      </div>
                      <div className="bot-connection-row__identity-cell" role="cell">
                        <button
                          type="button"
                          className="bot-connection-identity"
                          disabled={busy}
                          onClick={() => setExpandedConnectionId((current) => current === QQ_CONNECTION_ID ? "" : QQ_CONNECTION_ID)}
                          title={appID || "QQ Bot"}
                        >
                          <span className="bot-connection-identity__main">
                            <strong>QQ Bot</strong>
                            <code>{appID || "—"}</code>
                          </span>
                        </button>
                      </div>
                      <div className="bot-connection-row__state" role="cell">
                        <span className={`bot-connection-row__status bot-connection-row__status--${qqOnline ? "connected" : qqConfigured ? "configured" : "disconnected"}`}>
                            {statusText}
                          </span>
                          <ToggleSegment
                            value={draft.qq.enabled}
                            disabled={busy}
                            onChange={(enabled) => {
                              if (enabled && !qqCanEnableAccess) {
                                focusQQAccessSettings();
                                return;
                              }
                              updateQQ({ enabled });
                              void persistQQ({ enabled });
                            }}
                          />
                        </div>
                        <div className="bot-connection-row__actions" role="cell">
                          <button
                            type="button"
                            className={`btn btn--small${selectedQQ ? " btn--primary" : " btn--secondary"}`}
                            disabled={busy}
                            onClick={() => setExpandedConnectionId((current) => current === QQ_CONNECTION_ID ? "" : QQ_CONNECTION_ID)}
                          >
                            {t("settings.botManage")}
                          </button>
                        </div>
                      </div>
                      {qqDiagMessage ? <em className="bot-connection-row__diag">{qqDiagMessage}</em> : null}
                    </div>
                  );
                }
                const connection = item.connection;
                const sessionID = firstConnectionRemote(connection);
                const diagMessage = diagnosticMessage(diagnostics[connection.id]);
                const connectionStatusClass = connection.status === "connected" ? "connected" : "disconnected";
                return (
                  <div key={connection.id} className="bot-connection-row" role="rowgroup">
                    <div className="bot-connection-row__grid" role="row">
                      <div className="bot-connection-row__channel" role="cell">
                        <span>{botConnectionLabel(connection, t)}</span>
                      </div>
                      <div className="bot-connection-row__identity-cell" role="cell">
                        <button
                          type="button"
                          className="bot-connection-identity"
                          disabled={busy}
                          onClick={() => setExpandedConnectionId((current) => current === connection.id ? "" : connection.id)}
                          title={sessionID || connection.label || botConnectionLabel(connection, t)}
                        >
                          <span className="bot-connection-identity__main">
                            <strong>{connection.label || botConnectionLabel(connection, t)}</strong>
                            <code>{sessionID || "—"}</code>
                          </span>
                        </button>
                      </div>
                      <div className="bot-connection-row__state" role="cell">
                        <span className={`bot-connection-row__status bot-connection-row__status--${connectionStatusClass}`}>
                          {connection.status === "connected" ? t("settings.botConnectionConnected") : connection.status || t("settings.botConnectionDisconnected")}
                        </span>
                        <ToggleSegment
                          value={connection.enabled}
                          disabled={busy}
                          onChange={(enabled) => void persistConnection(connection.id, { enabled })}
                        />
                      </div>
                      <div className="bot-connection-row__actions" role="cell">
                        <button
                          type="button"
                          className={`btn btn--small${expandedConnectionId === connection.id ? " btn--primary" : " btn--secondary"}`}
                          disabled={busy}
                          onClick={() => setExpandedConnectionId((current) => current === connection.id ? "" : connection.id)}
                        >
                          {t("settings.botManage")}
                        </button>
                      </div>
                    </div>
                    {diagMessage ? <em className="bot-connection-row__diag">{diagMessage}</em> : null}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {selectedQQ ? (
          <article className="bot-detail-card" aria-labelledby="bot-detail-title">
            <div className="bot-detail-card__head">
              <div className="bot-detail-card__identity">
                <div className="bot-detail-card__title" id="bot-detail-title">
                  QQ Bot
                  <span className="badge badge--neutral">QQ</span>
                  <span className={`badge ${qqOnline ? "badge--project" : qqConfigured ? "badge--feedback" : "badge--feedback"}`}>
                    {qqOnline ? t("settings.botConnectionConnected") : qqConfigured ? t("settings.botConnectionConfigured") : t("settings.botConnectionDisconnected")}
                  </span>
                </div>
                <div className="bot-detail-card__desc">{t("settings.botAutoSaveHint")}</div>
              </div>
              <div className="bot-detail-card__actions">
                <button type="button" className="btn btn--small" onClick={() => setExpandedConnectionId("")}>
                  {t("common.collapse")}
                </button>
              </div>
            </div>

            <section className="bot-detail-section">
              <div className="bot-detail-section__head">{t("settings.botConnectionSummary")}</div>
              <div className="bot-detail-summary">
                <div>
                  <span>{t("settings.botConnectionColumnChannel")}</span>
                  <strong>QQ</strong>
                </div>
                <div>
                  <span>{t("settings.botConnectionColumnRemote")}</span>
                  <code title={draft.qq.appId.trim() || undefined}>{draft.qq.appId.trim() || "—"}</code>
                </div>
                <div>
                  <span>{t("settings.botConnectionColumnScope")}</span>
                  <strong>{t("settings.botScopeGlobal")}</strong>
                </div>
                <div>
                  <span>{t("settings.botConnectionColumnStatus")}</span>
                  <strong>{qqOnline ? t("settings.botConnectionConnected") : qqConfigured ? t("settings.botConnectionConfigured") : t("settings.botConnectionDisconnected")}</strong>
                </div>
              </div>
            </section>

            <section className="bot-detail-section">
              <div className="bot-detail-section__head">{t("settings.botRuntimeSettings")}</div>
              <SettingsField label={t("settings.botEnableBot")} hint={t("settings.botGatewayEnabled")}>
                <ToggleSegment
                  value={draft.qq.enabled}
                  disabled={busy}
                  onChange={(enabled) => {
                    if (enabled && !qqCanEnableAccess) {
                      focusQQAccessSettings();
                      return;
                    }
                    updateQQ({ enabled });
                    void persistQQ({ enabled });
                  }}
                />
              </SettingsField>
              <SettingsField label={t("settings.botSandbox")} hint={t("settings.botInstallQQHint")}>
                <ToggleSegment
                  value={draft.qq.sandbox}
                  disabled={busy}
                  onLabel={t("settings.toggleOn")}
                  offLabel={t("settings.toggleOff")}
                  onChange={(sandbox) => {
                    updateQQ({ sandbox });
                    void persistQQ({ sandbox });
                  }}
                />
              </SettingsField>
            </section>

            <section className="bot-detail-section">
              <div className="bot-detail-section__head">{t("settings.botCredential")}</div>
              <div className="bot-credential-stack">
                <div className="bot-credential-line">
                  <span>{draft.qq.appId.trim() ? t("settings.botCredentialApp", { value: draft.qq.appId.trim() }) : t("settings.botCredentialConfigured")}</span>
                  <strong>{draft.qq.secretSet ? t("settings.botSecretSet") : t("settings.botSecretMissing")}</strong>
                </div>
                <div className="bot-secret-row bot-secret-row--qq">
                  <input
                    className="mem-input"
                    value={draft.qq.appId}
                    disabled={busy}
                    placeholder={t("settings.botAppId")}
                    spellCheck={false}
                    aria-label={t("settings.botAppId")}
                    onChange={(event) => updateQQ({ appId: event.target.value })}
                    onBlur={(event) => void persistQQ({ appId: event.currentTarget.value })}
                  />
                  <input
                    className="mem-input"
                    value={draft.qq.appSecretEnv || DEFAULT_QQ_SECRET_ENV}
                    disabled={busy}
                    placeholder={DEFAULT_QQ_SECRET_ENV}
                    spellCheck={false}
                    aria-label={t("settings.botSecretEnv")}
                    onChange={(event) => updateQQ({ appSecretEnv: event.target.value })}
                    onBlur={(event) => void persistQQ({ appSecretEnv: event.currentTarget.value || DEFAULT_QQ_SECRET_ENV })}
                  />
                  <input
                    className="mem-input"
                    type="password"
                    value={qqSecretValue}
                    disabled={busy}
                    placeholder={draft.qq.secretSet ? t("settings.botSecretReplace") : t("settings.botSecretPaste")}
                    aria-label={t("settings.botSecretValue")}
                    onChange={(event) => setQQSecretValue(event.target.value)}
                  />
                  <button type="button" className="btn btn--secondary btn--small" disabled={busy || !qqCanSaveAndEnable} onClick={() => void saveQQAndEnable()}>
                    {draft.qq.secretSet ? t("settings.saveKey") : t("settings.botSaveAndEnable")}
                  </button>
                  <button type="button" className="btn btn--secondary btn--small" disabled={busy || !draft.qq.secretSet} onClick={() => void clearQQSecret()}>
                    {t("settings.clearKey")}
                  </button>
                </div>
                {!qqCanEnableAccess ? <div className="bot-connect-panel__hint bot-connect-panel__hint--warning">{t("settings.botQQAccessRequired")}</div> : null}
              </div>
            </section>

            <details
              ref={allowlistRef}
              className={`bot-access-panel${allowlistFocused ? " bot-access-panel--focused" : ""}`}
              data-focus-target="bot-allowlist"
              open={allowlistOpen}
              onToggle={(event) => setAllowlistOpen(event.currentTarget.open)}
            >
              <summary className="bot-access-panel__summary">
                <span>
                  <strong>{t("settings.botAccessControl")}</strong>
                  <small>{t("settings.botAllowlistHint")}</small>
                </span>
                <ChevronDown className="bot-access-panel__chevron" size={16} aria-hidden="true" />
              </summary>
              {allowlistOpen ? (
                <div className="bot-access-panel__body">
                  <SettingsField label={t("settings.botAccessMode")} hint={t("settings.botAccessControlHint")}>
                    <ToggleSegment
                      value={!draft.allowlist.allowAll}
                      disabled={busy}
                      onLabel={t("settings.botAccessWhitelist")}
                      offLabel={t("settings.botAccessAll")}
                      onChange={(whitelistOnly) => {
                        const patch = { enabled: whitelistOnly, allowAll: !whitelistOnly };
                        updateAllowlist(patch);
                        void persistAllowlist(patch);
                      }}
                    />
                  </SettingsField>
                  {draft.allowlist.allowAll ? <div className="bot-access-panel__warning">{t("settings.botAllowAllWarn")}</div> : null}
                  <SettingsField label={t("settings.botAllowlistEntries")} hint={t("settings.botListPlaceholder")}>
                    <div className="bot-list-grid bot-list-grid--qq">
                      <label className="bot-list-input">
                        <span>{t("settings.botQQUsers")}</span>
                        <textarea
                          className="mem-input bot-list-input__textarea"
                          value={allowlistText.qqUsers}
                          disabled={busy || draft.allowlist.allowAll}
                          placeholder={t("settings.botListPlaceholder")}
                          spellCheck={false}
                          onChange={(event) => setAllowlistText((prev) => ({ ...prev, qqUsers: event.target.value }))}
                          onBlur={(event) => persistAllowlistText("qqUsers", event.currentTarget.value)}
                        />
                      </label>
                      <label className="bot-list-input">
                        <span>{t("settings.botQQGroups")}</span>
                        <textarea
                          className="mem-input bot-list-input__textarea"
                          value={allowlistText.qqGroups}
                          disabled={busy || draft.allowlist.allowAll}
                          placeholder={t("settings.botListPlaceholder")}
                          spellCheck={false}
                          onChange={(event) => setAllowlistText((prev) => ({ ...prev, qqGroups: event.target.value }))}
                          onBlur={(event) => persistAllowlistText("qqGroups", event.currentTarget.value)}
                        />
                      </label>
                    </div>
                  </SettingsField>
                </div>
              ) : null}
            </details>

            <section className="bot-detail-section bot-detail-section--danger">
              <div>
                <div className="bot-detail-section__head">{t("settings.botDangerZone")}</div>
                <p>{t("settings.deleteBotHint")}</p>
              </div>
              <InlineConfirmButton
                label={t("settings.deleteBot")}
                confirmLabel={t("settings.confirmDeleteBot")}
                cancelLabel={t("common.cancel")}
                disabled={busy}
                danger
                onConfirm={() => void removeQQBot()}
              />
            </section>
          </article>
        ) : null}

        {selectedConnection ? (
          <article className="bot-detail-card" aria-labelledby="bot-detail-title">
            <div className="bot-detail-card__head">
              <div className="bot-detail-card__identity">
                <div className="bot-detail-card__title" id="bot-detail-title">
                  {selectedConnection.label || botConnectionLabel(selectedConnection, t)}
                  <span className="badge badge--neutral">{botConnectionLabel(selectedConnection, t)}</span>
                  <span className={`badge ${selectedConnection.status === "connected" ? "badge--project" : "badge--feedback"}`}>
                    {selectedConnection.status === "connected" ? t("settings.botConnectionConnected") : selectedConnection.status || t("settings.botConnectionDisconnected")}
                  </span>
                </div>
                <div className="bot-detail-card__desc">{t("settings.botAutoSaveHint")}</div>
              </div>
              <div className="bot-detail-card__actions">
                <button type="button" className="btn btn--small" disabled={busy} onClick={() => void diagnoseConnection(selectedConnection.id)}>
                  {t("settings.botDiagnose")}
                </button>
                {(selectedConnection.provider === "feishu" || selectedConnection.provider === "weixin") ? (
                  <button type="button" className="btn btn--small" disabled={busy || !selectedConnectionRemote} onClick={() => void testConnection(selectedConnection)}>
                    {t("settings.botTest")}
                  </button>
                ) : null}
                <button type="button" className="btn btn--small" onClick={() => setExpandedConnectionId("")}>
                  {t("common.collapse")}
                </button>
              </div>
            </div>

              {diagnosticMessage(selectedDiagnostic) ? (
                <div className="bot-detail-notice">
                  <span>{diagnosticMessage(selectedDiagnostic)}</span>
                  {selectedDiagnosticDetail ? (
                    <div className="bot-diagnostic-actions">
                      <button type="button" className="btn btn--secondary btn--small" disabled={busy} onClick={() => void copyConnectionDiagnostic(selectedConnection)}>
                        <Clipboard aria-hidden="true" />
                        {t("settings.botCopyDiagnostic")}
                      </button>
                      <button type="button" className="btn btn--primary btn--small" disabled={busy} onClick={() => void reportConnectionDiagnostic(selectedConnection)}>
                        <Send aria-hidden="true" />
                        {t("settings.botSendDiagnostic")}
                      </button>
                      <small>{t("settings.botDiagnosticPrivacy")}</small>
                    </div>
                  ) : null}
                </div>
              ) : null}

              <section className="bot-detail-section">
                <div className="bot-detail-section__head">{t("settings.botConnectionSummary")}</div>
                <div className="bot-detail-summary">
                  <div>
                    <span>{t("settings.botConnectionColumnChannel")}</span>
                    <strong>{botConnectionLabel(selectedConnection, t)}</strong>
                  </div>
                  <div>
                    <span>{t("settings.botConnectionColumnRemote")}</span>
                    <code title={selectedConnectionRemote || undefined}>{selectedConnectionRemote || "—"}</code>
                  </div>
                  <div>
                    <span>{t("settings.botConnectionColumnScope")}</span>
                    <strong>{botConnectionScopeLabel(selectedConnection, t)}</strong>
                  </div>
                  <div>
                    <span>{t("settings.botConnectionColumnStatus")}</span>
                    <strong>{selectedConnection.status === "connected" ? t("settings.botConnectionConnected") : selectedConnection.status || t("settings.botConnectionDisconnected")}</strong>
                  </div>
                </div>
              </section>

              <details
                ref={allowlistRef}
                className={`bot-access-panel${allowlistFocused ? " bot-access-panel--focused" : ""}`}
                data-focus-target="bot-allowlist"
                open={allowlistOpen}
                onToggle={(event) => setAllowlistOpen(event.currentTarget.open)}
              >
                <summary className="bot-access-panel__summary">
                  <span>
                    <strong>{t("settings.botAccessControl")}</strong>
                    <small>{t("settings.botAllowlistHint")}</small>
                  </span>
                  <ChevronDown className="bot-access-panel__chevron" size={16} aria-hidden="true" />
                </summary>
                {allowlistOpen ? (
                  <div className="bot-access-panel__body">
                    <SettingsField label={t("settings.botAccessMode")} hint={t("settings.botAccessControlHint")}>
                      <ToggleSegment
                        value={!draft.allowlist.allowAll}
                        disabled={busy}
                        onLabel={t("settings.botAccessWhitelist")}
                        offLabel={t("settings.botAccessAll")}
                        onChange={(whitelistOnly) => {
                          const patch = { enabled: whitelistOnly, allowAll: !whitelistOnly };
                          updateAllowlist(patch);
                          void persistAllowlist(patch);
                        }}
                      />
                    </SettingsField>
                    {draft.allowlist.allowAll ? <div className="bot-access-panel__warning">{t("settings.botAllowAllWarn")}</div> : null}
                    <SettingsField label={t("settings.botAllowlistEntries")} hint={t("settings.botListPlaceholder")}>
                      <div className="bot-list-grid">
                        <label className="bot-list-input">
                          <span>{t("settings.botQQUsers")}</span>
                          <textarea
                            className="mem-input bot-list-input__textarea"
                            value={allowlistText.qqUsers}
                            disabled={busy || draft.allowlist.allowAll}
                            placeholder={t("settings.botListPlaceholder")}
                            spellCheck={false}
                            onChange={(event) => setAllowlistText((prev) => ({ ...prev, qqUsers: event.target.value }))}
                            onBlur={(event) => persistAllowlistText("qqUsers", event.currentTarget.value)}
                          />
                        </label>
                        <label className="bot-list-input">
                          <span>{t("settings.botFeishuLarkUsers")}</span>
                          <textarea
                            className="mem-input bot-list-input__textarea"
                            value={allowlistText.feishuUsers}
                            disabled={busy || draft.allowlist.allowAll}
                            placeholder={t("settings.botListPlaceholder")}
                            spellCheck={false}
                            onChange={(event) => setAllowlistText((prev) => ({ ...prev, feishuUsers: event.target.value }))}
                            onBlur={(event) => persistAllowlistText("feishuUsers", event.currentTarget.value)}
                          />
                        </label>
                        <label className="bot-list-input">
                          <span>{t("settings.botWeixinUsers")}</span>
                          <textarea
                            className="mem-input bot-list-input__textarea"
                            value={allowlistText.weixinUsers}
                            disabled={busy || draft.allowlist.allowAll}
                            placeholder={t("settings.botListPlaceholder")}
                            spellCheck={false}
                            onChange={(event) => setAllowlistText((prev) => ({ ...prev, weixinUsers: event.target.value }))}
                            onBlur={(event) => persistAllowlistText("weixinUsers", event.currentTarget.value)}
                          />
                        </label>
                        <label className="bot-list-input">
                          <span>{t("settings.botQQGroups")}</span>
                          <textarea
                            className="mem-input bot-list-input__textarea"
                            value={allowlistText.qqGroups}
                            disabled={busy || draft.allowlist.allowAll}
                            placeholder={t("settings.botListPlaceholder")}
                            spellCheck={false}
                            onChange={(event) => setAllowlistText((prev) => ({ ...prev, qqGroups: event.target.value }))}
                            onBlur={(event) => persistAllowlistText("qqGroups", event.currentTarget.value)}
                          />
                        </label>
                        <label className="bot-list-input">
                          <span>{t("settings.botFeishuLarkGroups")}</span>
                          <textarea
                            className="mem-input bot-list-input__textarea"
                            value={allowlistText.feishuGroups}
                            disabled={busy || draft.allowlist.allowAll}
                            placeholder={t("settings.botListPlaceholder")}
                            spellCheck={false}
                            onChange={(event) => setAllowlistText((prev) => ({ ...prev, feishuGroups: event.target.value }))}
                            onBlur={(event) => persistAllowlistText("feishuGroups", event.currentTarget.value)}
                          />
                        </label>
                        <label className="bot-list-input">
                          <span>{t("settings.botWeixinGroups")}</span>
                          <textarea
                            className="mem-input bot-list-input__textarea"
                            value={allowlistText.weixinGroups}
                            disabled={busy || draft.allowlist.allowAll}
                            placeholder={t("settings.botListPlaceholder")}
                            spellCheck={false}
                            onChange={(event) => setAllowlistText((prev) => ({ ...prev, weixinGroups: event.target.value }))}
                            onBlur={(event) => persistAllowlistText("weixinGroups", event.currentTarget.value)}
                          />
                        </label>
                      </div>
                    </SettingsField>
                  </div>
                ) : null}
              </details>

              <section className="bot-detail-section">
                <div className="bot-detail-section__head">{t("settings.botRuntimeSettings")}</div>
                <SettingsField label={t("settings.botToolApprovalMode")} hint={t("settings.botToolApprovalModeHint")}>
                  <div className="provider-add-segmented" role="group" aria-label={t("settings.botToolApprovalMode")}>
                    {BOT_TOOL_APPROVAL_MODES.map((mode) => (
                      <button
                        key={mode || "inherit"}
                        type="button"
                        className={selectedConnectionToolApprovalMode === mode ? "provider-add-segmented__item provider-add-segmented__item--active" : "provider-add-segmented__item"}
                        disabled={busy}
                        onClick={() => void persistConnection(selectedConnection.id, { toolApprovalMode: mode as BotConnectionToolApprovalMode })}
                      >
                        {t(`settings.botToolApprovalMode.${mode || "inherit"}` as DictKey)}
                      </button>
                    ))}
                  </div>
                </SettingsField>
                <SettingsField label={t("settings.botChannelModel")} hint={t("settings.botChannelModelHint")}>
                  <ModelPicker
                    s={s}
                    refs={refs}
                    value={toRef(selectedConnection.model, s)}
                    disabled={busy}
                    emptyOptionLabel={t("settings.botChannelModelAuto")}
                    emptyOptionHint={settingsModelMeta(s, t)}
                    onPick={(model) => void persistConnection(selectedConnection.id, { model })}
                  />
                </SettingsField>
                <SettingsField label={t("settings.botWorkspaceRoot")} hint={t("settings.botWorkspaceRootHint")}>
                  <input
                    className="mem-input"
                    value={selectedConnection.workspaceRoot}
                    disabled={busy}
                    placeholder={t("settings.botWorkspaceRootPlaceholder")}
                    spellCheck={false}
                    onChange={(event) => updateConnection(selectedConnection.id, { workspaceRoot: event.target.value })}
                    onBlur={(event) => void persistConnection(selectedConnection.id, { workspaceRoot: event.currentTarget.value })}
                  />
                </SettingsField>
              </section>

              <section className="bot-detail-section">
                <div className="bot-detail-section__head">{t("settings.botCredential")}</div>
                <div className="bot-credential-stack">
                  <div className="bot-credential-line">
                    <span>{botConnectionCredentialSummary(selectedConnection, t)}</span>
                    <strong>{selectedConnection.credential.secretSet ? t("settings.botSecretSet") : t("settings.botSecretMissing")}</strong>
                  </div>
                  {botConnectionSecretEnv(selectedConnection) ? (
                    <div className="bot-secret-row">
                      <input
                        className="mem-input"
                        value={botConnectionSecretEnv(selectedConnection)}
                        disabled={busy}
                        spellCheck={false}
                        onChange={(event) => updateConnectionCredential(selectedConnection.id, botConnectionSecretPatch(selectedConnection, event.target.value))}
                        onBlur={(event) => void persistConnectionCredential(selectedConnection.id, botConnectionSecretPatch(selectedConnection, event.currentTarget.value))}
                      />
                      <input
                        className="mem-input"
                        type="password"
                        value={connectionSecrets[selectedConnection.id] ?? ""}
                        disabled={busy}
                        placeholder={selectedConnection.credential.secretSet ? t("settings.botSecretReplace") : t("settings.botSecretPaste")}
                        onChange={(event) => setConnectionSecrets((prev) => ({ ...prev, [selectedConnection.id]: event.target.value }))}
                      />
                      <button type="button" className="btn btn--secondary btn--small" disabled={busy || !(connectionSecrets[selectedConnection.id] ?? "").trim()} onClick={() => void saveConnectionSecret(selectedConnection)}>
                        {t("settings.saveKey")}
                      </button>
                      <button type="button" className="btn btn--secondary btn--small" disabled={busy || !selectedConnection.credential.secretSet} onClick={() => void clearConnectionSecret(selectedConnection)}>
                        {t("settings.clearKey")}
                      </button>
                    </div>
                  ) : null}
                </div>
              </section>

              <section className="bot-detail-section bot-detail-section--danger">
                <div>
                  <div className="bot-detail-section__head">{t("settings.botDangerZone")}</div>
                  <p>{t("settings.deleteBotHint")}</p>
                </div>
                <InlineConfirmButton
                  label={t("settings.deleteBot")}
                  confirmLabel={t("settings.confirmDeleteBot")}
                  cancelLabel={t("common.cancel")}
                  disabled={busy}
                  danger
                  onConfirm={() => removeConnection(selectedConnection)}
                />
              </section>
          </article>
        ) : null}

        <div className="bot-add-panel">
          <div className="bot-phone-connect__top">
            <div className="bot-phone-connect__title">
              <strong>{t("settings.botConnectPhoneTitle")}</strong>
              <span>{t("settings.botConnectPhoneSubtitle")}</span>
            </div>
          </div>

          <div className="bot-phone-targets" role="tablist" aria-label={t("settings.botChannels")}>
            {BOT_INSTALL_TARGETS.map((target) => (
              <button
                key={target}
                type="button"
                role="tab"
                aria-selected={installTarget === target}
                className={`bot-phone-target${installTarget === target ? " bot-phone-target--active" : ""}`}
                disabled={busy || install.status === "starting"}
                onClick={() => setInstallTarget(target)}
              >
                <strong>{botTargetLabel(target, t)}</strong>
                <span>{botTargetHint(target, t)}</span>
              </button>
            ))}
          </div>

          {isQQInstallTarget ? (
            <div className="bot-connect-panel bot-connect-panel--manual bot-connect-panel--qq">
              <div className="bot-connect-panel__body">
                <div className="bot-qq-simple__head">
                  <div>
                    <strong>{selectedInstallLabel}</strong>
                    <p>{t("settings.botInstallManualQQ")}</p>
                  </div>
                  <span className={`bot-qq-simple__status${qqConfigured ? " bot-qq-simple__status--ready" : ""}`}>
                    {qqConfigured ? <CheckCircle2 aria-hidden="true" /> : <KeyRound aria-hidden="true" />}
                    {draft.qq.secretSet ? t("settings.botSecretSet") : t("settings.botSecretMissing")}
                  </span>
                </div>
                <div className="bot-manual-form bot-manual-form--qq">
                  <div className="bot-card-field">
                    <span>{t("settings.botAppId")}</span>
                    <div>
                      <input
                        className="mem-input"
                        aria-label={t("settings.botAppId")}
                        value={draft.qq.appId}
                        disabled={busy}
                        spellCheck={false}
                        onChange={(event) => updateQQ({ appId: event.target.value })}
                        onBlur={(event) => void persistQQ({ appId: event.currentTarget.value })}
                      />
                    </div>
                  </div>
                  <div className="bot-card-field">
                    <span>{t("settings.botAppSecret")}</span>
                    <div>
                      <input
                        className="mem-input"
                        type="password"
                        value={qqSecretValue}
                        disabled={busy}
                        placeholder={draft.qq.secretSet ? t("settings.botSecretSavedOptional") : t("settings.botSecretPaste")}
                        spellCheck={false}
                        aria-label={t("settings.botSecretValue")}
                        onChange={(event) => setQQSecretValue(event.target.value)}
                      />
                    </div>
                  </div>
                  <div className="bot-qq-simple__actions">
                    <button type="button" className="btn btn--primary btn--small" disabled={busy || !qqCanSaveAndEnable} onClick={() => void saveQQAndEnable()}>
                      {t("settings.botSaveAndEnable")}
                    </button>
                  </div>
                  {!qqCanEnableAccess ? <div className="bot-connect-panel__hint bot-connect-panel__hint--warning">{t("settings.botQQAccessRequired")}</div> : null}
                </div>
              </div>
            </div>
          ) : (
            <div className="bot-connect-panel bot-connect-panel--phone">
              <div className="bot-connect-panel__qr">
                {selectedInstallConnection ? (
                  <div className="bot-connect-panel__state bot-connect-panel__state--success">
                    <CheckCircle2 aria-hidden="true" />
                  </div>
                ) : install.status === "showing" && installQrURL ? (
                  installQrIsImage ? (
                    <img src={installQrURL} alt={t("settings.botInstallQrAlt")} />
                  ) : (
                    <Suspense fallback={<div className="bot-connect-panel__state"><QrCode aria-hidden="true" /></div>}>
                      <QRCodeSVG className="bot-connect-panel__qr-code" value={installQrURL} size={196} marginSize={1} />
                    </Suspense>
                  )
                ) : install.status === "starting" ? (
                  <div className="bot-connect-panel__state">
                    <Loader2 className="bot-spin" aria-hidden="true" />
                    <span>{t("settings.botInstallStarting")}</span>
                  </div>
                ) : install.status === "error" ? (
                  <div className="bot-connect-panel__state bot-connect-panel__state--error">
                    <RefreshCw aria-hidden="true" />
                  </div>
                ) : (
                  <div className="bot-connect-panel__state">
                    <QrCode aria-hidden="true" />
                  </div>
                )}
              </div>
              <div className="bot-connect-panel__body">
                <strong>{selectedInstallLabel}</strong>
                <p>
                  {selectedInstallConnection
                    ? t("settings.botInstallAlreadyConnected", { provider: selectedInstallLabel })
                    : install.message || botTargetHint(installTarget, t)}
                </p>
                {install.status === "showing" && install.timeLeft > 0 ? (
                  <span className="bot-connect-panel__timer">{t("settings.botInstallTimeLeft", { time: formatInstallTimeLeft(install.timeLeft) })}</span>
                ) : null}
                {installUserCode ? <code>{installUserCode}</code> : null}
                <div className="bot-connect-panel__actions">
                  {!selectedInstallConnection && install.status !== "showing" && install.status !== "starting" ? (
                    <button type="button" className="btn btn--primary btn--small" disabled={busy} onClick={() => void startInstall(installTarget)}>
                      {install.status === "error" ? <RefreshCw aria-hidden="true" /> : <QrCode aria-hidden="true" />}
                      {install.status === "error" ? t("settings.botInstallRetry") : t("settings.botInstallGenerate")}
                    </button>
                  ) : null}
                  {install.status === "showing" ? (
                    <button type="button" className="btn btn--secondary btn--small" disabled={busy} onClick={() => void pollInstall()}>
                      {t("settings.botInstallCheck")}
                    </button>
                  ) : null}
                  {selectedInstallConnection ? (
                    <button type="button" className="btn btn--secondary btn--small" disabled={busy} onClick={() => void diagnoseConnection(selectedInstallConnection.id)}>
                      {t("settings.botDiagnose")}
                    </button>
                  ) : null}
                </div>
              </div>
            </div>
          )}
        </div>
    </div>
  );
}

function diagnosticMessage(diag?: BotConnectionDiagnostic | string): string {
  if (typeof diag === "string") return diag;
  return diag?.message || diag?.status || "";
}

function diagnosticReportDetail(diag?: BotConnectionDiagnostic | string): string {
  if (typeof diag === "string") return "";
  return diag?.reportDetail || "";
}

function botTargetLabel(target: BotInstallTarget, t: ReturnType<typeof useT>): string {
  switch (target) {
    case "qq": return "QQ";
    case "lark": return "Lark";
    case "weixin": return t("settings.botWeixin");
    default: return t("settings.botFeishu");
  }
}

function botTargetHint(target: BotInstallTarget, t: ReturnType<typeof useT>): string {
  switch (target) {
    case "qq": return t("settings.botInstallQQHint");
    case "lark": return t("settings.botInstallLarkHint");
    case "weixin": return t("settings.botInstallWeixinHint");
    default: return t("settings.botInstallFeishuHint");
  }
}

function qqBotAdded(qq: BotSettingsView["qq"]): boolean {
  return Boolean(qq.enabled || qq.secretSet || qq.appId.trim());
}

function qqAccessReady(allowlist: BotAllowlistView): boolean {
  if (allowlist.allowAll) return true;
  if (!allowlist.enabled) return false;
  return asArray(allowlist.qqUsers).some((value) => value.trim()) || asArray(allowlist.qqGroups).some((value) => value.trim());
}

function botInstallTargetMatchesConnection(target: BotOfficialInstallTarget, connection: BotConnectionView): boolean {
  if (target === "weixin") return connection.provider === "weixin";
  if (target === "lark") return connection.provider === "feishu" && connection.domain === "lark";
  return connection.provider === "feishu" && connection.domain !== "lark";
}

function formatInstallUserCode(code: string): string {
  const compact = code.replace(/[^a-z0-9]/gi, "").toUpperCase().slice(0, 8);
  if (compact.length <= 4) return compact;
  return `${compact.slice(0, 4)}-${compact.slice(4)}`;
}

function formatInstallTimeLeft(seconds: number): string {
  const value = Math.max(0, Math.floor(seconds));
  const minutes = Math.floor(value / 60);
  const rest = value % 60;
  return `${minutes}:${String(rest).padStart(2, "0")}`;
}

function botConnectionLabel(connection: BotConnectionView, t: ReturnType<typeof useT>): string {
  if (connection.domain === "lark") return "Lark";
  if (connection.provider === "weixin") return t("settings.botWeixin");
  if (connection.provider === "qq") return "QQ";
  return t("settings.botFeishu");
}

function firstConnectionRemote(connection: BotConnectionView): string {
  return connection.sessionMappings.find((mapping) => mapping.remoteId.trim())?.remoteId ?? "";
}

function botConnectionScopeLabel(connection: BotConnectionView, t: ReturnType<typeof useT>): string {
  return connection.workspaceRoot.trim() ? t("settings.botScopeProject") : t("settings.botScopeGlobal");
}

function botConnectionSecretEnv(connection: BotConnectionView): string {
  return connection.provider === "weixin" ? connection.credential.tokenEnv : connection.credential.appSecretEnv;
}

function botConnectionSecretPatch(connection: BotConnectionView, value: string): Partial<BotConnectionView["credential"]> {
  return connection.provider === "weixin" ? { tokenEnv: value } : { appSecretEnv: value };
}

function botConnectionCredentialSummary(connection: BotConnectionView, t: ReturnType<typeof useT>): string {
  if (connection.provider === "weixin") {
    return connection.credential.accountId
      ? t("settings.botCredentialAccount", { value: connection.credential.accountId })
      : t("settings.botCredentialLocalWeixin");
  }
  if (connection.credential.appId) {
    return t("settings.botCredentialApp", { value: connection.credential.appId });
  }
  return t("settings.botCredentialConfigured");
}

function sanitizeBotDraft(draft: BotSettingsView): BotSettingsView {
  const bot = normalizeBotSettings(draft);
  return {
    ...bot,
    model: bot.model.trim(),
    toolApprovalMode: normalizeBotToolApprovalMode(bot.toolApprovalMode),
    maxSteps: Math.max(0, Math.floor(bot.maxSteps || 0)),
    debounceMs: Math.max(0, Math.floor(bot.debounceMs || 0)),
    allowlist: {
      ...bot.allowlist,
      qqUsers: uniqueStrings(bot.allowlist.qqUsers.map((v) => v.trim())),
      feishuUsers: uniqueStrings(bot.allowlist.feishuUsers.map((v) => v.trim())),
      weixinUsers: uniqueStrings(bot.allowlist.weixinUsers.map((v) => v.trim())),
      qqGroups: uniqueStrings(bot.allowlist.qqGroups.map((v) => v.trim())),
      feishuGroups: uniqueStrings(bot.allowlist.feishuGroups.map((v) => v.trim())),
      weixinGroups: uniqueStrings(bot.allowlist.weixinGroups.map((v) => v.trim())),
    },
    qq: {
      ...bot.qq,
      appId: bot.qq.appId.trim(),
      appSecretEnv: bot.qq.appSecretEnv.trim(),
    },
    feishu: {
      ...bot.feishu,
      domain: bot.feishu.domain === "lark" ? "lark" : "feishu",
      appId: bot.feishu.appId.trim(),
      appSecretEnv: bot.feishu.appSecretEnv.trim(),
      verificationToken: bot.feishu.verificationToken.trim(),
      mode: bot.feishu.mode === "websocket" ? "websocket" : "webhook",
      webhookPort: Math.max(0, Math.floor(bot.feishu.webhookPort || 0)),
    },
    weixin: {
      ...bot.weixin,
      accountId: bot.weixin.accountId.trim(),
      tokenEnv: bot.weixin.tokenEnv.trim(),
      apiBase: bot.weixin.apiBase.trim().replace(/\/+$/, ""),
    },
    connections: bot.connections.map(normalizeBotConnection).filter((conn) => conn.id && conn.provider),
  };
}

function botDraftWithDerivedGatewayState(draft: BotSettingsView): BotSettingsView {
  const bot = sanitizeBotDraft(draft);
  return {
    ...bot,
    enabled: bot.qq.enabled || bot.connections.some((connection) => connection.enabled),
  };
}

