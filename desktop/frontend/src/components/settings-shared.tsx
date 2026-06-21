import { type ReactNode } from "react";
import { useT } from "../lib/i18n";
import type { SettingsView } from "../lib/types";
import { providerIsConfigured } from "../lib/providerModels";
import { Tooltip } from "./Tooltip";

// ── Layout primitives ──────────────────────────────────────────────────────

export function SettingsSection({
  title,
  description,
  actions,
  children,
}: {
  title?: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
  children: ReactNode;
}) {
  const hasHead = Boolean(title || description || actions);
  return (
    <section className="settings-section">
      {hasHead && (
        <div className="settings-section__head">
          <div>
            {title && <div className="settings-section__title">{title}</div>}
            {description && (
              <div className="settings-section__desc">
                <SettingsHint hint={description} />
              </div>
            )}
          </div>
          {actions && <div className="settings-section__actions">{actions}</div>}
        </div>
      )}
      <div className="settings-section__body">{children}</div>
    </section>
  );
}

export function SettingsField({
  label,
  hint,
  children,
  className,
  stacked = false,
}: {
  label: ReactNode;
  hint?: ReactNode;
  children: ReactNode;
  className?: string;
  stacked?: boolean;
}) {
  return (
    <div className={`settings-field${stacked ? " settings-field--stacked" : ""}${className ? ` ${className}` : ""}`}>
      <div className="settings-field__copy">
        <div className="settings-field__label">{label}</div>
        {hint && (
          <div className="settings-field__hint">
            <SettingsHint hint={hint} />
          </div>
        )}
      </div>
      <div className="settings-field__control">{children}</div>
    </div>
  );
}

export function SettingsHint({ hint }: { hint: ReactNode }) {
  if (typeof hint === "string" || typeof hint === "number") {
    const label = String(hint);
    return (
      <Tooltip label={label} fill block className="settings-field__hint-tooltip">
        <span className="settings-field__hint-line">{label}</span>
      </Tooltip>
    );
  }
  return hint;
}

// ── Shared types ────────────────────────────────────────────────────────────

export type SectionProps = {
  s: SettingsView;
  busy: boolean;
  apply: (fn: () => Promise<unknown>) => Promise<void>;
};

export type SettingsInitialFocus = { target: "bot-allowlist"; connectionId?: string };

// ── Model ref helpers ──────────────────────────────────────────────────────

// allRefs returns every "provider/model" ref from configured providers.
export function allRefs(s: SettingsView): string[] {
  const out: string[] = [];
  for (const p of s.providers) {
    if (!p.added || !providerIsConfigured(p)) continue;
    for (const m of p.models) out.push(`${p.name}/${m}`);
  }
  return out;
}

// toRef normalises a stored model id (a provider name, a bare model, or a ref) to
// a "provider/model" ref so a <select> of refs can show it selected.
export function toRef(model: string, s: SettingsView): string {
  if (!model) return "";
  if (model.includes("/")) return model;
  const byName = s.providers.find((p) => p.name === model);
  if (byName) return `${byName.name}/${byName.default || byName.models[0] || ""}`;
  const byModel = s.providers.find((p) => p.models.includes(model));
  if (byModel) return `${byModel.name}/${model}`;
  return model;
}

// ── String utilities ───────────────────────────────────────────────────────

export function uniqueStrings(values: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const value of values) {
    if (value && !seen.has(value)) {
      seen.add(value);
      out.push(value);
    }
  }
  return out;
}

// ── Proxy mode (shared constant + type + label) ─────────────────────────────

export const PROXY_MODES = ["auto", "custom", "off"] as const;
export type ProxyMode = (typeof PROXY_MODES)[number];

export function normalizeProxyMode(mode: string): ProxyMode {
  switch (mode) {
    case "custom":
      return "custom";
    case "off":
      return "off";
    default:
      return "auto";
  }
}

export function proxyModeLabel(mode: ProxyMode, t: ReturnType<typeof useT>): string {
  switch (mode) {
    case "auto":
      return t("settings.proxyMode.auto");
    case "custom":
      return t("settings.proxyMode.custom");
    case "off":
      return t("settings.proxyMode.off");
  }
}

// ── Shared UI components ───────────────────────────────────────────────────

export function ToggleSegment({
  value,
  disabled,
  onLabel,
  offLabel,
  onChange,
}: {
  value: boolean;
  disabled: boolean;
  onLabel?: string;
  offLabel?: string;
  onChange: (value: boolean) => void;
}) {
  const t = useT();
  return (
    <div className="set-seg">
      <button
        type="button"
        className={`set-seg__btn${value ? " set-seg__btn--on" : ""}`}
        disabled={disabled}
        onClick={() => onChange(true)}
      >
        {onLabel ?? t("settings.toggleOn")}
      </button>
      <button
        type="button"
        className={`set-seg__btn${!value ? " set-seg__btn--on" : ""}`}
        disabled={disabled}
        onClick={() => onChange(false)}
      >
        {offLabel ?? t("settings.toggleOff")}
      </button>
    </div>
  );
}
