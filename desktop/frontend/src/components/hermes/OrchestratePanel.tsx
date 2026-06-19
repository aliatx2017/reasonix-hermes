import { useState } from 'react';
import { GitMerge, GitBranch, Wrench, Copy, Terminal } from 'lucide-react';

export function OrchestratePanel() {
  const [copied, setCopied] = useState<string | null>(null);

  const copyCmd = (cmd: string) => {
    navigator.clipboard.writeText(cmd).then(() => {
      setCopied(cmd);
      setTimeout(() => setCopied(null), 1400);
    });
  };

  const workflows = [
    {
      icon: <GitBranch size={18} />,
      label: 'Chain',
      cmd: '/chain refactor the auth module',
      desc: 'Analyze → Implement. First agent analyzes the task, second implements based on the analysis. One sequential turn.',
    },
    {
      icon: <GitMerge size={18} />,
      label: 'Pair',
      cmd: '/pair review the new API endpoint',
      desc: 'Review + Implement → Merge. Parallel reviewer and implementer, then a third agent merges results.',
    },
    {
      icon: <Wrench size={18} />,
      label: 'CI‑Fix',
      cmd: '/ci-fix go test ./...',
      desc: 'Run CI command, parse failures, spawn one fix agent per failing test. Re-run to verify.',
    },
  ];

  return (
    <section className="settings-section">
      <h3 className="settings-section__title">
        <Terminal size={16} style={{ marginRight: 6 }} />
        Orchestration
      </h3>
      <p className="settings-section__desc">
        Multi-agent workflows available as slash commands in the CLI chat. Type <code>/chain</code>,{' '}
        <code>/pair</code>, or <code>/ci-fix</code> followed by your task.
      </p>

      <div style={{ marginTop: 12, display: 'flex', flexDirection: 'column', gap: 10 }}>
        {workflows.map((w) => (
          <div
            key={w.label}
            style={{
              padding: '12px 14px',
              borderRadius: 8,
              background: 'var(--color-surface-raised)',
              border: '1px solid var(--color-border)',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 6 }}>
              <span style={{ color: 'var(--color-accent)' }}>{w.icon}</span>
              <span style={{ fontWeight: 700, fontSize: 14 }}>{w.label}</span>
            </div>
            <p style={{ fontSize: 12, color: 'var(--color-text-muted)', margin: '0 0 8px' }}>
              {w.desc}
            </p>
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 8,
                padding: '6px 10px',
                borderRadius: 6,
                background: 'var(--color-code-bg)',
                fontFamily: 'var(--mono)',
                fontSize: 12,
                color: 'var(--color-code-fg)',
              }}
            >
              <code style={{ flex: 1 }}>{w.cmd}</code>
              <button
                onClick={() => copyCmd(w.cmd)}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  color: copied === w.cmd ? 'var(--color-green)' : 'var(--color-text-muted)',
                  padding: 2,
                  display: 'flex',
                  alignItems: 'center',
                }}
                title="Copy command"
              >
                {copied === w.cmd ? '✓' : <Copy size={14} />}
              </button>
            </div>
          </div>
        ))}
      </div>

      <div
        style={{
          marginTop: 16,
          padding: '10px 14px',
          borderRadius: 8,
          background: 'var(--color-accent-subtle)',
          border: '1px solid var(--color-accent)',
          fontSize: 12,
          color: 'var(--color-text)',
        }}
      >
        <strong>Tip:</strong> These run in the CLI chat TUI. For desktop orchestration, open a
        session in the terminal and use the slash commands — results appear inline in your chat.
      </div>
    </section>
  );
}
