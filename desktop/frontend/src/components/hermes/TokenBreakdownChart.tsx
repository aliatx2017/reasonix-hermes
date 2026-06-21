import { useState, useEffect } from 'react';
import { BarChart3 } from 'lucide-react';
import { app } from '../../lib/bridge';

interface TurnUsagePoint {
  turn: number;
  promptTokens: number;
  completionTokens: number;
  cacheHitTokens: number;
  cacheMissTokens: number;
}

interface HermesDashboardPayload {
  turnUsage?: TurnUsagePoint[];
}

const MAX_BARS = 32;
const BAR_WIDTH = 4;
const BAR_GAP = 1;
const CHART_HEIGHT = 48;

export function TokenBreakdownChart() {
  const [points, setPoints] = useState<TurnUsagePoint[]>([]);

  useEffect(() => {
    // Prefer push events; fall back to polling.
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn('hermes:dashboard', (payload: HermesDashboardPayload) => {
          if (payload?.turnUsage) setPoints(payload.turnUsage);
        });
        app
          .TurnUsageHistory()
          .then(setPoints)
          .catch((e) => { console.warn('hermes: turn usage history fetch (push path) failed', e) });
        return () => {
          try {
            unsub();
          } catch {
            /* ignore */
          }
        };
      }
    } catch {
      /* fall through */
    }

    app
      .TurnUsageHistory()
      .then(setPoints)
      .catch((e) => { console.warn('hermes: turn usage history fetch failed', e) });
    const id = setInterval(
      () =>
        app
          .TurnUsageHistory()
          .then(setPoints)
          .catch((e) => { console.warn('hermes: turn usage history poll failed', e) }),
      5000,
    );
    return () => clearInterval(id);
  }, []);

  if (!points || points.length === 0) {
    return (
      <div
        style={{
          fontSize: 13,
          color: 'var(--color-text-muted)',
          fontStyle: 'italic',
          padding: '8px 0',
        }}
      >
        No token data yet. Run a few turns to see the breakdown.
      </div>
    );
  }

  // Show the last MAX_BARS entries.
  const visible = points.length > MAX_BARS ? points.slice(points.length - MAX_BARS) : points;
  const maxTokens = Math.max(...visible.map((p) => p.promptTokens + p.completionTokens), 1);

  const cacheHitRate =
    points.reduce((s, p) => s + p.cacheHitTokens, 0) /
    Math.max(
      points.reduce((s, p) => s + p.cacheHitTokens + p.cacheMissTokens, 0),
      1,
    );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      {/* Summary row */}
      <div style={{ display: 'flex', gap: 16, fontSize: 11, color: 'var(--color-text-muted)' }}>
        <span>
          <BarChart3 size={11} style={{ marginRight: 4, verticalAlign: -1 }} />
          {points.length} turns
        </span>
        <span>Cache: {(cacheHitRate * 100).toFixed(1)}%</span>
        <span>Peak: {maxTokens.toLocaleString()} tok</span>
      </div>

      {/* Sparkline bars */}
      <div
        style={{
          display: 'flex',
          alignItems: 'flex-end',
          gap: BAR_GAP,
          height: CHART_HEIGHT,
          padding: '2px 0',
        }}
      >
        {visible.map((p, i) => {
          const promptH = (p.promptTokens / maxTokens) * CHART_HEIGHT;
          const completionH = (p.completionTokens / maxTokens) * CHART_HEIGHT;
          const isLast = i === visible.length - 1;
          return (
            <div
              key={i}
              title={`Turn ${p.turn}: ${p.promptTokens.toLocaleString()} prompt + ${p.completionTokens.toLocaleString()} completion (${p.cacheHitTokens.toLocaleString()} cached)`}
              style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'flex-end',
                width: BAR_WIDTH,
                height: CHART_HEIGHT,
                opacity: isLast ? 1 : 0.65,
              }}
            >
              <div
                style={{
                  height: `${Math.max(completionH, 0.5)}px`,
                  background: 'var(--accent)',
                  borderRadius: '1px 1px 0 0',
                  transition: 'height 0.2s',
                }}
              />
              <div
                style={{
                  height: `${Math.max(promptH, 0.5)}px`,
                  background: 'var(--accent-soft)',
                  borderRadius: '1px 1px 0 0',
                  transition: 'height 0.2s',
                }}
              />
            </div>
          );
        })}
      </div>

      {/* Legend */}
      <div style={{ display: 'flex', gap: 10, fontSize: 10, color: 'var(--color-text-muted)' }}>
        <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
          <span style={{ width: 8, height: 8, background: 'var(--accent)', borderRadius: 2 }} />
          Completion
        </span>
        <span style={{ display: 'flex', alignItems: 'center', gap: 3 }}>
          <span
            style={{ width: 8, height: 8, background: 'var(--accent-soft)', borderRadius: 2 }}
          />
          Prompt
        </span>
      </div>
    </div>
  );
}
