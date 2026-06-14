import { useState, useEffect } from "react";
import { X } from "lucide-react";

export interface TaskFormData {
  name: string;
  cron: string;
  prompt: string;
  model: string;
  enabled: boolean;
}

interface CreateEditTaskModalProps {
  open: boolean;
  task?: TaskFormData | null; // null = create new, non-null = edit existing
  onClose: () => void;
  onSave: (data: TaskFormData) => void;
  onDelete?: (name: string) => void;
}

const DEFAULT_TASK: TaskFormData = {
  name: "",
  cron: "0 9 * * *",
  prompt: "",
  model: "",
  enabled: true,
};

export function CreateEditTaskModal({ open, task, onClose, onSave, onDelete }: CreateEditTaskModalProps) {
  const [data, setData] = useState<TaskFormData>(DEFAULT_TASK);

  useEffect(() => {
    if (open) {
      setData(task ?? { ...DEFAULT_TASK });
    }
  }, [open, task]);

  if (!open) return null;

  const isEditing = task != null;

  const handleSave = () => {
    if (!data.name.trim() || !data.cron.trim() || !data.prompt.trim()) return;
    onSave(data);
    onClose();
  };

  return (
    <div
      style={{
        position: "fixed", top: 0, left: 0, right: 0, bottom: 0,
        background: "rgba(0,0,0,0.4)", zIndex: 1000,
        display: "flex", alignItems: "center", justifyContent: "center",
      }}
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div
        style={{
          background: "var(--color-surface-raised, #fff)", borderRadius: 12,
          padding: 24, width: 420, maxWidth: "90vw",
          boxShadow: "0 8px 32px rgba(0,0,0,0.2)",
        }}
      >
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
          <h3 style={{ margin: 0, fontSize: 16, fontWeight: 600 }}>
            {isEditing ? "Edit Task" : "New Task"}
          </h3>
          <button
            onClick={onClose}
            style={{ background: "none", border: "none", cursor: "pointer", color: "var(--color-text-muted)" }}
          >
            <X size={18} />
          </button>
        </div>

        {/* Name */}
        <label style={{ display: "block", fontSize: 12, fontWeight: 500, marginBottom: 4, color: "var(--color-text-muted)" }}>
          Name
        </label>
        <input
          type="text"
          value={data.name}
          onChange={(e) => setData({ ...data, name: e.target.value })}
          placeholder="daily-review"
          style={inputStyle}
        />

        {/* Cron */}
        <label style={{ display: "block", fontSize: 12, fontWeight: 500, marginTop: 12, marginBottom: 4, color: "var(--color-text-muted)" }}>
          Cron expression
        </label>
        <input
          type="text"
          value={data.cron}
          onChange={(e) => setData({ ...data, cron: e.target.value })}
          placeholder="0 9 * * *"
          style={inputStyle}
        />

        {/* Prompt */}
        <label style={{ display: "block", fontSize: 12, fontWeight: 500, marginTop: 12, marginBottom: 4, color: "var(--color-text-muted)" }}>
          Prompt
        </label>
        <textarea
          value={data.prompt}
          onChange={(e) => setData({ ...data, prompt: e.target.value })}
          placeholder="What should the agent do?"
          rows={3}
          style={{ ...inputStyle, resize: "vertical" }}
        />

        {/* Model (optional) */}
        <label style={{ display: "block", fontSize: 12, fontWeight: 500, marginTop: 12, marginBottom: 4, color: "var(--color-text-muted)" }}>
          Model (optional)
        </label>
        <input
          type="text"
          value={data.model}
          onChange={(e) => setData({ ...data, model: e.target.value })}
          placeholder="default model"
          style={inputStyle}
        />

        {/* Enabled toggle */}
        <label style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 12, fontSize: 13, cursor: "pointer" }}>
          <input
            type="checkbox"
            checked={data.enabled}
            onChange={(e) => setData({ ...data, enabled: e.target.checked })}
          />
          Enabled
        </label>

        {/* Buttons */}
        <div style={{ display: "flex", justifyContent: "space-between", marginTop: 20 }}>
          <div>
            {isEditing && onDelete && (
              <button
                onClick={() => { onDelete(data.name); onClose(); }}
                style={{
                  padding: "6px 14px", borderRadius: 6, border: "1px solid var(--color-error, #e53e3e)",
                  background: "transparent", color: "var(--color-error, #e53e3e)", cursor: "pointer", fontSize: 13,
                }}
              >
                Delete
              </button>
            )}
          </div>
          <div style={{ display: "flex", gap: 8 }}>
            <button
              onClick={onClose}
              style={{
                padding: "6px 14px", borderRadius: 6, border: "1px solid var(--color-border)",
                background: "transparent", color: "var(--color-text)", cursor: "pointer", fontSize: 13,
              }}
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={!data.name.trim() || !data.cron.trim() || !data.prompt.trim()}
              style={{
                padding: "6px 14px", borderRadius: 6, border: "none",
                background: "var(--color-accent)", color: "#fff", cursor: "pointer", fontSize: 13,
                opacity: (!data.name.trim() || !data.cron.trim() || !data.prompt.trim()) ? 0.5 : 1,
              }}
            >
              {isEditing ? "Save" : "Create"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  width: "100%", padding: "6px 10px", fontSize: 13, borderRadius: 6,
  border: "1px solid var(--color-border)", background: "var(--color-surface)",
  color: "var(--color-text)", boxSizing: "border-box",
};
