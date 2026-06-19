import { useState } from 'react';
import { Clock, CheckCircle2, XCircle, Play, Plus, Edit2, Trash2 } from 'lucide-react';
import type { ScheduleDashboardView, ScheduleTaskView } from '../../lib/types';
import { CreateEditTaskModal, type TaskFormData } from './CreateEditTaskModal';
import { app } from '../../lib/bridge';

interface ScheduleWidgetProps {
  data: ScheduleDashboardView | null;
  onRefresh?: () => void;
}

export function ScheduleWidget({ data, onRefresh }: ScheduleWidgetProps) {
  const [modalOpen, setModalOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<ScheduleTaskView | null>(null);

  const handleCreate = () => {
    setEditingTask(null);
    setModalOpen(true);
  };

  const handleEdit = (task: ScheduleTaskView) => {
    setEditingTask(task);
    setModalOpen(true);
  };

  const handleSave = async (form: TaskFormData) => {
    await app.AddScheduledTask(form.name, form.cron, form.prompt, form.model, form.enabled);
    onRefresh?.();
  };

  const handleDelete = async (name: string) => {
    await app.RemoveScheduledTask(name);
    onRefresh?.();
  };

  const tasks = data?.tasks ?? [];
  const recentRuns = data?.recentRuns ?? [];

  return (
    <div className="hermes-widget" style={{ padding: '8px 0' }}>
      {/* Header */}
      <div
        style={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          marginBottom: 8,
        }}
      >
        <h4 style={{ fontSize: 13, fontWeight: 600, margin: 0, color: 'var(--color-text-muted)' }}>
          <Clock size={13} style={{ marginRight: 4, verticalAlign: 'middle' }} />
          Scheduled Tasks{tasks.length > 0 ? ` (${tasks.length})` : ''}
        </h4>
        <button
          onClick={handleCreate}
          title="Add task"
          style={{
            background: 'var(--color-accent)',
            border: 'none',
            borderRadius: 6,
            color: '#fff',
            cursor: 'pointer',
            padding: '2px 8px',
            fontSize: 11,
            display: 'flex',
            alignItems: 'center',
            gap: 2,
          }}
        >
          <Plus size={13} />
          Task
        </button>
      </div>

      {/* Empty state */}
      {tasks.length === 0 && (
        <div style={{ padding: 12, opacity: 0.7, fontSize: 13 }}>
          No tasks configured. Click <strong>+ Task</strong> to schedule automated agent runs.
        </div>
      )}

      {/* Task list */}
      {tasks.length > 0 && (
        <div style={{ marginBottom: 12 }}>
          {tasks.map((t) => (
            <div
              key={t.name}
              style={{
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                padding: '6px 10px',
                marginBottom: 4,
                borderRadius: 6,
                background: 'var(--color-bg-secondary)',
                fontSize: 12,
                gap: 8,
              }}
            >
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontWeight: 600 }}>{t.name}</div>
                <div style={{ color: 'var(--color-text-muted)', fontSize: 11 }}>
                  {t.cron} · {t.prompt}
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6, flexShrink: 0 }}>
                {t.enabled ? (
                  <span style={{ color: 'var(--color-success, #22c55e)', fontSize: 11 }}>
                    <Play size={10} style={{ marginRight: 2 }} />
                    {t.nextRun ? formatNextRun(t.nextRun) : 'pending'}
                  </span>
                ) : (
                  <span style={{ color: 'var(--color-text-disabled)', fontSize: 11 }}>
                    disabled
                  </span>
                )}
                <button onClick={() => handleEdit(t)} title="Edit" style={iconBtnStyle}>
                  <Edit2 size={12} />
                </button>
                <button
                  onClick={() => handleDelete(t.name)}
                  title="Delete"
                  style={{ ...iconBtnStyle, color: 'var(--color-error, #e53e3e)' }}
                >
                  <Trash2 size={12} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Recent runs */}
      {recentRuns.length > 0 && (
        <div>
          <h4
            style={{
              fontSize: 13,
              fontWeight: 600,
              margin: '0 0 8px',
              color: 'var(--color-text-muted)',
            }}
          >
            Recent Runs
          </h4>
          {recentRuns.map((r, i) => (
            <div
              key={`${r.taskName}-${r.runAt}-${i}`}
              style={{
                display: 'flex',
                gap: 8,
                alignItems: 'flex-start',
                padding: '4px 10px',
                marginBottom: 2,
                borderRadius: 4,
                fontSize: 12,
              }}
            >
              {r.success ? (
                <CheckCircle2
                  size={14}
                  style={{ color: 'var(--color-success, #22c55e)', marginTop: 1, flexShrink: 0 }}
                />
              ) : (
                <XCircle
                  size={14}
                  style={{ color: 'var(--color-error, #ef4444)', marginTop: 1, flexShrink: 0 }}
                />
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <span style={{ fontWeight: 600 }}>{r.taskName}</span>
                <span style={{ color: 'var(--color-text-muted)', marginLeft: 6 }}>
                  {r.duration}
                </span>
                {r.summary && (
                  <div style={{ color: 'var(--color-text-muted)', fontSize: 11, marginTop: 1 }}>
                    {r.summary}
                  </div>
                )}
                {r.error && (
                  <div style={{ color: 'var(--color-error, #ef4444)', fontSize: 11, marginTop: 1 }}>
                    {r.error}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Modal */}
      <CreateEditTaskModal
        open={modalOpen}
        task={
          editingTask
            ? {
                name: editingTask.name,
                cron: editingTask.cron,
                prompt: editingTask.prompt,
                model: editingTask.model ?? '',
                enabled: editingTask.enabled,
              }
            : null
        }
        onClose={() => setModalOpen(false)}
        onSave={handleSave}
        onDelete={handleDelete}
      />
    </div>
  );
}

const iconBtnStyle: React.CSSProperties = {
  background: 'transparent',
  border: 'none',
  cursor: 'pointer',
  padding: 2,
  color: 'var(--color-text-muted)',
};

function formatNextRun(iso: string): string {
  const d = new Date(iso);
  const now = Date.now();
  const diff = d.getTime() - now;
  if (diff < 0) return 'now';
  const mins = Math.floor(diff / 60000);
  if (mins < 60) return `in ${mins}m`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `in ${hrs}h`;
  return d.toLocaleDateString();
}
