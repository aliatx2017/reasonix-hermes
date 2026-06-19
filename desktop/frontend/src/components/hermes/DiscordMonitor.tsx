import { MessageCircle, Send, MessagesSquare } from "lucide-react";
import type { BotLiveStatusView, BotPlatformStatus } from "../../lib/types";

interface BotLiveMonitorProps {
  status: BotLiveStatusView | null;
}

const platformIcons: Record<string, React.ComponentType<{ size: number }>> = {
  discord: MessageCircle,
  telegram: Send,
  line: MessagesSquare,
  slack: ({ size }) => (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M14.5 2C13.1 2 12 3.1 12 4.5v5c0 .8.7 1.5 1.5 1.5h5c1.4 0 2.5-1.1 2.5-2.5S19.9 6 18.5 6H17V4.5C17 3.1 15.9 2 14.5 2z"/>
      <path d="M9.5 22c1.4 0 2.5-1.1 2.5-2.5v-5c0-.8-.7-1.5-1.5-1.5h-5C4.1 13 3 14.1 3 15.5S4.1 18 5.5 18H7v1.5C7 20.9 8.1 22 9.5 22z"/>
      <path d="M2 9.5C2 8.1 3.1 7 4.5 7h5c.8 0 1.5.7 1.5 1.5v5C11 14.9 9.9 16 8.5 16h-5C2.1 16 1 14.9 1 13.5v-4z"/>
      <path d="M22 14.5c0 1.4-1.1 2.5-2.5 2.5h-5c-.8 0-1.5-.7-1.5-1.5v-5C13 9.1 14.1 8 15.5 8h5C21.9 8 23 9.1 23 10.5v4z"/>
    </svg>
  ),
};

const platformLabels: Record<string, string> = {
  discord: "Discord",
  telegram: "Telegram",
  line: "LINE",
  slack: "Slack",
};

export function BotLiveMonitor({ status }: BotLiveMonitorProps) {
  if (!status || !status.platforms || status.platforms.length === 0) return null;

  return (
    <div style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
      {status.platforms.map((p: BotPlatformStatus) => {
        const Icon = platformIcons[p.platform] || MessageCircle;
        const color = p.running ? "var(--color-green)" : "var(--color-text-muted)";
        const label = platformLabels[p.platform] || p.platform;
        const title = p.webhookURL
          ? `${label} Bot · ${p.activeSessions} sessions · Webhook: ${p.webhookURL}`
          : `${label} Bot · ${p.activeSessions} sessions`;

        return (
          <div
            key={p.platform}
            className="hermes-bot-monitor-chip"
            title={title}
            style={{
              display: "inline-flex",
              alignItems: "center",
              gap: 3,
              fontSize: 11,
              padding: "1px 6px",
              borderRadius: 4,
              background: "var(--color-surface-raised)",
              border: "1px solid var(--color-border)",
              cursor: "default",
              whiteSpace: "nowrap",
            }}
          >
            <Icon size={12} />
            <svg width={6} height={6} viewBox="0 0 6 6">
              <circle cx={3} cy={3} r={3} fill={p.running ? "var(--color-green)" : "var(--color-text-muted)"} />
            </svg>
            <span style={{ fontWeight: 600, color }}>{label}</span>
            {p.running && p.activeSessions > 0 && (
              <span style={{ color: "var(--color-text-muted)" }}>
                {p.activeSessions}
              </span>
            )}
          </div>
        );
      })}
    </div>
  );
}

// Keep legacy name for backward compatibility — used by StatusBar.
export { BotLiveMonitor as DiscordMonitor };
