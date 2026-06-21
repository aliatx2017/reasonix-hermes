import React, { useState, useCallback, useEffect } from 'react';
import { Activity, GitCompare, FileSearch } from 'lucide-react';
import type { SessionMeta, SessionComparisonView } from '../../lib/types';
import { app } from '../../lib/bridge';

export function EvalPanel() {
  const [sessions, setSessions] = useState<SessionMeta[]>([]);
  const [pathA, setPathA] = useState('');
  const [pathB, setPathB] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<SessionComparisonView | null>(null);

  // Load available sessions on mount.
  useEffect(() => {
    app
      .ListSessions()
      .then(setSessions)
      .catch((e) => { console.warn('EvalPanel: ListSessions fetch failed', e); setSessions([]); });
    // Refresh every 15s in case new sessions are created.
    const iv = setInterval(() => {
      app
        .ListSessions()
        .then(setSessions)
        .catch((e) => { console.warn('hermes: session list poll failed', e) });
    }, 15000);
    return () => clearInterval(iv);
  }, []);

  // Build a deduplicated list of session file paths from SessionMeta.
  const paths = React.useMemo(() => {
    const p = new Set<string>();
    for (const s of sessions) {
      if (s.path) p.add(s.path);
    }
    return Array.from(p).sort();
  }, [sessions]);

  const compare = useCallback(async () => {
    if (!pathA || !pathB) return;
    setLoading(true);
    setError(null);
    setResult(null);
    try {
      const r = await app.CompareSessions(pathA, pathB);
      setResult(r);
    } catch (e) {
      setError(String((e as Error)?.message ?? e));
    } finally {
      setLoading(false);
    }
  }, [pathA, pathB]);

  // Jaccard bar color
  const barColor = (sim: number) => {
    if (sim >= 0.8) return 'var(--color-green)';
    if (sim >= 0.5) return 'var(--color-yellow)';
    return 'var(--color-red)';
  };

  return (
    <section className="settings-section">
      <h3 className="settings-section__title">
        <Activity size={16} style={{ marginRight: 6 }} />
        Session Evaluation
      </h3>
      <p className="settings-section__desc">
        Compare two saved agent sessions — tool usage, token efficiency, turn similarity. Use{' '}
        <code>reasonix eval compare &lt;a&gt; &lt;b&gt;</code> from the CLI.
      </p>

      {/* Session pickers */}
      <div style={{ display: 'flex', gap: 12, marginTop: 12, flexWrap: 'wrap' }}>
        <div style={{ flex: 1, minWidth: 200 }}>
          <label
            style={{
              fontSize: 12,
              color: 'var(--color-text-muted)',
              display: 'block',
              marginBottom: 4,
            }}
          >
            Session A
          </label>
          <select
            value={pathA}
            onChange={(e) => setPathA(e.target.value)}
            style={{
              width: '100%',
              padding: '6px 8px',
              borderRadius: 6,
              border: '1px solid var(--color-border)',
              background: 'var(--color-surface)',
              color: 'var(--color-text)',
              fontSize: 12,
            }}
          >
            <option value="">— select session —</option>
            {paths.map((p) => (
              <option key={p} value={p}>
                {shortenPath(p)}
              </option>
            ))}
          </select>
        </div>
        <div style={{ flex: 1, minWidth: 200 }}>
          <label
            style={{
              fontSize: 12,
              color: 'var(--color-text-muted)',
              display: 'block',
              marginBottom: 4,
            }}
          >
            Session B
          </label>
          <select
            value={pathB}
            onChange={(e) => setPathB(e.target.value)}
            style={{
              width: '100%',
              padding: '6px 8px',
              borderRadius: 6,
              border: '1px solid var(--color-border)',
              background: 'var(--color-surface)',
              color: 'var(--color-text)',
              fontSize: 12,
            }}
          >
            <option value="">— select session —</option>
            {paths.map((p) => (
              <option key={p} value={p}>
                {shortenPath(p)}
              </option>
            ))}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'flex-end' }}>
          <button
            onClick={compare}
            disabled={!pathA || !pathB || loading}
            style={{
              padding: '6px 14px',
              borderRadius: 6,
              border: 'none',
              background: loading ? 'var(--color-surface-disabled)' : 'var(--color-accent)',
              color: '#fff',
              fontWeight: 600,
              fontSize: 13,
              cursor: loading ? 'default' : 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: 6,
            }}
          >
            <GitCompare size={14} />
            {loading ? 'Comparing…' : 'Compare'}
          </button>
        </div>
      </div>

      {error && (
        <div
          style={{
            marginTop: 8,
            padding: '8px 12px',
            borderRadius: 6,
            background: 'var(--color-error-subtle)',
            color: 'var(--color-error)',
            fontSize: 13,
          }}
        >
          {error}
        </div>
      )}

      {/* Results */}
      {result && (
        <div style={{ marginTop: 16 }}>
          {/* Stats cards */}
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(auto-fill, minmax(130px, 1fr))',
              gap: 8,
              marginBottom: 12,
            }}
          >
            <StatCard
              label="Turns"
              a={`${result.turnsA}`}
              b={`${result.turnsB}`}
              delta={result.turnsB - result.turnsA}
            />
            <StatCard
              label="Tokens In"
              a={fmtNum(result.tokensInA)}
              b={fmtNum(result.tokensInB)}
              delta={result.tokensInB - result.tokensInA}
            />
            <StatCard
              label="Tokens Out"
              a={fmtNum(result.tokensOutA)}
              b={fmtNum(result.tokensOutB)}
              delta={result.tokensOutB - result.tokensOutA}
            />
            <StatCard
              label="Cost"
              a={`${result.currency}${result.costA.toFixed(4)}`}
              b={`${result.currency}${result.costB.toFixed(4)}`}
              delta={result.costB - result.costA}
            />
          </div>

          {/* Similarity bar */}
          <div style={{ marginBottom: 12 }}>
            <div
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                fontSize: 12,
                marginBottom: 4,
              }}
            >
              <span style={{ color: 'var(--color-text-muted)' }}>
                Tool sequence similarity (Jaccard)
              </span>
              <span style={{ fontWeight: 600, color: barColor(result.similarity) }}>
                {(result.similarity * 100).toFixed(1)}%
              </span>
            </div>
            <div
              style={{
                width: '100%',
                height: 8,
                borderRadius: 4,
                background: 'var(--color-surface-raised)',
                overflow: 'hidden',
              }}
            >
              <div
                style={{
                  width: `${Math.round(result.similarity * 100)}%`,
                  height: '100%',
                  background: barColor(result.similarity),
                  borderRadius: 4,
                  transition: 'width 0.3s ease',
                }}
              />
            </div>
          </div>

          {/* Tool usage table */}
          {result.toolDiffs.length > 0 && (
            <div style={{ marginBottom: 12 }}>
              <h4
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  margin: '0 0 6px',
                  color: 'var(--color-text-muted)',
                }}
              >
                Tool Usage
              </h4>
              <table
                style={{
                  width: '100%',
                  borderCollapse: 'collapse',
                  fontSize: 12,
                  borderRadius: 6,
                  overflow: 'hidden',
                }}
              >
                <thead>
                  <tr style={{ background: 'var(--color-surface-raised)' }}>
                    <th style={th}>Tool</th>
                    <th style={{ ...th, textAlign: 'right' }}>A</th>
                    <th style={{ ...th, textAlign: 'right' }}>B</th>
                    <th style={{ ...th, textAlign: 'right' }}>Δ</th>
                  </tr>
                </thead>
                <tbody>
                  {result.toolDiffs
                    .filter((d) => d.countA > 0 || d.countB > 0)
                    .map((d) => (
                      <tr key={d.name} style={{ borderBottom: '1px solid var(--color-border)' }}>
                        <td style={td}>{d.name}</td>
                        <td style={{ ...td, textAlign: 'right' }}>{d.countA}</td>
                        <td style={{ ...td, textAlign: 'right' }}>{d.countB}</td>
                        <td
                          style={{
                            ...td,
                            textAlign: 'right',
                            color:
                              d.delta > 0
                                ? 'var(--color-green)'
                                : d.delta < 0
                                  ? 'var(--color-red)'
                                  : 'var(--color-text-muted)',
                          }}
                        >
                          {d.delta > 0 ? `+${d.delta}` : `${d.delta}`}
                        </td>
                      </tr>
                    ))}
                </tbody>
              </table>
            </div>
          )}

          {/* Turn diff summary */}
          {result.turnDiffs.length > 0 && (
            <div>
              <h4
                style={{
                  fontSize: 13,
                  fontWeight: 600,
                  margin: '0 0 6px',
                  color: 'var(--color-text-muted)',
                }}
              >
                Turn Comparison ({result.turnDiffs.filter((t) => t.match).length}/
                {result.turnDiffs.length} matched)
              </h4>
              <div
                style={{
                  display: 'flex',
                  flexWrap: 'wrap',
                  gap: 4,
                  maxHeight: 160,
                  overflowY: 'auto',
                  padding: 4,
                }}
              >
                {result.turnDiffs.map((t) => (
                  <div
                    key={t.index}
                    style={{
                      width: 28,
                      height: 28,
                      borderRadius: 4,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: 11,
                      fontWeight: 700,
                      background: t.match ? 'var(--color-green-subtle)' : 'var(--color-red-subtle)',
                      color: t.match ? 'var(--color-green)' : 'var(--color-red)',
                      border: `1px solid ${t.match ? 'var(--color-green)' : 'var(--color-red)'}`,
                    }}
                    title={`Turn ${t.index}: ${t.match ? 'matched' : 'diverged'}`}
                  >
                    {t.index}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Empty state */}
      {!result && !loading && !error && (
        <div
          style={{
            marginTop: 16,
            padding: '24px',
            borderRadius: 8,
            background: 'var(--color-surface-raised)',
            border: '1px dashed var(--color-border)',
            textAlign: 'center',
          }}
        >
          <FileSearch size={32} style={{ color: 'var(--color-text-muted)', marginBottom: 8 }} />
          <p style={{ color: 'var(--color-text-muted)', fontSize: 13, margin: 0 }}>
            Select two session files and click Compare to analyze the differences.
          </p>
        </div>
      )}
    </section>
  );
}

// Mini components & helpers

function StatCard({ label, a, b, delta }: { label: string; a: string; b: string; delta: number }) {
  const deltaColor =
    delta > 0 ? 'var(--color-green)' : delta < 0 ? 'var(--color-red)' : 'var(--color-text-muted)';
  const sign = delta > 0 ? '+' : '';
  return (
    <div
      style={{
        padding: '10px 12px',
        borderRadius: 6,
        background: 'var(--color-surface-raised)',
        border: '1px solid var(--color-border)',
      }}
    >
      <div style={{ fontSize: 11, color: 'var(--color-text-muted)', marginBottom: 2 }}>{label}</div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'baseline' }}>
        <div>
          <span style={{ fontSize: 13, fontWeight: 600 }}>{a}</span>
          <span style={{ fontSize: 11, color: 'var(--color-text-muted)', margin: '0 4px' }}>
            vs
          </span>
          <span style={{ fontSize: 13, fontWeight: 600 }}>{b}</span>
        </div>
        {delta !== 0 && (
          <span style={{ fontSize: 11, color: deltaColor, fontWeight: 600 }}>
            {sign}
            {typeof delta === 'number' && !label.startsWith('Cost') ? delta : delta.toFixed(4)}
          </span>
        )}
      </div>
    </div>
  );
}

function shortenPath(p: string): string {
  // Strip $HOME prefix
  const home = '$HOME';
  if (p.startsWith(home)) return `~${p.slice(home.length)}`;
  // Take last 50 chars
  if (p.length > 55) return `…${p.slice(-50)}`;
  return p;
}

function fmtNum(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  return `${n}`;
}

const th: React.CSSProperties = {
  padding: '6px 10px',
  fontWeight: 600,
  color: 'var(--color-text-muted)',
  fontSize: 11,
  textAlign: 'left',
  borderBottom: '1px solid var(--color-border)',
};
const td: React.CSSProperties = {
  padding: '5px 10px',
  fontSize: 12,
  color: 'var(--color-text)',
};
