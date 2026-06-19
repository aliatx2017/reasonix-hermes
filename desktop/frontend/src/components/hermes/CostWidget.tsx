import { Coins } from 'lucide-react';
import type { CostSummaryView } from '../../lib/types';

interface CostWidgetProps {
  data: CostSummaryView | null;
}

export function CostWidget({ data }: CostWidgetProps) {
  if (!data) {
    return (
      <div className="hermes-widget" style={{ padding: 12, opacity: 0.7, fontSize: 13 }}>
        No cost data available yet. Start a session to track spending.
      </div>
    );
  }

  const cost = data.sessionCost;
  const displayCost =
    cost < 0.01 ? `${(cost * 100).toFixed(2)}¢` : `${data.currency}${cost.toFixed(4)}`;

  return (
    <div
      className="hermes-widget"
      style={{
        padding: 10,
        borderRadius: 8,
        background: 'var(--color-bg-secondary)',
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        fontSize: 13,
      }}
    >
      <Coins size={16} style={{ color: 'var(--color-accent, #f59e0b)', flexShrink: 0 }} />
      <div style={{ flex: 1 }}>
        <div style={{ fontWeight: 600 }}>{displayCost}</div>
        <div style={{ fontSize: 11, color: 'var(--color-text-muted)' }}>session cost</div>
      </div>
    </div>
  );
}
