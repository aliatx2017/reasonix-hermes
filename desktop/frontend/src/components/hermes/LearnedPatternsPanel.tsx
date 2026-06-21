import { useState, useEffect } from 'react';
import { Brain, Lightbulb, Route } from 'lucide-react';
import type { LearnedPatternView, LearnedTrajectoryView } from '../../lib/types';
import { app } from '../../lib/bridge';

export function LearnedPatternsPanel() {
  const [patterns, setPatterns] = useState<LearnedPatternView[]>([]);
  const [trajectories, setTrajectories] = useState<LearnedTrajectoryView[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    app
      .LearnedPatterns()
      .then(([p, t]) => {
        setPatterns(p ?? []);
        setTrajectories(t ?? []);
        setLoading(false);
      })
      .catch((e) => { console.warn('LearnedPatternsPanel: patterns fetch failed', e); setLoading(false); });
    const iv = setInterval(() => {
      app
        .LearnedPatterns()
        .then(([p, t]) => {
          setPatterns(p ?? []);
          setTrajectories(t ?? []);
        })
        .catch((e) => { console.warn('hermes: learned patterns poll failed', e) });
    }, 15000);
    return () => clearInterval(iv);
  }, []);

  if (loading) {
    return (
      <section className="settings-section">
        <h3 className="settings-section__title">
          <Brain size={16} style={{ marginRight: 6 }} />
          Learned Patterns
        </h3>
        <p className="settings-section__desc" style={{ color: 'var(--color-text-muted)' }}>
          Loading…
        </p>
      </section>
    );
  }

  return (
    <section className="settings-section">
      <h3 className="settings-section__title">
        <Brain size={16} style={{ marginRight: 6 }} />
        Learned Patterns
      </h3>
      <p className="settings-section__desc">
        Detected agent behaviour patterns and multi-turn trajectories. Enable <code>[learn]</code>{' '}
        in config and use <code>/learn</code> in chat.
      </p>

      {/* Patterns */}
      {patterns.length === 0 ? (
        <EmptyHint
          icon={<Lightbulb size={24} />}
          text="No patterns detected yet. Patterns form after repeated tool sequences (e.g. edit-then-test, write-then-build)."
        />
      ) : (
        <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 8 }}>
          {patterns.map((p) => (
            <div
              key={p.name}
              style={{
                padding: '10px 14px',
                borderRadius: 8,
                background: 'var(--color-surface-raised)',
                border: '1px solid var(--color-border)',
              }}
            >
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: 4,
                }}
              >
                <span style={{ fontWeight: 600, fontSize: 13 }}>{p.name}</span>
                <span
                  style={{
                    fontSize: 11,
                    padding: '1px 8px',
                    borderRadius: 10,
                    background:
                      p.confidence >= 4
                        ? 'var(--color-green-subtle)'
                        : 'var(--color-yellow-subtle)',
                    color: p.confidence >= 4 ? 'var(--color-green)' : 'var(--color-yellow)',
                  }}
                >
                  ×{p.confidence}
                </span>
              </div>
              <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginBottom: 4 }}>
                <strong>Trigger:</strong> {p.trigger}
              </div>
              <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginBottom: 6 }}>
                <strong>Action:</strong> {p.action}
              </div>
              {p.draft && (
                <pre
                  style={{
                    fontSize: 11,
                    margin: 0,
                    padding: '6px 8px',
                    borderRadius: 4,
                    background: 'var(--color-surface)',
                    color: 'var(--color-text-muted)',
                    maxHeight: 80,
                    overflow: 'hidden',
                  }}
                >
                  {p.draft.slice(0, 300)}
                </pre>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Trajectories */}
      {trajectories.length === 0 ? (
        <div style={{ marginTop: 16 }}>
          <EmptyHint
            icon={<Route size={24} />}
            text="No trajectories recorded. Trajectories group turns with similar tool sequences."
          />
        </div>
      ) : (
        <div style={{ marginTop: 16 }}>
          <h4
            style={{
              fontSize: 13,
              fontWeight: 600,
              margin: '0 0 8px',
              color: 'var(--color-text-muted)',
              display: 'flex',
              alignItems: 'center',
              gap: 6,
            }}
          >
            <Route size={14} /> Multi-Turn Trajectories
          </h4>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
            {trajectories.map((t, i) => (
              <div
                key={i}
                style={{
                  padding: '6px 10px',
                  borderRadius: 6,
                  fontSize: 12,
                  background: 'var(--color-surface-raised)',
                  border: '1px solid var(--color-border)',
                  display: 'flex',
                  alignItems: 'center',
                  gap: 8,
                }}
              >
                <span style={{ fontWeight: 600 }}>{t.label}</span>
                <span style={{ color: 'var(--color-text-muted)' }}>turns {t.turns.join(', ')}</span>
                <span
                  style={{
                    fontSize: 11,
                    padding: '1px 6px',
                    borderRadius: 8,
                    background: 'var(--color-accent-subtle)',
                    color: 'var(--color-accent)',
                  }}
                >
                  ×{t.count}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}
    </section>
  );
}

function EmptyHint({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div
      style={{
        marginTop: 12,
        padding: '24px',
        borderRadius: 8,
        background: 'var(--color-surface-raised)',
        border: '1px dashed var(--color-border)',
        textAlign: 'center',
      }}
    >
      <div style={{ color: 'var(--color-text-muted)', marginBottom: 8 }}>{icon}</div>
      <p style={{ color: 'var(--color-text-muted)', fontSize: 13, margin: 0 }}>{text}</p>
    </div>
  );
}
