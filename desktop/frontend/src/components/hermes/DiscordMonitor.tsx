import { MessageCircle, Circle } from "lucide-react";

interface BotLiveStatusView {
  running: boolean;
  platform: string;
  activeSessions: number;
  status: string;
}

interface DiscordMonitorProps {
  status: BotLiveStatusView | null;
}

export function DiscordMonitor({ status }: DiscordMonitorProps) {
  if (!status) return null;

  const color = status.running ? "var(--color-green)" : "var(--color-text-muted)";

  return (
    <div
      className="hermes-discord-monitor"
      title={`Discord Bot: ${status.status} · ${status.activeSessions} active sessions`}
      style={{
        display: "inline-flex",
        alignItems: "center",
        gap: 4,
        fontSize: 11,
        padding: "1px 6px",
        borderRadius: 4,
        background: "var(--color-surface-raised)",
        border: "1px solid var(--color-border)",
        cursor: "default",
        whiteSpace: "nowrap",
      }}
    >
      <MessageCircle size={12} style={{ color }} />
      <Circle
        size={6}
        fill={status.running ? "var(--color-green)" : "var(--color-text-muted)"}
        color={status.running ? "var(--color-green)" : "var(--color-text-muted)"}
      />
      <span style={{ fontWeight: 600, color }}>Discord</span>
      {status.running && status.activeSessions > 0 && (
        <span style={{ color: "var(--color-text-muted)" }}>
          {status.activeSessions} online
        </span>
      )}
      {!status.running && (
        <span style={{ color: "var(--color-text-muted)" }}>offline</span>
      )}
    </div>
  );
}
