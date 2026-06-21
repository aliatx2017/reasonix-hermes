import { useState, useEffect } from 'react';
import { GitBranch, Clock } from 'lucide-react';
import { app } from '../../lib/bridge';
import type { SubagentNodeView } from '../../lib/types';

interface HermesDashboardPayload {
  subagents?: SubagentNodeView[];
}

export function SubagentTreePanel() {
  const [nodes, setNodes] = useState<SubagentNodeView[]>([]);

  useEffect(() => {
    // Prefer push events; fall back to polling.
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn('hermes:dashboard', (payload: HermesDashboardPayload) => {
          if (payload?.subagents) setNodes(payload.subagents);
        });
        app
          .SubagentTree()
          .then(setNodes)
          .catch((e) => { console.warn('hermes: subagent tree fetch (push path) failed', e) });
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

    // Polling fallback.
    app
      .SubagentTree()
      .then(setNodes)
      .catch((e) => { console.warn('hermes: subagent tree fetch failed', e) });
    const id = setInterval(
      () =>
        app
          .SubagentTree()
          .then(setNodes)
          .catch((e) => { console.warn('hermes: subagent tree poll failed', e) }),
      10000,
    );
    return () => clearInterval(id);
  }, []);

  if (!nodes || nodes.length === 0) {
    return (
      <div
        style={{
          fontSize: 13,
          color: 'var(--color-text-muted)',
          fontStyle: 'italic',
          padding: '8px 0',
        }}
      >
        No sub-agent tasks in this session. Sub-agents spawn when the model uses the{' '}
        <code>task</code> tool or runs subagent skills.
      </div>
    );
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
      {nodes.map((n) => (
        <div
          key={n.ref}
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
            padding: '6px 10px',
            borderRadius: 6,
            background: 'var(--color-surface-raised)',
            border: '1px solid var(--color-border)',
            fontSize: 12,
          }}
        >
          <GitBranch size={14} style={{ color: 'var(--color-accent)', flexShrink: 0 }} />
          <span style={{ fontWeight: 600 }}>{n.name || n.ref}</span>
          <span
            style={{
              fontSize: 10,
              padding: '1px 5px',
              borderRadius: 3,
              background: n.kind === 'task' ? 'var(--color-surface)' : 'var(--color-accent-subtle)',
              color: 'var(--color-text-muted)',
            }}
          >
            {n.kind}
          </span>
          <span style={{ color: 'var(--color-text-muted)' }}>{n.model}</span>
          <span
            style={{
              fontSize: 10,
              padding: '1px 5px',
              borderRadius: 3,
              background:
                n.status === 'completed'
                  ? 'var(--color-green-subtle)'
                  : n.status === 'running'
                    ? 'var(--color-accent-subtle)'
                    : 'var(--color-surface)',
              color:
                n.status === 'completed'
                  ? 'var(--color-green)'
                  : n.status === 'running'
                    ? 'var(--color-accent)'
                    : 'var(--color-text-muted)',
              marginLeft: 'auto',
            }}
          >
            {n.status}
          </span>
          <span style={{ color: 'var(--color-text-muted)', fontSize: 10 }}>
            <Clock size={10} style={{ verticalAlign: 'middle', marginRight: 2 }} />
            {n.createdAt ? new Date(n.createdAt).toLocaleTimeString() : ''}
          </span>
        </div>
      ))}
    </div>
  );
}
