import { Target, CheckCircle2, AlertTriangle, XCircle, Loader2 } from 'lucide-react';

interface GoalProgressView {
  active: boolean;
  goal: string;
  status: string;
  turns: number;
  blocks: number;
}

interface GoalProgressWidgetProps {
  goal: GoalProgressView | null;
  compact?: boolean;
}

const STATUS_ICONS: Record<string, React.ReactNode> = {
  running: <Loader2 size={12} style={{ animation: 'spin 1s linear infinite' }} />,
  complete: <CheckCircle2 size={12} color="var(--color-green)" />,
  blocked: <AlertTriangle size={12} color="var(--color-yellow)" />,
  stopped: <XCircle size={12} color="var(--color-text-muted)" />,
};

const STATUS_LABELS: Record<string, string> = {
  running: 'active',
  complete: 'done',
  blocked: 'blocked',
  stopped: 'stopped',
};

export function GoalProgressWidget({ goal, compact }: GoalProgressWidgetProps) {
  if (!goal || !goal.active) return null;

  const icon = STATUS_ICONS[goal.status] || STATUS_ICONS.running;
  const label = STATUS_LABELS[goal.status] || goal.status;

  if (compact) {
    return (
      <div
        title={`Goal: ${goal.goal} · ${goal.turns} turns · ${label}`}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 4,
          fontSize: 11,
          padding: '1px 6px',
          borderRadius: 4,
          background: 'var(--color-accent-subtle)',
          border: '1px solid var(--color-accent)',
          cursor: 'default',
          whiteSpace: 'nowrap',
          maxWidth: 200,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        <Target size={12} style={{ color: 'var(--color-accent)', flexShrink: 0 }} />
        <span
          style={{
            fontWeight: 600,
            color: 'var(--color-accent)',
            overflow: 'hidden',
            textOverflow: 'ellipsis',
          }}
        >
          {goal.goal}
        </span>
        <span style={{ color: 'var(--color-text-muted)', flexShrink: 0 }}>{icon}</span>
        <span style={{ color: 'var(--color-text-muted)', flexShrink: 0 }}>{goal.turns}t</span>
      </div>
    );
  }

  return (
    <div
      style={{
        padding: '10px 14px',
        borderRadius: 8,
        background: 'var(--color-surface-raised)',
        border: '1px solid var(--color-border)',
      }}
    >
      <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
        <Target size={16} style={{ color: 'var(--color-accent)' }} />
        <span style={{ fontWeight: 600, flex: 1 }}>{goal.goal}</span>
        <span
          style={{
            fontSize: 11,
            padding: '1px 6px',
            borderRadius: 4,
            background:
              goal.status === 'running' ? 'var(--color-accent-subtle)' : 'var(--color-surface)',
            border: '1px solid var(--color-border)',
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
          }}
        >
          {icon}
          {label}
        </span>
      </div>
      <div
        style={{
          display: 'flex',
          gap: 16,
          fontSize: 12,
          color: 'var(--color-text-muted)',
          marginTop: 6,
        }}
      >
        <span>{goal.turns} turns</span>
        {goal.blocks > 0 && <span>{goal.blocks} blocks</span>}
      </div>
      {/* Progress bar */}
      <div
        style={{
          marginTop: 6,
          height: 3,
          borderRadius: 2,
          background: 'var(--color-border)',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            height: '100%',
            borderRadius: 2,
            background:
              goal.status === 'complete'
                ? 'var(--color-green)'
                : goal.status === 'blocked'
                  ? 'var(--color-yellow)'
                  : 'var(--color-accent)',
            width: goal.status === 'complete' ? '100%' : `${Math.min(goal.turns * 10, 90)}%`,
            transition: 'width 0.3s ease',
          }}
        />
      </div>
    </div>
  );
}
