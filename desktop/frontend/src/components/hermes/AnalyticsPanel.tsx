import { useState, useEffect } from "react";
import { BarChart3, TrendingUp, Zap } from "lucide-react";
import type { TurnTimelinePoint } from "../../lib/types";
import { app } from "../../lib/bridge";

export function AnalyticsPanel() {
  const [timeline, setTimeline] = useState<TurnTimelinePoint[]>([]);

  useEffect(() => {
    app.TurnTimeline().then(setTimeline).catch(() => setTimeline([]));
    const iv = setInterval(() => {
      app.TurnTimeline().then(setTimeline).catch(() => {});
    }, 5000);
    return () => clearInterval(iv);
  }, []);

  if (timeline.length === 0) {
    return (
      <section className="settings-section">
        <h3 className="settings-section__title">
          <BarChart3 size={16} style={{ marginRight: 6 }} />
          Turn Analytics
        </h3>
        <p className="settings-section__desc">
          Per-turn token breakdown, tool calls, and cache efficiency.
        </p>
        <div style={{
          marginTop: 12, padding: "24px", borderRadius: 8,
          background: "var(--color-surface-raised)", border: "1px dashed var(--color-border)",
          textAlign: "center",
        }}>
          <p style={{ color: "var(--color-text-muted)", fontSize: 13, margin: 0 }}>
            Start a session to see per-turn analytics.
          </p>
        </div>
      </section>
    );
  }

  // Aggregate stats
  const totalPrompt = timeline.reduce((s, p) => s + p.promptTokens, 0);
  const totalCompletion = timeline.reduce((s, p) => s + p.completionTokens, 0);
  const totalCacheHit = timeline.reduce((s, p) => s + p.cacheHitTokens, 0);
  const totalCacheMiss = timeline.reduce((s, p) => s + p.cacheMissTokens, 0);
  const totalTokens = totalPrompt + totalCompletion;
  const cacheHitRate = totalCacheHit + totalCacheMiss > 0
    ? totalCacheHit / (totalCacheHit + totalCacheMiss) : 0;
  const maxTokens = Math.max(...timeline.map((p) => p.totalTokens), 1);

  // Tool usage totals
  const toolTotals = new Map<string, number>();
  for (const p of timeline) {
    for (const t of p.toolCalls) {
      toolTotals.set(t, (toolTotals.get(t) || 0) + 1);
    }
  }
  const topTools = Array.from(toolTotals.entries())
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10);

  return (
    <section className="settings-section">
      <h3 className="settings-section__title">
        <BarChart3 size={16} style={{ marginRight: 6 }} />
        Turn Analytics ({timeline.length} turns)
      </h3>
      <p className="settings-section__desc">
        Per-turn token breakdown, tool calls, and cache efficiency.
      </p>

      {/* Aggregate cards */}
      <div style={{
        display: "grid", gridTemplateColumns: "repeat(auto-fill, minmax(120px, 1fr))",
        gap: 8, marginTop: 12, marginBottom: 16,
      }}>
        <AggCard label="Total Tokens" value={fmtNum(totalTokens)} />
        <AggCard label="Prompt" value={fmtNum(totalPrompt)} />
        <AggCard label="Completion" value={fmtNum(totalCompletion)} />
        <AggCard label="Cache Hit Rate" value={`${(cacheHitRate * 100).toFixed(0)}%`}
          color={cacheHitRate > 0.5 ? "var(--color-green)" : cacheHitRate > 0.2 ? "var(--color-yellow)" : "var(--color-red)"} />
      </div>

      {/* Token per turn bar chart */}
      <div style={{ marginBottom: 16 }}>
        <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)", display: "flex", alignItems: "center", gap: 6 }}>
          <TrendingUp size={14} />
          Tokens per Turn
        </h4>
        <div style={{ display: "flex", alignItems: "flex-end", gap: 3, height: 80, padding: "0 4px" }}>
          {timeline.map((p) => {
            const h = Math.max(2, (p.totalTokens / maxTokens) * 80);
            return (
              <div key={p.turn} style={{
                flex: 1, height: h, borderRadius: "2px 2px 0 0",
                background: `linear-gradient(to top, var(--color-accent), var(--color-accent-subtle))`,
                transition: "height 0.2s ease",
              }} title={`Turn ${p.turn}: ${fmtNum(p.totalTokens)} tokens`} />
            );
          })}
        </div>
        <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10, color: "var(--color-text-muted)", marginTop: 4 }}>
          <span>Turn 1</span>
          <span>Turn {timeline.length}</span>
        </div>
      </div>

      {/* Cache hit/miss per turn stacked bars */}
      <div style={{ marginBottom: 16 }}>
        <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)", display: "flex", alignItems: "center", gap: 6 }}>
          <Zap size={14} />
          Cache Efficiency
          <span style={{ fontSize: 11, marginLeft: 4, color: "var(--color-text-muted)" }}>
            {Math.round(cacheHitRate * 100)}% hit
          </span>
        </h4>
        <div style={{ display: "flex", alignItems: "flex-end", gap: 2, height: 40, padding: "0 4px" }}>
          {timeline.map((p) => {
            const promptTotal = p.cacheHitTokens + p.cacheMissTokens || 1;
            const hitPct = (p.cacheHitTokens / promptTotal) * 100;
            const missPct = 100 - hitPct;
            return (
              <div key={p.turn} style={{ flex: 1, height: 40, display: "flex", flexDirection: "column" }}
                title={`Turn ${p.turn}: ${fmtNum(p.cacheHitTokens)} hit / ${fmtNum(p.cacheMissTokens)} miss`}>
                <div style={{ flex: 1, background: "var(--color-green)", borderRadius: "2px 2px 0 0" }} />
                <div style={{ height: `${missPct}%`, background: "var(--color-red-subtle)", borderRadius: "0 0 2px 2px" }} />
              </div>
            );
          })}
        </div>
        <div style={{ display: "flex", gap: 12, fontSize: 11, color: "var(--color-text-muted)", marginTop: 4 }}>
          <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <span style={{ width: 10, height: 10, borderRadius: 2, background: "var(--color-green)" }} /> Cache hit
          </span>
          <span style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <span style={{ width: 10, height: 10, borderRadius: 2, background: "var(--color-red-subtle)" }} /> Cache miss
          </span>
        </div>
      </div>

      {/* Tool usage breakdown */}
      {topTools.length > 0 && (
        <div style={{ marginBottom: 16 }}>
          <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)" }}>
            Top Tools
          </h4>
          <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
            {topTools.map(([name, count], i) => {
              const pct = (count / Math.max(...topTools.map(([, c]) => c))) * 100;
              return (
                <div key={name} style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 12 }}>
                  <span style={{ minWidth: 80, color: "var(--color-text)" }}>{name}</span>
                  <div style={{ flex: 1, height: 14, borderRadius: 4, background: "var(--color-surface-disabled)", overflow: "hidden" }}>
                    <div style={{
                      width: `${pct}%`, height: "100%", borderRadius: 4,
                      background: i < 3 ? "var(--color-accent)" : "var(--color-accent-subtle)",
                      transition: "width 0.3s ease",
                    }} />
                  </div>
                  <span style={{ minWidth: 32, textAlign: "right", color: "var(--color-text-muted)" }}>{count}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Turn tool grid */}
      <div>
        <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)" }}>
          Tools per Turn
        </h4>
        <div style={{
          display: "grid", gridTemplateColumns: `repeat(${Math.min(timeline.length, 8)}, 1fr)`,
          gap: 4, fontSize: 10, overflowX: "auto",
        }}>
          {timeline.map((p) => (
            <div key={p.turn} style={{ textAlign: "center" }}>
              <div style={{
                fontWeight: 700, marginBottom: 2, color: "var(--color-text-muted)",
                borderBottom: "1px solid var(--color-border)", paddingBottom: 2,
              }}>
                T{p.turn}
              </div>
              {p.toolCalls.length === 0 ? (
                <div style={{ color: "var(--color-text-muted)", opacity: 0.5 }}>—</div>
              ) : (
                p.toolCalls.map((t, i) => (
                  <div key={i} style={{
                    padding: "1px 3px", margin: "1px 0", borderRadius: 3,
                    background: "var(--color-accent-subtle)", color: "var(--color-text)",
                    whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis",
                  }}>
                    {t}
                  </div>
                ))
              )}
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

function AggCard({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <div style={{
      padding: "8px 10px", borderRadius: 6,
      background: "var(--color-surface-raised)", border: "1px solid var(--color-border)",
    }}>
      <div style={{ fontSize: 11, color: "var(--color-text-muted)", marginBottom: 2 }}>{label}</div>
      <div style={{ fontSize: 15, fontWeight: 700, color: color || "var(--color-text)" }}>{value}</div>
    </div>
  );
}

function fmtNum(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return `${n}`;
}
