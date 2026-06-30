// ContextPanel shows the active tab's context gauge and token usage.
// All visible text is routed through the i18n dictionary.
import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import { asArray } from "../lib/array";
import { app } from "../lib/bridge";
import { useI18n, type Locale, type Translator } from "../lib/i18n";
import { formatMoneyLocalized } from "../lib/money";
import type { DictKey } from "../locales/en";
import type { BalanceInfo, ContextInfo, ContextPanelInfo, UsageSourceStats, WireUsage } from "../lib/types";

interface ContextPanelProps {
  tabId?: string;
  context?: ContextInfo;
  usage?: WireUsage;
  sessionTokens?: number;
  sessionCost?: number;
  sessionCurrency?: string;
  sessionTurns?: number;
  turnTokens?: number;
  turnCost?: number;
  balance?: BalanceInfo;
  sessionGen?: number;
  refreshKey?: number;
}

function fmtTokens(n: number): string {
  if (n >= 1000) return `${Math.round(n / 1000)}k`;
  return String(n);
}

function fmtDuration(ms: number, t: Translator): string {
  if (ms <= 0) return "-";
  const totalSeconds = Math.max(1, Math.round(ms / 1000));
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes <= 0) return t("context.durationSeconds", { seconds });
  return t("context.durationMinutesSeconds", { minutes, seconds });
}

function fmtOptionalTokens(tokens?: number): string {
  if (typeof tokens !== "number" || tokens <= 0) return "-";
  return tokens.toLocaleString();
}

interface MetricTokenDisplay {
  display: string;
  exact: string;
}

function numberLocale(locale: Locale | string): string {
  if (locale === "zh") return "zh-CN";
  if (locale === "zh-TW") return "zh-TW";
  return "en";
}

export function formatMetricTokens(tokens: number | undefined, locale: Locale | string): MetricTokenDisplay {
  if (typeof tokens !== "number" || tokens <= 0) {
    return { display: "-", exact: "-" };
  }
  const tag = numberLocale(locale);
  const exact = tokens.toLocaleString(tag);
  return { display: exact, exact };
}

function fmtTurns(turns: number | undefined, t: Translator): string {
  if (typeof turns !== "number" || turns < 0) return "-";
  return t(turns === 1 ? "history.turnOne" : "history.turnOther", { n: turns });
}

function fmtUsageCacheRate(usage?: WireUsage): string {
  if (!usage) return "-";
  const denom = usage.cacheHitTokens + usage.cacheMissTokens;
  if (denom <= 0) return "-";
  return `${((usage.cacheHitTokens / denom) * 100).toFixed(2)}%`;
}

export function formatCacheHitRate(hitTokens: number, missTokens: number): string {
  const denom = hitTokens + missTokens;
  if (denom <= 0) return "-";
  return `${((hitTokens / denom) * 100).toFixed(2)}%`;
}

interface HealthResult {
  tone: "good" | "notice" | "warn";
  shortKey: DictKey;
  vars: Record<string, string | number>;
}

export function contextCostDisplay({
  info,
  sessionCost,
  sessionCurrency,
  usage,
}: {
  info?: Pick<ContextPanelInfo, "sessionCost" | "sessionCurrency" | "sessionCostUsd"> | null;
  sessionCost?: number;
  sessionCurrency?: string;
  usage?: Pick<WireUsage, "cost" | "costUsd" | "currency">;
}): { amount: number; currency?: string } {
  if (info?.sessionCost && info.sessionCost > 0) {
    return { amount: info.sessionCost, currency: info.sessionCurrency || sessionCurrency || usage?.currency };
  }
  if (sessionCost && sessionCost > 0) {
    return { amount: sessionCost, currency: sessionCurrency || info?.sessionCurrency || usage?.currency };
  }
  if (usage?.cost && usage.cost > 0) {
    return { amount: usage.cost, currency: usage.currency || sessionCurrency || info?.sessionCurrency };
  }
  if (info?.sessionCostUsd && info.sessionCostUsd > 0) {
    return { amount: info.sessionCostUsd, currency: info.sessionCurrency || sessionCurrency || usage?.currency };
  }
  if (usage?.costUsd && usage.costUsd > 0) {
    return { amount: usage.costUsd, currency: usage.currency || sessionCurrency || info?.sessionCurrency };
  }
  return { amount: 0, currency: info?.sessionCurrency || sessionCurrency || usage?.currency };
}

interface ContextBreakdown {
  promptTokens: number;
  completionTokens: number;
  reasoningTokens: number;
  otherTokens: number;
  promptPct: number;
  completionPct: number;
  reasoningPct: number;
  otherPct: number;
}

function nonNegativeTokenCount(value: number): number {
  return Number.isFinite(value) ? Math.max(0, value) : 0;
}

export function contextBreakdown(
  usedTokens: number,
  windowTokens: number,
  promptTokens: number,
  completionTokens: number,
  reasoningTokens: number,
): ContextBreakdown {
  const used = nonNegativeTokenCount(usedTokens);
  const window = nonNegativeTokenCount(windowTokens);
  let prompt = nonNegativeTokenCount(promptTokens);
  let reasoning = Math.min(nonNegativeTokenCount(reasoningTokens), nonNegativeTokenCount(completionTokens));
  let completion = Math.max(0, nonNegativeTokenCount(completionTokens) - reasoning);
  const known = prompt + completion + reasoning;

  if (known > used && known > 0) {
    const scale = used / known;
    prompt *= scale;
    completion *= scale;
    reasoning *= scale;
  }

  const normalizedKnown = Math.min(used, prompt + completion + reasoning);
  const other = Math.max(0, used - normalizedKnown);
  const hasWindow = window > 0;
  const promptPct = hasWindow ? Math.min(100, (prompt / window) * 100) : 0;
  const completionPct = hasWindow ? Math.min(100, ((prompt + completion) / window) * 100) : 0;
  const reasoningPct = hasWindow ? Math.min(100, ((prompt + completion + reasoning) / window) * 100) : 0;
  const otherPct = hasWindow ? Math.min(100, (used / window) * 100) : 0;

  return {
    promptTokens: Math.round(prompt),
    completionTokens: Math.round(completion),
    reasoningTokens: Math.round(reasoning),
    otherTokens: Math.round(other),
    promptPct,
    completionPct,
    reasoningPct,
    otherPct,
  };
}

function contextHealth(usagePct: number, cachePct: number, readCount: number): HealthResult {
  if (usagePct >= 85) {
    return {
      tone: "warn",
      shortKey: "context.healthNearLimitShort",
      vars: { pct: usagePct },
    };
  }
  if (readCount >= 8) {
    return {
      tone: "notice",
      shortKey: "context.healthManyFilesShort",
      vars: { count: readCount },
    };
  }
  if (cachePct > 0 && cachePct < 50) {
    return {
      tone: "notice",
      shortKey: "context.healthLowCacheShort",
      vars: { pct: cachePct },
    };
  }
  return {
    tone: "good",
    shortKey: "context.healthGoodShort",
    vars: {},
  };
}

const SOURCE_ORDER = ["executor", "planner", "subagent", "compaction", "classifier", "title"];

function sourceTone(source: string): string {
  switch (source) {
    case "executor": return "teal";
    case "planner": return "blue";
    case "subagent": return "amber";
    case "compaction": return "slate";
    case "classifier": return "violet";
    case "title": return "rose";
    default: return "default";
  }
}

function sourceLabel(source: string, t: Translator): string {
  switch (source) {
    case "executor": return t("context.sourceExecutor");
    case "planner": return t("context.sourcePlanner");
    case "subagent": return t("context.sourceSubagent");
    case "compaction": return t("context.sourceCompaction");
    case "classifier": return t("context.sourceClassifier");
    case "title": return t("context.sourceTitle");
    default: return source;
  }
}

function sourceCost(stats: UsageSourceStats): number {
  return stats.sessionCost && stats.sessionCost > 0 ? stats.sessionCost : stats.sessionCostUsd ?? 0;
}

function sourceTokenTotal(row: Pick<ContextSourceRow, "promptTokens" | "completionTokens" | "totalTokens">): number {
  return row.totalTokens > 0 ? row.totalTokens : row.promptTokens + row.completionTokens;
}

export interface ContextSourceRow {
  source: string;
  label: string;
  promptTokens: number;
  completionTokens: number;
  cacheHitTokens: number;
  cacheMissTokens: number;
  totalTokens: number;
  cost: number;
  currency?: string;
  requests: number;
}

export function contextSourceRows(info: ContextPanelInfo | null, sessionCurrency?: string): ContextSourceRow[] {
  const entries = Object.entries(info?.sources ?? {});
  if (entries.length === 0) return [];
  return entries
    .filter(([, stats]) =>
      (stats.requestCount ?? 0) > 0 ||
      (stats.promptTokens ?? 0) > 0 ||
      (stats.completionTokens ?? 0) > 0 ||
      (stats.cacheHitTokens ?? 0) > 0 ||
      (stats.cacheMissTokens ?? 0) > 0 ||
      sourceCost(stats) > 0
    )
    .sort(([a], [b]) => {
      const ia = SOURCE_ORDER.indexOf(a);
      const ib = SOURCE_ORDER.indexOf(b);
      if (ia >= 0 || ib >= 0) return (ia >= 0 ? ia : SOURCE_ORDER.length) - (ib >= 0 ? ib : SOURCE_ORDER.length);
      return a.localeCompare(b);
    })
    .map(([source, stats]) => ({
      source,
      label: source,
      promptTokens: stats.promptTokens ?? 0,
      completionTokens: stats.completionTokens ?? 0,
      cacheHitTokens: stats.cacheHitTokens ?? 0,
      cacheMissTokens: stats.cacheMissTokens ?? 0,
      totalTokens: stats.totalTokens ?? 0,
      cost: sourceCost(stats),
      currency: stats.sessionCurrency || sessionCurrency || info?.sessionCurrency,
      requests: stats.requestCount ?? 0,
    }));
}

export function ContextPanel({
  tabId,
  context,
  usage,
  sessionTokens,
  sessionCost,
  sessionCurrency,
  sessionTurns,
  turnTokens,
  turnCost,
  balance,
  sessionGen,
  refreshKey,
}: ContextPanelProps) {
  const { locale, t } = useI18n();
  const [info, setInfo] = useState<ContextPanelInfo | null>(null);
  const refreshSeq = useRef(0);

  const refresh = useCallback(async () => {
    if (!tabId) return;
    const seq = ++refreshSeq.current;
    try {
      const next = await app.ContextPanel(tabId);
      if (refreshSeq.current === seq) {
        setInfo(next);
      }
    } catch {
      /* bridge unavailable */
    }
  }, [tabId]);

  useEffect(() => {
    refreshSeq.current += 1;
    setInfo(null);
    void refresh();
  }, [refresh, sessionGen]);

  useEffect(() => {
    void refresh();
  }, [refresh, refreshKey]);

  const hasPanelUsage = Boolean(
    (info?.requestCount ?? 0) > 0 ||
    (info?.promptTokens ?? 0) > 0 ||
    (info?.completionTokens ?? 0) > 0 ||
    (info?.totalTokens ?? 0) > 0 ||
    (info?.reasoningTokens ?? 0) > 0 ||
    (info?.cacheHitTokens ?? 0) > 0 ||
    (info?.cacheMissTokens ?? 0) > 0
  );
  const usedTokens = context?.used && context.used > 0 ? context.used : info?.usedTokens ?? 0;
  const windowTokens = context?.window && context.window > 0 ? context.window : info?.windowTokens ?? 0;
  const promptTokens = hasPanelUsage ? info?.promptTokens ?? 0 : usage?.promptTokens ?? 0;
  const completionTokens = hasPanelUsage ? info?.completionTokens ?? 0 : usage?.completionTokens ?? 0;
  const totalTokens = info?.totalTokens && info.totalTokens > 0
    ? info.totalTokens
    : sessionTokens && sessionTokens > 0
      ? sessionTokens
      : usage?.totalTokens && usage.totalTokens > 0
        ? usage.totalTokens
        : promptTokens + completionTokens;
  const reasoningTokens = hasPanelUsage ? info?.reasoningTokens ?? 0 : usage?.reasoningTokens ?? 0;
  const cacheHitTokens = hasPanelUsage ? info?.cacheHitTokens ?? 0 : usage?.cacheHitTokens ?? 0;
  const cacheMissTokens = hasPanelUsage ? info?.cacheMissTokens ?? 0 : usage?.cacheMissTokens ?? 0;
  // Session-cumulative values for the metrics cards (方案A: 纯前端改数据源)
  const sessionCacheHit = info?.sessionCacheHitTokens ?? usage?.sessionCacheHitTokens ?? context?.cacheHitTokens ?? 0;
  const sessionCacheMiss = info?.sessionCacheMissTokens ?? usage?.sessionCacheMissTokens ?? context?.cacheMissTokens ?? 0;
  const sessionCompletion = info?.sessionCompletionTokens ?? 0;
  const sessionCacheHitMetric = formatMetricTokens(sessionCacheHit, locale);
  const sessionCacheMissMetric = formatMetricTokens(sessionCacheMiss, locale);
  const sessionCompletionMetric = formatMetricTokens(sessionCompletion, locale);
  const totalTokensMetric = formatMetricTokens(totalTokens, locale);
  const cost = contextCostDisplay({ info, sessionCost, sessionCurrency, usage });
  const sourceUsageRows = contextSourceRows(info, sessionCurrency);
  const showSourceUsageRows = sourceUsageRows.length > 0;
  const sourceTotalTokens = sourceUsageRows.reduce((sum, row) => sum + sourceTokenTotal(row), 0);
  const readFiles = asArray(info?.readFiles);
  const changedFiles = asArray(info?.changedFiles);

  const usagePct = windowTokens > 0 ? Math.min(100, Math.round((usedTokens / windowTokens) * 100)) : 0;
  const compactPct = context?.compactRatio ? Math.round(context.compactRatio * 100) : 0;
  const cacheDenom = cacheHitTokens + cacheMissTokens;
  const cachePct = cacheDenom > 0 ? (cacheHitTokens / cacheDenom) * 100 : 0;
  const cachePctDisplay = formatCacheHitRate(cacheHitTokens, cacheMissTokens);
  const breakdown = contextBreakdown(usedTokens, windowTokens, promptTokens, completionTokens, reasoningTokens);
  const donutStyle = {
    background: `conic-gradient(#13a7a5 0 ${breakdown.promptPct}%, #2f6df6 ${breakdown.promptPct}% ${breakdown.completionPct}%, #f97316 ${breakdown.completionPct}% ${breakdown.reasoningPct}%, var(--border) ${breakdown.reasoningPct}% ${breakdown.otherPct}%, var(--border-soft) ${breakdown.otherPct}% 100%)`,
  };
  const eventTimes = [
    ...readFiles.map((file) => file.time),
    ...changedFiles.map((file) => file.latestTime ?? 0),
  ].filter((time) => time > 0);
  const derivedElapsed = eventTimes.length > 1 ? Math.max(...eventTimes) - Math.min(...eventTimes) : 0;
  const elapsed = info?.elapsedMs && info.elapsedMs > 0 ? info.elapsedMs : derivedElapsed;
  const derivedRequestCount = Math.max(readFiles.length + changedFiles.length, 0);
  const requestCount = info?.requestCount && info.requestCount > 0 ? info.requestCount : derivedRequestCount;
  const health = contextHealth(usagePct, Math.round(cachePct), readFiles.length);
  const balanceLabel = balance?.available && balance.display ? balance.display : "-";
  const turnCostLabel = formatMoneyLocalized(turnCost, sessionCurrency, { locale, empty: "dash" });
  const sessionCostLabel = formatMoneyLocalized(sessionCost, sessionCurrency, { locale, empty: "dash" });

  return (
    <div className="context-panel">
      <div className="context-panel__body">
        <section className="context-panel__overview">
          <section className="context-panel__usage">
            <SectionHeading title={t("context.windowTitle")} meta={t("context.windowSubtitle")} />
            <div className="context-panel__usage-visual">
              <div className="context-panel__donut" style={donutStyle}>
                <div className="context-panel__donut-core">
                  <strong>{fmtTokens(usedTokens)}</strong>
                  <span>/ {fmtTokens(windowTokens)} tokens</span>
                </div>
              </div>
              <div className="context-panel__percent">{usagePct}%</div>
            </div>
            <div className="context-panel__usage-progress" aria-label={t("context.windowSubtitle")}>
              <div className="context-panel__progress-head">
                <strong>{fmtTokens(usedTokens)} / {fmtTokens(windowTokens)}</strong>
                <span>{usagePct}%</span>
              </div>
              <div className="context-panel__progress-track" aria-hidden="true">
                <span className="context-panel__progress-fill" style={{ width: `${usagePct}%` }} />
              </div>
            </div>
            <div className="context-panel__summary-rows">
              <MiniStat label={t("status.compactLabel")} value={compactPct > 0 ? `${compactPct}%` : "-"} />
              <MiniStat label={t("status.cacheAvgLabel")} value={formatCacheHitRate(sessionCacheHit, sessionCacheMiss)} />
              <MiniStat label={t("context.sessionCost")} value={sessionCostLabel} />
              <MiniStat label={t("status.sessionTurnsLabel")} value={fmtTurns(sessionTurns, t)} />
            </div>
            <div className="context-panel__breakdown">
              <TokenLegend label={t("context.prompt")} value={breakdown.promptTokens} color="prompt" />
              <TokenLegend label={t("context.completion")} value={breakdown.completionTokens} color="completion" />
              <TokenLegend label={t("context.reasoning")} value={breakdown.reasoningTokens} color="reasoning" />
              <TokenLegend label={t("context.other")} value={breakdown.otherTokens} color="other" />
              <div className="context-panel__total">
                <span>{t("context.total")}</span>
                <strong>{usedTokens.toLocaleString()} / {windowTokens.toLocaleString()}</strong>
              </div>
            </div>
          </section>
          <section className="context-panel__creation-grid" aria-label={t("context.overview")}>
            <MetricCard label={t("context.time")} value={fmtDuration(elapsed, t)} />
            <MetricCard label={t("context.requests")} value={requestCount > 0 ? String(requestCount) : "-"} />
            <MetricCard label={t("status.cacheLabel")} value={fmtUsageCacheRate(usage)} tone="accent" />
            <MetricCard label={t("status.turnTokensLabel")} value={fmtOptionalTokens(turnTokens)} />
            <MetricCard label={t("status.turnCostLabel")} value={turnCostLabel} />
            <MetricCard label={t("status.balanceLabel")} value={balanceLabel} tone="accent" />
          </section>
          <section className="context-panel__section">
            <SectionHeading title={t("context.runtimeMetrics")} />
            <div className="context-panel__runtime-card">
              <div className="context-panel__runtime-summary">
                <RuntimeMetric label={t("context.time")} value={fmtDuration(elapsed, t)} />
                <RuntimeMetric label={t("context.requests")} value={requestCount > 0 ? String(requestCount) : "-"} />
                <RuntimeMetric label={t("context.outputTokens")} value={sessionCompletionMetric.display} title={sessionCompletionMetric.exact} />
                <RuntimeMetric label={t("context.sessionTokens")} value={totalTokensMetric.display} title={totalTokensMetric.exact} />
              </div>
              <div className="context-panel__runtime-cache">
                <div className="context-panel__runtime-cache-head">
                  <span>{t("context.inputCache")}</span>
                  <strong>{formatCacheHitRate(sessionCacheHit, sessionCacheMiss)}</strong>
                </div>
                <SourceSplitBar
                  label={`${t("context.sourceCacheHit")}/${t("context.sourceCacheMiss")}`}
                  segments={[
                    { label: t("context.sourceCacheHit"), value: sessionCacheHit, tone: "hit" },
                    { label: t("context.sourceCacheMiss"), value: sessionCacheMiss, tone: "miss" },
                  ]}
                />
                <div className="context-panel__runtime-cache-values">
                  <SourceMetric label={t("context.sourceCacheHit")} value={sessionCacheHitMetric.display} title={sessionCacheHitMetric.exact} />
                  <SourceMetric label={t("context.sourceCacheMiss")} value={sessionCacheMissMetric.display} title={sessionCacheMissMetric.exact} />
                </div>
              </div>
            </div>
          </section>
          <section className="context-panel__section">
            <SectionHeading title={t("context.costMetrics")} />
            <div className="context-panel__stats">
              <MetricCard label={t("context.cacheHit")} value={cachePctDisplay} tone="accent" />
              <MetricCard label={t("context.sessionCost")} value={formatMoneyLocalized(cost.amount, cost.currency, { locale, empty: "dash" })} />
            </div>
            {showSourceUsageRows && (
              <div className="context-panel__source-list" aria-label={t("context.sourceBreakdown")}>
                <div className="context-panel__source-overview">
                  <div className="context-panel__source-overview-head">
                    <strong>{t("context.sourceBreakdown")}</strong>
                    <span>{t("context.sourceShareTokens")}</span>
                  </div>
                  <div className="context-panel__source-sharebar" aria-hidden="true">
                    {sourceUsageRows.map((row) => {
                      const sharePct = sourceTotalTokens > 0 ? (sourceTokenTotal(row) / sourceTotalTokens) * 100 : 0;
                      if (sharePct <= 0) return null;
                      return (
                        <span
                          className={`context-panel__source-share context-panel__source-tone--${sourceTone(row.source)}`}
                          key={row.source}
                          style={{ width: `${sharePct}%` }}
                        />
                      );
                    })}
                  </div>
                  <div className="context-panel__source-legend">
                    {sourceUsageRows.map((row) => {
                      const sharePct = sourceTotalTokens > 0 ? (sourceTokenTotal(row) / sourceTotalTokens) * 100 : 0;
                      return (
                        <span key={row.source}>
                          <i className={`context-panel__source-dot context-panel__source-tone--${sourceTone(row.source)}`} aria-hidden="true" />
                          {sourceLabel(row.label, t)} {sharePct > 0 ? `${sharePct.toFixed(0)}%` : "-"}
                        </span>
                      );
                    })}
                  </div>
                </div>
                {sourceUsageRows.map((row) => {
                  const inputMetric = formatMetricTokens(row.promptTokens, locale);
                  const outputMetric = formatMetricTokens(row.completionTokens, locale);
                  const hitMetric = formatMetricTokens(row.cacheHitTokens, locale);
                  const missMetric = formatMetricTokens(row.cacheMissTokens, locale);
                  const totalMetric = formatMetricTokens(sourceTokenTotal(row), locale);
                  const cacheReported = row.cacheHitTokens + row.cacheMissTokens > 0;
                  const cacheRate = cacheReported ? formatCacheHitRate(row.cacheHitTokens, row.cacheMissTokens) : t("context.cacheNotReported");
                  const costLabel = formatMoneyLocalized(row.cost, row.currency, { locale, empty: "dash" });
                  return (
                    <div className="context-panel__source-row" key={row.source}>
                      <div className="context-panel__source-head">
                        <span>
                          <i className={`context-panel__source-dot context-panel__source-tone--${sourceTone(row.source)}`} aria-hidden="true" />
                          {sourceLabel(row.label, t)}
                        </span>
                        <em>{t("context.sourceRequests", { count: row.requests })}</em>
                      </div>
                      <div className="context-panel__source-summary">
                        <SourceMetric label={t("context.total")} value={totalMetric.display} title={totalMetric.exact} />
                        <SourceMetric label={t("context.sourceCacheRate")} value={cacheRate} />
                        <SourceMetric label={t("context.sourceCost")} value={costLabel} />
                      </div>
                      <SourceSplitBar
                        label={`${t("context.sourceInput")}/${t("context.sourceOutput")}`}
                        segments={[
                          { label: t("context.sourceInput"), value: row.promptTokens, tone: "input" },
                          { label: t("context.sourceOutput"), value: row.completionTokens, tone: "output" },
                        ]}
                      />
                      {cacheReported ? (
                        <SourceSplitBar
                          label={`${t("context.sourceCacheHit")}/${t("context.sourceCacheMiss")}`}
                          segments={[
                            { label: t("context.sourceCacheHit"), value: row.cacheHitTokens, tone: "hit" },
                            { label: t("context.sourceCacheMiss"), value: row.cacheMissTokens, tone: "miss" },
                          ]}
                          compact
                        />
                      ) : (
                        <SourceSplitBar label={`${t("context.sourceCacheHit")}/${t("context.sourceCacheMiss")}`} segments={[]} compact />
                      )}
                      <details className="context-panel__source-details">
                        <summary>{t("context.sourceDetails")}</summary>
                        <div className="context-panel__source-metrics">
                          <SourceMetric label={t("context.sourceInput")} value={inputMetric.display} title={inputMetric.exact} />
                          <SourceMetric label={t("context.sourceOutput")} value={outputMetric.display} title={outputMetric.exact} />
                          <SourceMetric label={t("context.sourceCacheHit")} value={hitMetric.display} title={hitMetric.exact} />
                          <SourceMetric label={t("context.sourceCacheMiss")} value={missMetric.display} title={missMetric.exact} />
                        </div>
                      </details>
                    </div>
                  );
                })}
              </div>
            )}
          </section>
          <section className="context-panel__section context-panel__section--status">
            <SectionHeading title={t("context.sessionStatus")} />
            <div className="context-panel__stats">
              <MetricCard label={t("context.health")} value={t(health.shortKey, health.vars)} tone={health.tone} />
              <MetricCard label={t("context.compaction")} value={compactPct > 0 ? `${compactPct}%` : "-"} />
            </div>
          </section>
        </section>
      </div>

    </div>
  );
}

function SectionHeading({ title, meta, children }: { title: string; meta?: string; children?: ReactNode }) {
  return (
    <header className="context-panel__section-head">
      <h3>{title}</h3>
      {meta && <span>{meta}</span>}
      {children}
    </header>
  );
}

function TokenLegend({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="context-panel__legend-row">
      <span className={`context-panel__legend-dot context-panel__legend-dot--${color}`} />
      <span>{label}</span>
      <strong>{value.toLocaleString()}</strong>
    </div>
  );
}

function MiniStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="context-panel__mini-stat">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function MetricCard({ label, value, valueTitle, tone, wide }: { label: string; value: string; valueTitle?: string; tone?: "accent" | "good" | "notice" | "warn"; wide?: boolean }) {
  const toneClass = tone ? ` context-panel__metric--${tone}` : "";
  const wideClass = wide ? " context-panel__metric--wide" : "";
  const exactTitle = valueTitle && valueTitle !== value ? valueTitle : undefined;
  return (
    <div className={`context-panel__metric${toneClass}${wideClass}`} aria-label={exactTitle ? `${label}: ${exactTitle}` : undefined}>
      <span>{label}</span>
      <strong title={exactTitle}>{value}</strong>
    </div>
  );
}

function RuntimeMetric({ label, value, title }: { label: string; value: string; title?: string }) {
  const exactTitle = title && title !== value ? title : undefined;
  return (
    <div className="context-panel__runtime-metric" aria-label={exactTitle ? `${label}: ${exactTitle}` : undefined}>
      <span>{label}</span>
      <strong title={exactTitle}>{value}</strong>
    </div>
  );
}

function SourceMetric({ label, value, title }: { label: string; value: string; title?: string }) {
  const exactTitle = title && title !== value ? title : undefined;
  return (
    <div className="context-panel__source-metric" aria-label={exactTitle ? `${label}: ${exactTitle}` : undefined}>
      <span>{label}</span>
      <strong title={exactTitle}>{value}</strong>
    </div>
  );
}

function SourceSplitBar({ label, segments, compact }: { label: string; segments: Array<{ label: string; value: number; tone: string }>; compact?: boolean }) {
  const total = segments.reduce((sum, segment) => sum + Math.max(0, segment.value), 0);
  const visible = segments.filter((segment) => segment.value > 0);
  const compactClass = compact ? " context-panel__source-bar--compact" : "";
  if (total <= 0 || visible.length === 0) {
    return (
      <div className="context-panel__source-bar-row">
        <span>{label}</span>
        <div className={`context-panel__source-bar context-panel__source-bar--empty${compactClass}`} aria-hidden="true" />
      </div>
    );
  }
  return (
    <div className="context-panel__source-bar-row">
      <span>{label}</span>
      <div className={`context-panel__source-bar${compactClass}`}>
        {visible.map((segment) => {
          const width = (segment.value / total) * 100;
          return (
            <span
              className={`context-panel__source-bar-segment context-panel__source-bar-segment--${segment.tone}`}
              key={segment.tone}
              style={{ width: `${width}%` }}
              title={`${segment.label}: ${segment.value.toLocaleString()}`}
            />
          );
        })}
      </div>
    </div>
  );
}
