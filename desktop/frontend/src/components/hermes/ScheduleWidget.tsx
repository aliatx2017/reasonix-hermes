import { Clock, CheckCircle2, XCircle, Play } from "lucide-react";
import type { ScheduleDashboardView } from "../../lib/types";

interface ScheduleWidgetProps {
  data: ScheduleDashboardView | null;
}

export function ScheduleWidget({ data }: ScheduleWidgetProps) {
  if (!data || !data.active || data.tasks.length === 0) {
    return (
      <div className="hermes-widget" style={{ padding: 12, opacity: 0.7, fontSize: 13 }}>
        No scheduled tasks configured. Add tasks in <code>[schedule]</code> to enable automated agent runs.
      </div>
    );
  }

  return (
    <div className="hermes-widget" style={{ padding: "8px 0" }}>
      {/* Task list */}
      <div style={{ marginBottom: 12 }}>
        <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)" }}>
          <Clock size={13} style={{ marginRight: 4, verticalAlign: "middle" }} />
          Scheduled Tasks ({data.tasks.length})
        </h4>
        {data.tasks.map((t) => (
          <div
            key={t.name}
            style={{
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
              padding: "6px 10px",
              marginBottom: 4,
              borderRadius: 6,
              background: "var(--color-bg-secondary)",
              fontSize: 12,
              gap: 8,
            }}
          >
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontWeight: 600 }}>{t.name}</div>
              <div style={{ color: "var(--color-text-muted)", fontSize: 11 }}>
                {t.cron} · {t.prompt}
              </div>
            </div>
            <div style={{ textAlign: "right", flexShrink: 0 }}>
              {t.enabled ? (
                <span style={{ color: "var(--color-success, #22c55e)", fontSize: 11 }}>
                  <Play size={10} style={{ marginRight: 2 }} />
                  {t.nextRun ? formatNextRun(t.nextRun) : "pending"}
                </span>
              ) : (
                <span style={{ color: "var(--color-text-disabled)", fontSize: 11 }}>disabled</span>
              )}
            </div>
          </div>
        ))}
      </div>

      {/* Recent runs */}
      {data.recentRuns.length > 0 && (
        <div>
          <h4 style={{ fontSize: 13, fontWeight: 600, margin: "0 0 8px", color: "var(--color-text-muted)" }}>
            Recent Runs
          </h4>
          {data.recentRuns.map((r, i) => (
            <div
              key={`${r.taskName}-${r.runAt}-${i}`}
              style={{
                display: "flex",
                gap: 8,
                alignItems: "flex-start",
                padding: "4px 10px",
                marginBottom: 2,
                borderRadius: 4,
                fontSize: 12,
              }}
            >
              {r.success ? (
                <CheckCircle2 size={14} style={{ color: "var(--color-success, #22c55e)", marginTop: 1, flexShrink: 0 }} />
              ) : (
                <XCircle size={14} style={{ color: "var(--color-error, #ef4444)", marginTop: 1, flexShrink: 0 }} />
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <span style={{ fontWeight: 600 }}>{r.taskName}</span>
                <span style={{ color: "var(--color-text-muted)", marginLeft: 6 }}>{r.duration}</span>
                {r.summary && (
                  <div style={{ color: "var(--color-text-muted)", fontSize: 11, marginTop: 1 }}>
                    {r.summary}
                  </div>
                )}
                {r.error && (
                  <div style={{ color: "var(--color-error, #ef4444)", fontSize: 11, marginTop: 1 }}>
                    {r.error}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function formatNextRun(iso: string): string {
  const d = new Date(iso);
  const now = Date.now();
  const diff = d.getTime() - now;
  if (diff < 0) return "now";
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `in ${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `in ${hrs}h`;
  return d.toLocaleDateString();
}
