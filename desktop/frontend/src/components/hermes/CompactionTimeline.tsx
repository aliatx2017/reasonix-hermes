import { useState, useEffect } from 'react';
import { History, Zap, FolderOpen } from 'lucide-react';
import { app } from '../../lib/bridge';

interface CompactionEvent {
  trigger: string;
  messages: number;
  summary: string;
}

interface HermesDashboardPayload {
  compactions?: CompactionEvent[];
}

export function CompactionTimeline() {
  const [events, setEvents] = useState<CompactionEvent[]>([]);

  useEffect(() => {
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn('hermes:dashboard', (payload: HermesDashboardPayload) => {
          if (payload?.compactions) setEvents(payload.compactions);
        });
        app
          .CompactionHistory()
          .then(setEvents)
          .catch((e) => { console.warn('hermes: compaction history fetch (push path) failed', e) });
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
      .CompactionHistory()
      .then(setEvents)
      .catch((e) => { console.warn('hermes: compaction history fetch failed', e) });
    const id = setInterval(
      () =>
        app
          .CompactionHistory()
          .then(setEvents)
          .catch((e) => { console.warn('hermes: compaction history poll failed', e) }),
      5000,
    );
    return () => clearInterval(id);
  }, []);

  if (!events || events.length === 0) {
    return (
      <div
        style={{
          fontSize: 13,
          color: 'var(--color-text-muted)',
          fontStyle: 'italic',
          padding: '8px 0',
        }}
      >
        No compactions yet. Compaction runs automatically when the prompt nears the context window,
        or manually via <code>/compact</code>.
      </div>
    );
  }

  const totalFolded = events.reduce((s, e) => s + e.messages, 0);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div
        style={{
          display: 'flex',
          gap: 12,
          fontSize: 11,
          color: 'var(--color-text-muted)',
          marginBottom: 4,
        }}
      >
        <span>
          <History size={11} style={{ marginRight: 3, verticalAlign: -1 }} />
          {events.length} passes
        </span>
        <span>
          <FolderOpen size={11} style={{ marginRight: 3, verticalAlign: -1 }} />
          {totalFolded} msgs folded
        </span>
      </div>

      <div style={{ position: 'relative', paddingLeft: 16 }}>
        {/* Timeline line */}
        <div
          style={{
            position: 'absolute',
            left: 3,
            top: 4,
            bottom: 4,
            width: 2,
            background: 'var(--color-border)',
            borderRadius: 1,
          }}
        />

        {events
          .slice()
          .reverse()
          .map((e, i) => (
            <div
              key={i}
              style={{
                position: 'relative',
                padding: '6px 0 6px 16px',
                fontSize: 12,
                borderBottom: i < events.length - 1 ? '1px solid var(--color-border-soft)' : 'none',
              }}
            >
              {/* Dot */}
              <div
                style={{
                  position: 'absolute',
                  left: -16,
                  top: 10,
                  width: 8,
                  height: 8,
                  borderRadius: '50%',
                  background: e.trigger === 'manual' ? 'var(--accent)' : 'var(--color-text-muted)',
                  border: '2px solid var(--bg)',
                }}
              />
              <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                <Zap
                  size={10}
                  style={{
                    color: e.trigger === 'manual' ? 'var(--accent)' : 'var(--color-text-muted)',
                  }}
                />
                <strong>{e.trigger === 'manual' ? 'Manual' : 'Auto'} compaction</strong>
                <span style={{ color: 'var(--color-text-muted)' }}>· {e.messages} messages</span>
              </div>
              {e.summary && (
                <div
                  style={{
                    marginTop: 4,
                    fontSize: 11,
                    color: 'var(--color-text-2)',
                    lineHeight: 1.4,
                    maxHeight: 40,
                    overflow: 'hidden',
                  }}
                >
                  {e.summary.length > 120 ? e.summary.slice(0, 120) + '…' : e.summary}
                </div>
              )}
            </div>
          ))}
      </div>
    </div>
  );
}
