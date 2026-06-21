import { memo, useEffect, useMemo, useRef, useState } from "react";
import { Check, ChevronDown } from "lucide-react";
import { app } from "../lib/bridge";
import { useT, type DictKey } from "../lib/i18n";
import { apiKeyEnvFromProviderName, inferredVisionModels, mergedFetchedProviderModels, providerApiKeyEnvForSave, providerDefaultModel, providerIsConfigured, providerModelCandidates, providerRequiresKey } from "../lib/providerModels";
import type { ProviderView, SettingsView } from "../lib/types";
import { InlineConfirmButton } from "./InlineConfirmButton";
import { AnchoredPopover } from "./AnchoredPopover";
import { Tooltip } from "./Tooltip";
import { SettingsSection, SettingsField, allRefs, toRef, uniqueStrings, type SectionProps } from "./settings-shared";

// ── Model constants ────────────────────────────────────────────────────────

const EFFORT_PRESETS: readonly string[] = ["low", "medium", "high", "xhigh", "max"];
const REASONING_PROTOCOLS: readonly string[] = ["", "deepseek", "openai", "none"];

function normalizeReasoningProtocol(protocol: string | undefined): string {
  return REASONING_PROTOCOLS.includes(protocol ?? "") ? protocol ?? "" : "";
}

function reasoningProtocolLabel(protocol: string, t: ReturnType<typeof useT>): string {
  switch (protocol) {
    case "deepseek":
      return t("settings.reasoningProtocol.deepseek");
    case "openai":
      return t("settings.reasoningProtocol.openai");
    case "none":
      return t("settings.reasoningProtocol.none");
    default:
      return t("settings.reasoningProtocol.auto");
  }
}

// ── Model section components ───────────────────────────────────────────────

export type ModelsSectionProps = SectionProps & {
  backgroundApply: (fn: () => Promise<void>) => Promise<void>;
};

export function settingsModelMeta(s: SettingsView, t: ReturnType<typeof useT>): string {
  const ref = toRef(s.defaultModel, s);
  if (!ref) return t("common.none");
  if (!ref.includes("/")) return ref;
  const [provider, ...modelParts] = ref.split("/");
  const model = modelParts.join("/") || ref;
  const providerView = s.providers.find((p) => p.name === provider);
  return `${modelProviderLabel(provider, providerView, t)} · ${model}`;
}

export function StepLimitControl({
  value,
  presets,
  busy,
  onChange,
}: {
  value: number;
  presets: number[];
  busy: boolean;
  onChange: (value: number) => void;
}) {
  const t = useT();
  const normalized = normalizeStepLimit(value);
  const presetSet = new Set(presets.map(normalizeStepLimit));
  const [custom, setCustom] = useState(String(normalized));
  useEffect(() => setCustom(String(normalized)), [normalized]);
  const isCustom = !presetSet.has(normalized);
  const commitCustom = () => {
    const next = normalizeStepLimit(Number(custom));
    setCustom(String(next));
    if (next !== normalized) onChange(next);
  };
  return (
    <div className="step-limit-control">
      <div className="set-seg">
        {presets.map((preset) => {
          const n = normalizeStepLimit(preset);
          return (
            <button
              key={n}
              type="button"
              className={`set-seg__btn${normalized === n ? " set-seg__btn--on" : ""}`}
              disabled={busy}
              onClick={() => n !== normalized && onChange(n)}
            >
              {stepLimitLabel(n, t)}
            </button>
          );
        })}
        <button
          type="button"
          className={`set-seg__btn${isCustom ? " set-seg__btn--on" : ""}`}
          disabled={busy}
          onClick={() => {
            if (!isCustom) setCustom(String(normalized || 12));
          }}
        >
          {t("settings.stepLimit.custom")}
        </button>
      </div>
      <input
        className="mem-input step-limit-control__custom"
        value={custom}
        disabled={busy}
        inputMode="numeric"
        aria-label={t("settings.stepLimit.custom")}
        onChange={(e) => setCustom(e.target.value.replace(/[^\d]/g, ""))}
        onBlur={commitCustom}
        onKeyDown={(e) => {
          if (e.key === "Enter") e.currentTarget.blur();
        }}
      />
    </div>
  );
}

function normalizeStepLimit(value: number): number {
  return Number.isFinite(value) && value > 0 ? Math.trunc(value) : 0;
}

function stepLimitLabel(value: number, t: ReturnType<typeof useT>): string {
  return value === 0 ? t("settings.stepLimit.unlimited") : String(value);
}


export function ModelsSection({ s, busy, apply, backgroundApply }: ModelsSectionProps) {
  const t = useT();
  const [subtab, setSubtab] = useState<"usage" | "access">("usage");
  const autoRefreshKeyRef = useRef("");
  const refs = useMemo(() => allRefs(s), [s.providers]);
  const defaultRef = toRef(s.defaultModel, s);
  const plannerRef = toRef(s.plannerModel, s);
  const subagentRef = toRef(s.subagentModel, s);
  const plannerSelectRef = plannerRef === defaultRef ? "" : plannerRef;
  const [defaultProvider] = defaultRef.split("/");
  const defaultProviderView = s.providers.find((p) => p.name === defaultProvider);
  const modelIssue = !defaultProviderView
    ? t("settings.modelUnavailable", { ref: defaultRef || t("common.none") })
    : !providerIsConfigured(defaultProviderView)
      ? t("settings.modelNeedsKey", { provider: modelProviderLabel(defaultProvider, defaultProviderView, t) })
      : "";
  const agent = s.agent ?? { temperature: 0, maxSteps: 0, plannerMaxSteps: 0, systemPrompt: "", coldResumePrune: true, reasoningLanguage: "auto" };
  const setAgentSteps = (maxSteps: number, plannerMaxSteps: number) => (
    app.SetAgentParams(agent.temperature, maxSteps, plannerMaxSteps, agent.systemPrompt)
  );

  useEffect(() => {
    if (subtab !== "usage") return;
    const groups = providerAccessGroups(s.providers.filter((p) => p.added), t);
    const candidates = groups
      .map((group) => {
        const provider = group.providers.find((p) => providerIsConfigured(p) && p.baseUrl);
        return provider ? { group, provider } : null;
      })
      .filter((item): item is { group: ProviderAccessGroup; provider: ProviderView } => Boolean(item));
    const refreshKey = candidates.map(({ group, provider }) => `${group.id}:${provider.apiKeyEnv || provider.name}:${provider.baseUrl}`).join("|");
    if (!refreshKey || autoRefreshKeyRef.current === refreshKey) return;
    autoRefreshKeyRef.current = refreshKey;

    void backgroundApply(async () => {
      for (const { provider } of candidates) {
        // Background auto-refresh only protects a user-curated model list.
        // If the user hasn't specified any models, don't silently populate
        // the provider with every model from the API.
        if (!provider.models || provider.models.length === 0) continue;
        try {
          const fetched = await app.FetchProviderModels(provider);
          if (fetched.length === 0) continue;
          const models = mergedFetchedProviderModels(provider.models, fetched, { preserveCurated: true });
          const currentDefault = providerDefaultModel(provider.default, models);
          const visionModels = provider.visionModels.filter((model) => models.includes(model));
          if (sameStringList(provider.models, models) && provider.default === currentDefault && sameStringList(provider.visionModels, visionModels)) continue;
          await app.SaveProvider({ ...provider, models, default: currentDefault, visionModels });
        } catch {
          // Background discovery is opportunistic; manual refresh shows errors.
        }
      }
    });
  }, [backgroundApply, s.providers, subtab, t]);

  return (
    <>
      <div className="settings-subtabs">
        <button
          type="button"
          className={`settings-subtab${subtab === "usage" ? " settings-subtab--active" : ""}`}
          aria-selected={subtab === "usage"}
          onClick={() => setSubtab("usage")}
        >
          {t("settings.modelTab.usage")}
        </button>
        <button
          type="button"
          className={`settings-subtab${subtab === "access" ? " settings-subtab--active" : ""}`}
          aria-selected={subtab === "access"}
          onClick={() => setSubtab("access")}
        >
          {t("settings.modelTab.access")}
        </button>
      </div>

      {subtab === "usage" ? (
        <>
          <SettingsSection title={t("settings.modelUsage")}>
            <SettingsField label={t("settings.defaultModel")}>
              <ModelPicker
                s={s}
                refs={refs}
                value={toRef(s.defaultModel, s)}
                disabled={busy}
                onPick={(ref) => void apply(() => app.SetDefaultModel(ref))}
              />
            </SettingsField>

            <SettingsField label={t("settings.plannerModel")}>
              <ModelPicker
                s={s}
                refs={refs}
                value={plannerSelectRef}
                disabled={busy}
                includeSameDefault
                onPick={(ref) => void apply(() => app.SetPlannerModel(ref))}
              />
            </SettingsField>

            <SettingsField label={t("settings.subagentModel")}>
              <ModelPicker
                s={s}
                refs={refs}
                value={subagentRef}
                disabled={busy}
                emptyOptionLabel={t("settings.subagentModelDefault")}
                emptyOptionHint={t("common.auto")}
                onPick={(ref) => void apply(() => app.SetSubagentModel(ref))}
              />
            </SettingsField>

            <SettingsField label={t("settings.subagentEffort")} hint={t("settings.subagentHint")}>
              <select
                className="mem-select set-grow"
                value={s.subagentEffort || ""}
                disabled={busy}
                onChange={(e) => void apply(() => app.SetSubagentEffort(e.target.value))}
              >
                <option value="">{t("settings.subagentEffortDefault")}</option>
                {EFFORT_PRESETS.map((level) => (
                  <option key={level} value={level}>
                    {level}
                  </option>
                ))}
              </select>
            </SettingsField>

            {modelIssue && <div className="provider-fetch-banner provider-fetch-banner--warn">{modelIssue}</div>}
          </SettingsSection>
          <SettingsSection title={t("settings.agentRuntime")} description={t("settings.agentRuntimeHint")}>
            <SettingsField label={t("settings.executorMaxSteps")} hint={t("settings.executorMaxStepsHint")}>
              <StepLimitControl
                value={agent.maxSteps}
                presets={[10, 25, 50, 0]}
                busy={busy}
                onChange={(next) => void apply(() => setAgentSteps(next, agent.plannerMaxSteps))}
              />
            </SettingsField>
            <SettingsField label={t("settings.plannerMaxSteps")} hint={plannerSelectRef ? t("settings.plannerMaxStepsHint") : t("settings.plannerMaxStepsDisabledHint")}>
              <StepLimitControl
                value={agent.plannerMaxSteps}
                presets={[6, 12, 25, 0]}
                busy={busy}
                onChange={(next) => void apply(() => setAgentSteps(agent.maxSteps, next))}
              />
            </SettingsField>
            <SettingsField label={t("settings.coldResumePrune")} hint={t("settings.coldResumePruneHint")}>
              <div className="set-seg">
                {([true, false] as const).map((on) => (
                  <button
                    key={on ? "on" : "off"}
                    className={`set-seg__btn${agent.coldResumePrune === on ? " set-seg__btn--on" : ""}`}
                    disabled={busy}
                    onClick={() => void apply(() => app.SetColdResumePrune(on))}
                  >
                    {on ? t("settings.coldResumePrune.on") : t("settings.coldResumePrune.off")}
                  </button>
                ))}
              </div>
            </SettingsField>
            <SettingsField label={t("settings.reasoningLanguage")} hint={t("settings.reasoningLanguageHint")}>
              <div className="set-seg">
                {(["auto", "zh", "en"] as const).map((lang) => (
                  <button
                    key={lang}
                    className={`set-seg__btn${agent.reasoningLanguage === lang ? " set-seg__btn--on" : ""}`}
                    disabled={busy}
                    onClick={() => void apply(() => app.SetReasoningLanguage(lang))}
                  >
                    {t(`settings.reasoningLanguage.${lang}`)}
                  </button>
                ))}
              </div>
            </SettingsField>
          </SettingsSection>
        </>
      ) : (
        <ProvidersSection s={s} busy={busy} apply={apply} />
      )}
    </>
  );
}

type ModelPickerOption = {
  ref: string;
  provider: string;
  model: string;
  providerView?: ProviderView;
};

export function ModelPicker({
  s,
  refs,
  value,
  disabled,
  includeSameDefault = false,
  emptyOptionLabel,
  emptyOptionHint,
  onPick,
}: {
  s: SettingsView;
  refs: string[];
  value: string;
  disabled: boolean;
  includeSameDefault?: boolean;
  emptyOptionLabel?: string;
  emptyOptionHint?: string;
  onPick: (ref: string) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const triggerRef = useRef<HTMLButtonElement>(null);
  // Debounce search to avoid expensive filtering on every keystroke
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 150);
    return () => clearTimeout(timer);
  }, [query]);
  const q = debouncedQuery.trim().toLowerCase();
  const emptyLabel = includeSameDefault ? t("settings.plannerNone") : emptyOptionLabel;
  const emptyHint = includeSameDefault ? t("settings.plannerNoneHint") : emptyOptionHint;
  const emptyMeta = includeSameDefault ? t("settings.plannerNoneHintShort") : emptyOptionHint;
  const selected = refs.includes(value) ? modelOptionFromRef(value, s) : null;
  const selectedLabel = value === "" && emptyLabel
    ? emptyLabel
    : selected?.model || value || t("common.none");
  const selectedMeta = value === "" && emptyLabel
    ? emptyMeta || ""
    : selected
    ? modelOptionMeta(selected, t)
    : t("settings.noModelsConfigured");
  const emptyOptionVisible = Boolean(emptyLabel) && (!q || `${emptyLabel} ${emptyHint || ""}`.toLowerCase().includes(q));

  const groups = useMemo(() => {
    const providerOrder: string[] = [];
    const providerSeen = new Set<string>();
    for (const p of s.providers) {
      const id = providerGroupID(p);
      if (!providerSeen.has(id)) {
        providerOrder.push(id);
        providerSeen.add(id);
      }
    }
    const options = refs
      .map((ref) => modelOptionFromRef(ref, s))
      .filter((opt): opt is ModelPickerOption => Boolean(opt))
      .filter((opt) => !q || `${opt.ref} ${opt.provider} ${modelProviderLabel(opt.provider, opt.providerView, t)} ${opt.model}`.toLowerCase().includes(q));
    for (const opt of options) {
      const groupID = modelOptionGroupID(opt);
      if (!providerSeen.has(groupID)) {
        providerOrder.push(groupID);
        providerSeen.add(groupID);
      }
    }
    return providerOrder
      .map((groupID) => {
        const providerViews = s.providers.filter((p) => providerGroupID(p) === groupID);
        const firstProvider = providerViews[0];
        return {
          groupID,
          label: firstProvider ? providerGroupLabel(firstProvider, t) : groupID,
          keySet: providerViews.some((p) => p.keySet),
          requiresKey: providerViews.every((p) => providerRequiresKey(p)),
          options: uniqueModelOptions(options.filter((opt) => modelOptionGroupID(opt) === groupID)),
        };
      })
      .filter((group) => group.options.length > 0);
  }, [q, refs, s, t]);

  useEffect(() => {
    if (!open) setQuery("");
  }, [open]);

  const pick = (ref: string) => {
    setOpen(false);
    if (ref !== value) onPick(ref);
  };

  return (
    <div className="settings-model-picker">
      <button
        ref={triggerRef}
        type="button"
        className="settings-model-picker__trigger"
        disabled={disabled || (!includeSameDefault && !emptyOptionLabel && refs.length === 0)}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((next) => !next)}
      >
        <span className="settings-model-picker__selected">
          <span>{selectedLabel}</span>
          <small>{selectedMeta}</small>
        </span>
        <ChevronDown size={16} className={`settings-model-picker__chev${open ? " settings-model-picker__chev--open" : ""}`} />
      </button>
      <AnchoredPopover
        open={open && !disabled}
        anchorRef={triggerRef}
        onClose={() => setOpen(false)}
        className="settings-model-picker__menu"
        placement="bottom"
        style={{ width: triggerRef.current?.getBoundingClientRect().width }}
      >
        <div className="settings-model-picker__search">
          <input
            value={query}
            placeholder={t("settings.searchModels")}
            onChange={(e) => setQuery(e.target.value)}
            autoFocus
          />
        </div>
        <div className="settings-model-picker__list" role="listbox">
          {emptyOptionVisible && (
            <button
              type="button"
              role="option"
              aria-selected={value === ""}
              className={`settings-model-picker__option settings-model-picker__option--pinned${value === "" ? " settings-model-picker__option--selected" : ""}`}
              onClick={() => pick("")}
            >
              <span>
                <strong>{emptyLabel}</strong>
                {emptyHint && <small>{emptyHint}</small>}
              </span>
              {value === "" && <Check size={14} />}
            </button>
          )}
          {groups.map((group) => (
            <div className="settings-model-picker__group" key={group.groupID}>
              <div className="settings-model-picker__group-title">
                <span>{group.label}</span>
                <small>{providerKeyStatusLabel(group, t)}</small>
              </div>
              {group.options.map((opt) => (
                <button
                  key={opt.ref}
                  type="button"
                  role="option"
                  aria-selected={opt.ref === value}
                  className={`settings-model-picker__option${opt.ref === value ? " settings-model-picker__option--selected" : ""}`}
                  onClick={() => pick(opt.ref)}
                >
                  <span>
                    <strong>{opt.model}</strong>
                    <small>{modelOptionMeta(opt, t)}</small>
                  </span>
                  {opt.ref === value && <Check size={14} />}
                </button>
              ))}
            </div>
          ))}
          {!emptyOptionVisible && groups.length === 0 && <div className="settings-model-picker__empty">{t("settings.noMatchingModels")}</div>}
        </div>
      </AnchoredPopover>
    </div>
  );
}

function modelOptionFromRef(ref: string, s: SettingsView): ModelPickerOption | null {
  if (!ref) return null;
  const [provider, ...modelParts] = ref.split("/");
  const model = modelParts.join("/") || ref;
  return {
    ref,
    provider,
    model,
    providerView: s.providers.find((p) => p.name === provider),
  };
}

function modelOptionMeta(option: ModelPickerOption, t: ReturnType<typeof useT>): string {
  const key = option.providerView ? providerKeyStatusLabel(option.providerView, t) : t("settings.noKey");
  return `${modelProviderLabel(option.provider, option.providerView, t)} · ${key}`;
}

function providerKeyStatusLabel(provider: { keySet: boolean; requiresKey?: boolean; apiKeyEnv?: string }, t: ReturnType<typeof useT>): string {
  if (!providerRequiresKey(provider)) return t("settings.noKeyRequired");
  return provider.keySet ? t("settings.keySet") : t("settings.noKey");
}

function modelProviderLabel(provider: string, providerView: ProviderView | undefined, t: ReturnType<typeof useT>): string {
  return providerView ? providerGroupLabel(providerView, t) : provider;
}

function modelOptionGroupID(option: ModelPickerOption): string {
  return option.providerView ? providerGroupID(option.providerView) : `custom:${option.provider}`;
}

function uniqueModelOptions(options: ModelPickerOption[]): ModelPickerOption[] {
  const seen = new Set<string>();
  const out: ModelPickerOption[] = [];
  for (const option of options) {
    if (seen.has(option.model)) continue;
    seen.add(option.model);
    out.push(option);
  }
  return out;
}

function sameStringList(a: string[], b: string[]): boolean {
  if (a.length !== b.length) return false;
  return a.every((value, i) => value === b[i]);
}

function ProvidersSection({ s, busy, apply }: SectionProps) {
  const t = useT();
  const defaultProvider = toRef(s.defaultModel, s).split("/")[0];
  const [editing, setEditing] = useState<string | null>(null);
  const [adding, setAdding] = useState<AddProviderMode>(null);
  const [fetchingProvider, setFetchingProvider] = useState<string | null>(null);
  const [fetchResults, setFetchResults] = useState<Record<string, ProviderFetchResult>>({});
  const [modelDrafts, setModelDrafts] = useState<Record<string, ProviderModelDraft>>({});
  const groups = useMemo(() => providerAccessGroups(s.providers.filter((p) => p.added), t), [s.providers, t]);

  const setGroupFetchResult = (groupID: string, result: ProviderFetchResult | null) => {
    setFetchResults((prev) => {
      const next = { ...prev };
      if (result) next[groupID] = result;
      else delete next[groupID];
      return next;
    });
  };

  const setGroupModelDraft = (groupID: string, draft: ProviderModelDraft | null) => {
    setModelDrafts((prev) => {
      const next = { ...prev };
      if (draft) next[groupID] = draft;
      else delete next[groupID];
      return next;
    });
  };

  const modelDraftForFetch = (p: ProviderView, fetched: string[]): ProviderModelDraft => {
    const candidates = providerModelCandidates(p.models, fetched);
    const selected = mergedFetchedProviderModels(p.models, fetched, { preserveCurated: true });
    const visionSource = p.visionModelsConfigured ? p.visionModels : inferredVisionModels(candidates);
    return {
      providerName: p.name,
      candidates,
      selected: candidates.filter((model) => selected.includes(model)),
      visionModels: candidates.filter((model) => visionSource.includes(model)),
    };
  };

  const updateModelDraftSelection = (groupID: string, nextSelected: (draft: ProviderModelDraft) => string[]) => {
    setModelDrafts((prev) => {
      const draft = prev[groupID];
      if (!draft) return prev;
      const selectedSet = new Set(nextSelected(draft));
      return {
        ...prev,
        [groupID]: {
          ...draft,
          selected: draft.candidates.filter((model) => selectedSet.has(model)),
        },
      };
    });
  };

  const toggleModelDraftVision = (groupID: string, model: string) => {
    setModelDrafts((prev) => {
      const draft = prev[groupID];
      if (!draft) return prev;
      return {
        ...prev,
        [groupID]: {
          ...draft,
          visionModels: draft.visionModels.includes(model)
            ? draft.visionModels.filter((candidate) => candidate !== model)
            : draft.candidates.filter((candidate) => candidate === model || draft.visionModels.includes(candidate)),
        },
      };
    });
  };

  const refreshModels = async (group: ProviderAccessGroup, p: ProviderView) => {
    setFetchingProvider(group.id);
    setGroupFetchResult(group.id, null);
    setGroupModelDraft(group.id, null);
    try {
      let fetched: string[];
      try {
        fetched = await app.FetchProviderModels(p);
      } catch (e) {
        setGroupFetchResult(group.id, {
          kind: "warn",
          text: t("settings.fetchModelsFailedForProvider", { provider: group.label, err: String((e as Error)?.message ?? e) }),
        });
        return;
      }
      if (fetched.length === 0) {
        setGroupFetchResult(group.id, {
          kind: "warn",
          text: t("settings.fetchModelsEmptyForProvider", { provider: group.label }),
        });
        return;
      }
      const draft = modelDraftForFetch(p, fetched);
      setGroupModelDraft(group.id, draft);
      setGroupFetchResult(group.id, {
        kind: "ok",
        text: t("settings.fetchModelsReadyForProvider", { provider: group.label, n: draft.candidates.length }),
      });
    } finally {
      setFetchingProvider(null);
    }
  };

  const refreshGroup = async (group: ProviderAccessGroup) => {
    const probe = group.providers[0];
    if (!probe) return;
    await refreshModels(group, probe);
  };

  const saveKeyEnvAndAutoRefresh = async (group: ProviderAccessGroup, apiKeyEnv: string, value: string) => {
    const probe = group.providers[0];
    if (!probe || !apiKeyEnv) return;
    setFetchingProvider(group.id);
    setGroupFetchResult(group.id, null);
    setGroupModelDraft(group.id, null);
    try {
      await apply(async () => {
        await app.SetProviderKey(apiKeyEnv, value);
        try {
          const fetched = await app.FetchProviderModels({ ...probe, apiKeyEnv });
          if (fetched.length > 0) {
            const draft = modelDraftForFetch({ ...probe, apiKeyEnv }, fetched);
            setGroupModelDraft(group.id, draft);
            setGroupFetchResult(group.id, {
              kind: "ok",
              text: t("settings.fetchModelsReadyForProvider", { provider: group.label, n: draft.candidates.length }),
            });
            return;
          }
          setGroupFetchResult(group.id, {
            kind: "warn",
            text: t("settings.fetchModelsEmptyForProvider", { provider: group.label }),
          });
        } catch (e) {
          setGroupFetchResult(group.id, {
            kind: "warn",
            text: t("settings.fetchModelsAfterKeyFailedForProvider", { provider: group.label, err: String((e as Error)?.message ?? e) }),
          });
        }
      });
    } finally {
      setFetchingProvider(null);
    }
  };

  const saveProviderKey = async (group: ProviderAccessGroup, apiKeyEnv: string, value: string) => {
    if (!apiKeyEnv) return;
    setGroupFetchResult(group.id, null);
    setGroupModelDraft(group.id, null);
    await apply(() => app.SetProviderKey(apiKeyEnv, value));
  };

  const clearProviderKey = async (apiKeyEnv: string) => {
    if (!apiKeyEnv) return;
    await apply(() => app.ClearProviderKey(apiKeyEnv));
  };

  const saveModelDraft = async (group: ProviderAccessGroup) => {
    const draft = modelDrafts[group.id];
    const provider = draft ? group.providers.find((p) => p.name === draft.providerName) : null;
    const models = uniqueStrings(draft?.selected ?? []);
    const visionModels = uniqueStrings(draft?.visionModels ?? []).filter((model) => models.includes(model));
    if (!draft || !provider || models.length === 0) return;
    let saved = false;
    await apply(async () => {
      await app.SaveProvider({
        ...provider,
        models,
        visionModels,
        visionModelsConfigured: true,
        default: providerDefaultModel(provider.default, models),
      });
      saved = true;
    });
    if (!saved) return;
    setGroupModelDraft(group.id, null);
    setGroupFetchResult(group.id, {
      kind: "ok",
      text: t("settings.enabledModelsSavedForProvider", { provider: group.label, n: models.length }),
    });
  };

  return (
    <SettingsSection
      title={t("settings.providerAccess")}
      description={t("settings.providerAccessHint")}
      actions={
        <button className="btn btn--small" disabled={busy || adding !== null} onClick={() => setAdding("official")}>
          {t("settings.addProvider")}
        </button>
      }
    >
      <div className="provider-access-grid">
        {groups.length === 0 && adding === null && (
          <div className="provider-empty">
            <strong>{t("settings.providerAccessEmptyTitle")}</strong>
            <span>{t("settings.providerAccessEmptyHint")}</span>
            <div className="provider-empty__actions">
              <button type="button" className="btn btn--small" disabled={busy} onClick={() => setAdding("official")}>
                {t("settings.addProvider.officialChoice")}
              </button>
              <button type="button" className="btn btn--small" disabled={busy} onClick={() => setAdding("custom")}>
                {t("settings.addProvider.customChoice")}
              </button>
            </div>
          </div>
        )}
        {adding !== null && (
          <AddProviderPanel
            mode={adding}
            kinds={s.providerKinds}
            busy={busy}
            onMode={setAdding}
            onCancel={() => setAdding(null)}
            onAddOfficial={(kind, key) => apply(() => app.AddOfficialProviderAccess(kind, key)).then(() => setAdding(null))}
            onAddCustom={(pv) => apply(() => app.SaveProvider(pv)).then(() => setAdding(null))}
          />
        )}
        {adding === null && groups.map((group) => (
          <ProviderAccessCard
            key={group.id}
            group={group}
            busy={busy}
            fetching={fetchingProvider === group.id || group.providers.some((p) => fetchingProvider === p.name)}
            fetchResult={fetchResults[group.id]}
            modelDraft={modelDrafts[group.id]}
            defaultProvider={defaultProvider}
            editing={editing}
            kinds={s.providerKinds}
            onEdit={setEditing}
            onCancelEdit={() => setEditing(null)}
            onSave={(pv) => apply(() => app.SaveProvider(pv)).then(() => {
              setEditing(null);
              setGroupModelDraft(group.id, null);
            })}
            onRefresh={() => void refreshGroup(group)}
            onToggleDraftModel={(model) => updateModelDraftSelection(group.id, (draft) => (
              draft.selected.includes(model)
                ? draft.selected.filter((candidate) => candidate !== model)
                : [...draft.selected, model]
            ))}
            onToggleDraftVision={(model) => toggleModelDraftVision(group.id, model)}
            onSelectAllDraftModels={() => updateModelDraftSelection(group.id, (draft) => draft.candidates)}
            onClearDraftModels={() => updateModelDraftSelection(group.id, () => [])}
            onCancelDraftModels={() => setGroupModelDraft(group.id, null)}
            onSaveDraftModels={() => void saveModelDraft(group)}
            onSaveEditorKey={(env, value) => group.builtIn ? saveProviderKey(group, env, value) : saveKeyEnvAndAutoRefresh(group, env, value)}
            onClearEditorKey={clearProviderKey}
            onDelete={(p) => apply(() => app.RemoveProviderAccess(p.name))}
          />
        ))}
      </div>
    </SettingsSection>
  );
}

type ProviderAccessGroup = {
  id: string;
  label: string;
  description: string;
  builtIn: boolean;
  providers: ProviderView[];
  apiKeyEnv: string;
  keySet: boolean;
  requiresKey: boolean;
  configured: boolean;
  keySource?: string;
  keySourcePath?: string;
  baseUrl: string;
  kind: string;
  models: string[];
};

type ProviderFetchResult = {
  kind: "ok" | "warn";
  text: string;
};

type ProviderModelDraft = {
  providerName: string;
  candidates: string[];
  selected: string[];
  visionModels: string[];
};

type AddProviderMode = null | "official" | "custom";
type OfficialProviderKind = "deepseek";

const OFFICIAL_PROVIDER_CHOICES: Array<{ kind: OfficialProviderKind; labelKey: DictKey; descKey: DictKey; keyEnv: string }> = [
  { kind: "deepseek", labelKey: "settings.addProvider.official.deepseek", descKey: "settings.addProvider.official.deepseekDesc", keyEnv: "DEEPSEEK_API_KEY" },
];

function AddProviderPanel({
  mode,
  kinds,
  busy,
  onMode,
  onCancel,
  onAddOfficial,
  onAddCustom,
}: {
  mode: AddProviderMode;
  kinds: string[];
  busy: boolean;
  onMode: (mode: AddProviderMode) => void;
  onCancel: () => void;
  onAddOfficial: (kind: OfficialProviderKind, key: string) => Promise<void>;
  onAddCustom: (p: ProviderView) => void | Promise<void>;
}) {
  const t = useT();
  const [officialKind, setOfficialKind] = useState<OfficialProviderKind>("deepseek");
  const [key, setKey] = useState("");
  const selected = OFFICIAL_PROVIDER_CHOICES.find((choice) => choice.kind === officialKind) ?? OFFICIAL_PROVIDER_CHOICES[0];

  const header = (
    <div className="provider-add-panel__head">
      <div>
        <strong>{t("settings.addProvider.chooseTitle")}</strong>
        <span>{t("settings.addProvider.chooseHint")}</span>
      </div>
      <button type="button" className="btn btn--small" disabled={busy} onClick={onCancel}>
        {t("common.cancel")}
      </button>
    </div>
  );
  const modeSwitch = (
    <div className="provider-add-segmented" role="tablist" aria-label={t("settings.addProvider.chooseTitle")}>
      <button
        type="button"
        role="tab"
        aria-selected={mode === "official"}
        className={mode === "official" ? "provider-add-segmented__item provider-add-segmented__item--active" : "provider-add-segmented__item"}
        disabled={busy}
        onClick={() => onMode("official")}
      >
        {t("settings.addProvider.officialChoice")}
      </button>
      <button
        type="button"
        role="tab"
        aria-selected={mode === "custom"}
        className={mode === "custom" ? "provider-add-segmented__item provider-add-segmented__item--active" : "provider-add-segmented__item"}
        disabled={busy}
        onClick={() => onMode("custom")}
      >
        {t("settings.addProvider.customChoice")}
      </button>
    </div>
  );

  if (mode === "official") {
    return (
      <div className="provider-add-panel">
        {header}
        {modeSwitch}
        <div className="provider-add-panel__hint">{t("settings.addProvider.officialHint")}</div>
        <div className="provider-template-grid">
          {OFFICIAL_PROVIDER_CHOICES.map((choice) => (
            <button
              key={choice.kind}
              type="button"
              className={`provider-template-card${officialKind === choice.kind ? " provider-template-card--active" : ""}`}
              disabled={busy}
              onClick={() => setOfficialKind(choice.kind)}
            >
              <strong>{t(choice.labelKey)}</strong>
              <span>{t(choice.descKey)}</span>
            </button>
          ))}
        </div>
        <label className="set-label">{t("settings.providerKeyOptional")}</label>
        <input
          className="mem-input"
          type="password"
          placeholder={t("settings.setKey", { env: selected.keyEnv })}
          value={key}
          disabled={busy}
          onChange={(e) => setKey(e.target.value)}
        />
        <div className="prov-card__actions">
          <button type="button" className="btn btn--small" disabled={busy} onClick={onCancel}>
            {t("common.cancel")}
          </button>
          <button
            type="button"
            className="btn btn--primary btn--small"
            disabled={busy}
            onClick={() => void onAddOfficial(officialKind, key.trim())}
          >
            {t("settings.addProvider.confirm")}
          </button>
        </div>
      </div>
    );
  }

  if (mode === "custom") {
    return (
      <div className="provider-add-panel">
        {header}
        {modeSwitch}
        <div className="provider-add-panel__hint">{t("settings.addProvider.customHint")}</div>
        <ProviderEditor
          kinds={kinds}
          busy={busy}
          onCancel={onCancel}
          onSave={onAddCustom}
        />
      </div>
    );
  }
  return null;
}

function ProviderAccessCard({
  group,
  busy,
  fetching,
  fetchResult,
  modelDraft,
  defaultProvider,
  editing,
  kinds,
  onEdit,
  onCancelEdit,
  onSave,
  onRefresh,
  onToggleDraftModel,
  onToggleDraftVision,
  onSelectAllDraftModels,
  onClearDraftModels,
  onCancelDraftModels,
  onSaveDraftModels,
  onSaveEditorKey,
  onClearEditorKey,
  onDelete,
}: {
  group: ProviderAccessGroup;
  busy: boolean;
  fetching: boolean;
  fetchResult?: ProviderFetchResult;
  modelDraft?: ProviderModelDraft;
  defaultProvider: string;
  editing: string | null;
  kinds: string[];
  onEdit: (name: string) => void;
  onCancelEdit: () => void;
  onSave: (p: ProviderView) => void | Promise<void>;
  onRefresh: () => void;
  onToggleDraftModel: (model: string) => void;
  onToggleDraftVision: (model: string) => void;
  onSelectAllDraftModels: () => void;
  onClearDraftModels: () => void;
  onCancelDraftModels: () => void;
  onSaveDraftModels: () => void;
  onSaveEditorKey: (apiKeyEnv: string, value: string) => Promise<void>;
  onClearEditorKey?: (apiKeyEnv: string) => Promise<void>;
  onDelete?: (p: ProviderView) => Promise<void>;
}) {
  const t = useT();
  const editableProvider = group.providers[0];
  const isDefault = group.providers.some((p) => p.name === defaultProvider);
  const editingProvider = group.providers.find((p) => editing === p.name);
  const primaryProviderExpanded = Boolean(editableProvider && editing === editableProvider.name);
  const visibleModels = group.models.slice(0, 6);
  const hiddenModelCount = Math.max(0, group.models.length - visibleModels.length);
  return (
    <article className={`provider-access-card${group.builtIn ? " provider-access-card--builtin" : ""}`}>
      <div className="provider-access-card__head">
        <div className="provider-access-card__identity">
          <div className="provider-access-card__title">
            {group.label}
            <span className={`badge ${group.builtIn ? "badge--project" : "badge--neutral"}`}>
              {group.builtIn ? t("settings.builtinProviderBadge") : t("settings.customProviderBadge")}
            </span>
            <span className={`badge ${group.keySet ? "badge--project" : "badge--feedback"}`}>
              {providerKeyStatusLabel(group, t)}
            </span>
          </div>
          <div className="provider-access-card__desc">{group.description}</div>
        </div>
        <div className="provider-access-card__actions">
          {editableProvider && (
            <button
              className="btn btn--small"
              disabled={busy}
              aria-expanded={primaryProviderExpanded}
              onClick={() => primaryProviderExpanded ? onCancelEdit() : onEdit(editableProvider.name)}
            >
              {primaryProviderExpanded ? t("common.collapse") : t("settings.configureProvider")}
            </button>
          )}
          <button
            className="btn btn--small"
            disabled={busy || fetching || !group.baseUrl || !group.configured}
            onClick={onRefresh}
          >
            {fetching ? t("settings.fetchingModels") : t("settings.fetchModels")}
          </button>
          {editableProvider && onDelete && (
            isDefault && !group.builtIn ? (
              <Tooltip label={t("settings.cantDeleteDefault")}>
                <button className="btn btn--small" disabled>{t("settings.removeProviderAccess")}</button>
              </Tooltip>
            ) : (
              <InlineConfirmButton
                label={t("settings.removeProviderAccess")}
                confirmLabel={group.builtIn ? t("settings.confirmRemoveProviderAccess") : t("settings.confirmDeleteProvider")}
                cancelLabel={t("common.cancel")}
                disabled={busy}
                danger={!group.builtIn}
                onConfirm={() => onDelete(editableProvider)}
              />
            )
          )}
        </div>
      </div>

      <div className="provider-access-meta">
        <span>{group.kind}</span>
        <span>{group.baseUrl}</span>
        <span>{group.apiKeyEnv || t("common.none")}</span>
        {group.keySource && <span title={group.keySourcePath || undefined}>{t("settings.keySource", { source: group.keySource })}</span>}
      </div>

      <div className="provider-card-block">
        <div className="provider-card-block__label">{t(group.configured ? "settings.enabledModels" : "settings.modelList")}</div>
        <div className="provider-model-chips" aria-label={t(group.configured ? "settings.enabledModels" : "settings.modelList")}>
          {visibleModels.length > 0 ? visibleModels.map((model) => (
            <span className="provider-model-chip" key={model}>
              {model}
            </span>
          )) : <span className="provider-model-chip provider-model-chip--empty">{t("settings.noModelsConfigured")}</span>}
          {hiddenModelCount > 0 && (
            <span className="provider-model-chip provider-model-chip--more">
              {t("settings.moreModels", { n: hiddenModelCount })}
            </span>
          )}
        </div>
        {!group.configured && group.requiresKey && (
          <div className="provider-card-status provider-card-status--warn">
            {t("settings.modelsRequireKey")}
          </div>
        )}
        {fetchResult && (
          <div className={`provider-card-status provider-card-status--${fetchResult.kind}`}>
            {fetchResult.text}
          </div>
        )}
      </div>

      {modelDraft && (
        <ProviderModelDraftPicker
          draft={modelDraft}
          busy={busy}
          fetching={fetching}
          onToggle={onToggleDraftModel}
          onToggleVision={onToggleDraftVision}
          onSelectAll={onSelectAllDraftModels}
          onClear={onClearDraftModels}
          onCancel={onCancelDraftModels}
          onSave={onSaveDraftModels}
        />
      )}

      {group.providers.length > 1 && (
        <div className="provider-profiles">
          {group.providers.map((p) => {
            const profileExpanded = editing === p.name;
            return (
              <div className="provider-profile-row" key={p.name}>
                <span>{p.name}</span>
                <span>{p.models.join(", ") || t("common.none")}</span>
                <button
                  className="btn btn--small"
                  disabled={busy}
                  aria-expanded={profileExpanded}
                  onClick={() => profileExpanded ? onCancelEdit() : onEdit(p.name)}
                >
                  {profileExpanded ? t("common.collapse") : t("settings.configureProfile")}
                </button>
              </div>
            );
          })}
        </div>
      )}

      {editingProvider && (
        <ProviderEditor
          initial={editingProvider}
          kinds={kinds}
          busy={busy}
          onCancel={onCancelEdit}
          onSave={onSave}
          onSaveKey={onSaveEditorKey}
          onClearKey={onClearEditorKey}
        />
      )}
    </article>
  );
}

function ProviderModelDraftPicker({
  draft,
  busy,
  fetching,
  onToggle,
  onToggleVision,
  onSelectAll,
  onClear,
  onCancel,
  onSave,
}: {
  draft: ProviderModelDraft;
  busy: boolean;
  fetching: boolean;
  onToggle: (model: string) => void;
  onToggleVision: (model: string) => void;
  onSelectAll: () => void;
  onClear: () => void;
  onCancel: () => void;
  onSave: () => void;
}) {
  const t = useT();
  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  // Debounce search to avoid expensive filtering on every keystroke
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedQuery(query), 150);
    return () => clearTimeout(timer);
  }, [query]);
  const selected = new Set(draft.selected);
  const vision = new Set(draft.visionModels);
  const q = debouncedQuery.trim().toLowerCase();
  const visibleCandidates = q
    ? draft.candidates.filter((model) => model.toLowerCase().includes(q))
    : draft.candidates;
  const disabled = busy || fetching;

  return (
    <div className="provider-model-draft">
      <div className="provider-model-draft__head">
        <div>
          <div className="provider-card-block__label">{t("settings.modelCandidates")}</div>
          <span>{t("settings.modelCandidatesSelected", { n: draft.selected.length })}</span>
        </div>
        <div className="provider-model-draft__tools">
          <button type="button" className="btn btn--small" disabled={disabled || draft.selected.length === draft.candidates.length} onClick={onSelectAll}>
            {t("settings.selectAllModels")}
          </button>
          <button type="button" className="btn btn--small" disabled={disabled || draft.selected.length === 0} onClick={onClear}>
            {t("settings.clearModelSelection")}
          </button>
        </div>
      </div>
      <input
        className="mem-input provider-model-draft__search"
        placeholder={t("settings.modelCandidateSearch")}
        value={query}
        disabled={disabled}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div className="provider-model-draft__list" role="list" aria-label={t("settings.modelCandidates")}>
        {visibleCandidates.length > 0 ? visibleCandidates.map((model) => {
          const enabled = selected.has(model);
          return (
            <div className="provider-model-draft__option" key={model}>
              <label className="provider-model-draft__model">
                <input
                  type="checkbox"
                  checked={enabled}
                  disabled={disabled}
                  onChange={() => onToggle(model)}
                />
                <span>{model}</span>
              </label>
              <label className="provider-model-draft__vision">
                <input
                  type="checkbox"
                  checked={enabled && vision.has(model)}
                  disabled={disabled || !enabled}
                  onChange={() => onToggleVision(model)}
                />
                <span>{t("settings.visionModel")}</span>
              </label>
            </div>
          );
        }) : (
          <div className="provider-model-draft__empty">{t("settings.noMatchingCandidateModels")}</div>
        )}
      </div>
      <div className="provider-model-draft__actions">
        <button type="button" className="btn btn--small" disabled={disabled} onClick={onCancel}>
          {t("common.cancel")}
        </button>
        <button type="button" className="btn btn--primary btn--small" disabled={disabled || draft.selected.length === 0} onClick={onSave}>
          {t("settings.saveEnabledModels")}
        </button>
      </div>
    </div>
  );
}

function providerAccessGroups(providers: ProviderView[], t: ReturnType<typeof useT>): ProviderAccessGroup[] {
  const groups = new Map<string, ProviderAccessGroup>();
  for (const p of providers) {
    const id = providerGroupID(p);
    const builtIn = id.startsWith("builtin:");
    const existing = groups.get(id);
    if (existing) {
      existing.providers.push(p);
      existing.keySet = existing.keySet || p.keySet;
      existing.requiresKey = existing.requiresKey && providerRequiresKey(p);
      existing.configured = existing.configured || providerIsConfigured(p);
      if (!existing.keySource && p.keySource) existing.keySource = p.keySource;
      if (!existing.keySourcePath && p.keySourcePath) existing.keySourcePath = p.keySourcePath;
      existing.models = uniqueStrings([...existing.models, ...p.models]);
      continue;
    }
    groups.set(id, {
      id,
      label: providerGroupLabel(p, t),
      description: providerGroupDescription(p, t),
      builtIn,
      providers: [p],
      apiKeyEnv: p.apiKeyEnv,
      keySet: p.keySet,
      requiresKey: providerRequiresKey(p),
      configured: providerIsConfigured(p),
      keySource: p.keySource,
      keySourcePath: p.keySourcePath,
      baseUrl: p.baseUrl,
      kind: p.kind,
      models: uniqueStrings(p.models),
    });
  }
  return Array.from(groups.values());
}

function providerBaseHost(baseUrl: string): string {
  try {
    return new URL(baseUrl).hostname.toLowerCase();
  } catch {
    return "";
  }
}

function canonicalOfficialProviderName(name: string): string {
  switch (name.trim()) {
    case "deepseek-flash":
    case "deepseek-pro":
      return "deepseek";
    default:
      return name.trim();
  }
}

function officialProviderKind(p: ProviderView): string {
  if (!p.builtIn) return "";
  const name = canonicalOfficialProviderName(p.name);
  const host = providerBaseHost(p.baseUrl);
  if (name === "deepseek" && host === "api.deepseek.com") return "deepseek";
  return "";
}

function providerGroupID(p: ProviderView): string {
  const official = officialProviderKind(p);
  if (official) return `builtin:${official}`;
  return `custom:${p.name}`;
}

function providerGroupLabel(p: ProviderView, t?: ReturnType<typeof useT>): string {
  const id = providerGroupID(p);
  if (id === "builtin:deepseek") return t ? t("settings.providerLabel.deepseek") : "DeepSeek";
  return p.name;
}

function providerGroupDescription(p: ProviderView, t: ReturnType<typeof useT>): string {
  const id = providerGroupID(p);
  if (id === "builtin:deepseek") return t("settings.providerDesc.deepseek");
  return p.baseUrl;
}

function parseProviderListInput(value: string): string[] {

  return uniqueStrings(value
    .split(/[,，]/)
    .map((entry) => entry.trim())
    .filter(Boolean));
}

// Memoized model chips for ProviderEditor — prevents re-render when typing
// in name/key/baseUrl fields.
const ModelChips = memo(function ModelChips({ modelNames }: { modelNames: string[] }) {
  const t = useT();
  if (modelNames.length === 0) return null;
  return (
    <div className="provider-model-chips">
      {modelNames.slice(0, 8).map((model) => (
        <span className="provider-model-chip" key={model}>{model}</span>
      ))}
      {modelNames.length > 8 && (
        <span className="provider-model-chip provider-model-chip--more">{t("settings.moreModels", { n: modelNames.length - 8 })}</span>
      )}
    </div>
  );
});

function ProviderEditor({
  initial,
  kinds,
  busy,
  onCancel,
  onSave,
  onSaveKey,
  onClearKey,
}: {
  initial?: ProviderView;
  kinds: string[];
  busy: boolean;
  onCancel: () => void;
  onSave: (p: ProviderView) => void;
  onSaveKey?: (apiKeyEnv: string, value: string) => Promise<void>;
  onClearKey?: (apiKeyEnv: string) => Promise<void>;
}) {
  const t = useT();
  const [name, setName] = useState(initial?.name ?? "");
  const [kind, setKind] = useState(initial?.kind ?? kinds[0] ?? "openai");
  const [baseUrl, setBaseUrl] = useState(initial?.baseUrl ?? "");
  const [models, setModels] = useState((initial?.models ?? []).join(", "));
  const [visionModels, setVisionModels] = useState((initial?.visionModels ?? []).join(", "));
  const [visionModelsConfigured, setVisionModelsConfigured] = useState(
    Boolean(initial?.visionModelsConfigured ?? ((initial?.visionModels ?? []).length > 0)),
  );
  const [modelsUrl] = useState(initial?.modelsUrl ?? "");
  const [apiKeyEnv, setApiKeyEnv] = useState(initial?.apiKeyEnv ?? "");
  const [keyDraft, setKeyDraft] = useState("");
  const [balanceUrl, setBalanceUrl] = useState(initial?.balanceUrl ?? "");
  // Empty when unset so the placeholder (and its "0 = default" hint) reads instead
  // of a bare "0"; saved back as 0.
  const [ctx, setCtx] = useState(initial?.contextWindow ? String(initial.contextWindow) : "");
  const [reasoningProtocol, setReasoningProtocol] = useState(normalizeReasoningProtocol(initial?.reasoningProtocol));
  const [supportedEfforts, setSupportedEfforts] = useState<string[]>(initial?.supportedEfforts ?? []);
  const [customEffortDraft, setCustomEffortDraft] = useState("");
  const [defaultEffort, setDefaultEffort] = useState(initial?.defaultEffort ?? "");
  const [fetchingModels, setFetchingModels] = useState(false);
  const [fetchStatus, setFetchStatus] = useState<string | null>(null);
  const [fetchErr, setFetchErr] = useState<string | null>(null);
  const [advancedOpen, setAdvancedOpen] = useState(false);
  const builtIn = initial?.builtIn ?? false;
  const isNewCustomProvider = !initial;

  // Offer the kinds the kernel actually registered; if the stored kind is a
  // legacy/unknown one, keep it as an option so editing doesn't silently change it.
  const kindOptions = kind && !kinds.includes(kind) ? [kind, ...kinds] : kinds;

  // Split supportedEfforts into the 5 canonical presets (for checkbox UI) and
  // any user-added custom names (rendered as removable chips). The preset order
  // is fixed; custom names keep insertion order.
  const presetEfforts = supportedEfforts.filter((e) => EFFORT_PRESETS.includes(e));
  const customEfforts = supportedEfforts.filter((e) => !EFFORT_PRESETS.includes(e));

  const togglePreset = (level: string) => {
    const has = presetEfforts.includes(level);
    const nextPresets = has ? presetEfforts.filter((e) => e !== level) : [...presetEfforts, level];
    setSupportedEfforts([...nextPresets, ...customEfforts]);
    // If the removed preset was the default, fall back to "auto" (empty string).
    if (has && defaultEffort === level) setDefaultEffort("");
  };

  const addCustomEffort = () => {
    const v = customEffortDraft.trim().toLowerCase();
    if (!v || supportedEfforts.includes(v)) {
      setCustomEffortDraft("");
      return;
    }
    setSupportedEfforts([...presetEfforts, ...customEfforts, v]);
    setCustomEffortDraft("");
  };

  const removeCustomEffort = (level: string) => {
    setSupportedEfforts(supportedEfforts.filter((e) => e !== level));
    if (defaultEffort === level) setDefaultEffort("");
  };

  const fetchModels = async () => {
    setFetchingModels(true);
    setFetchStatus(null);
    setFetchErr(null);
    try {
      const effectiveApiKeyEnv = providerApiKeyEnvForSave(name, apiKeyEnv, keyDraft);
      if (!apiKeyEnv.trim()) setApiKeyEnv(effectiveApiKeyEnv);
      if (keyDraft.trim()) await app.SetProviderKey(effectiveApiKeyEnv, keyDraft.trim());
      const fetched = await app.FetchProviderModels({
        name: name.trim() || t("settings.newProviderDraftName"),
        builtIn: initial?.builtIn ?? false,
        added: initial?.added ?? true,
        kind: kind.trim() || kinds[0] || "openai",
        baseUrl: baseUrl.trim(),
        modelsUrl,
        models: [],
        visionModels: [],
        visionModelsConfigured: false,
        default: "",
        apiKeyEnv: effectiveApiKeyEnv,
        keySet: Boolean(keyDraft.trim()) || (initial?.keySet ?? false),
        balanceUrl: balanceUrl.trim(),
        contextWindow: Number(ctx) || 0,
        reasoningProtocol,
        supportedEfforts,
        defaultEffort,
      });
      if (fetched.length === 0) throw new Error(t("settings.fetchModelsEmpty"));
      setModels(fetched.join(", "));
      setVisionModels((current) => {
        const existing = parseProviderListInput(current).filter((model) => fetched.includes(model));
        return uniqueStrings([...existing, ...inferredVisionModels(fetched)]).filter((model) => fetched.includes(model)).join(", ");
      });
      setVisionModelsConfigured(true);
      if (keyDraft.trim()) setKeyDraft("");
      setDefaultEffort((v) => v);
      setFetchStatus(t("settings.fetchModelsSuccess", { n: fetched.length }));
    } catch (e) {
      setFetchErr(String((e as Error)?.message ?? e));
    } finally {
      setFetchingModels(false);
    }
  };

  const save = async () => {
    const ms = parseProviderListInput(models);
    const vms = parseProviderListInput(visionModels).filter((model) => ms.includes(model));
    const effectiveApiKeyEnv = providerApiKeyEnvForSave(name, apiKeyEnv, keyDraft);
    if (keyDraft.trim()) await app.SetProviderKey(effectiveApiKeyEnv, keyDraft.trim());
    onSave({
      name: name.trim(),
      builtIn: initial?.builtIn ?? false,
      added: initial?.added ?? true,
      kind: kind.trim() || kinds[0] || "openai",
      baseUrl: baseUrl.trim(),
      models: ms,
      visionModels: vms,
      visionModelsConfigured: visionModelsConfigured || vms.length > 0,
      default: ms[0] ?? "",
      apiKeyEnv: effectiveApiKeyEnv,
      modelsUrl,
      keySet: Boolean(keyDraft.trim()) || (initial?.keySet ?? false),
      balanceUrl: balanceUrl.trim(),
      contextWindow: Number(ctx) || 0,
      reasoningProtocol,
      supportedEfforts,
      // Clear the stored default if no levels are selected; the backend's
      // NormalizeEffort would otherwise silently ignore an unsupported value.
      defaultEffort: supportedEfforts.length > 0 ? defaultEffort : "",
    });
  };

  if (builtIn) {
    const keyEnv = initial?.apiKeyEnv.trim() ?? "";
    return (
      <div className="provider-editor provider-editor--builtin provider-editor--key-only">
        {initial && onSaveKey && keyEnv && (
          <>
            <div className="provider-key-status provider-key-status--managed provider-key-status--compact">
              <span title={initial.keySourcePath || undefined}>
                {initial.keySet ? t("settings.configuredKey", { env: keyEnv }) : t("settings.notConfiguredKey", { env: keyEnv })}
                {initial.keySource ? ` · ${t("settings.keySource", { source: initial.keySource })}` : ""}
              </span>
              {initial.keySet && onClearKey && (
                <InlineConfirmButton
                  label={t("settings.clearKey")}
                  confirmLabel={t("settings.confirmClearKey")}
                  cancelLabel={t("common.cancel")}
                  disabled={busy}
                  danger
                  onConfirm={() => onClearKey(keyEnv)}
                />
              )}
            </div>
            <KeyField
              apiKeyEnv={keyEnv}
              busy={busy}
              keySet={initial.keySet}
              onSet={(env, value) => onSaveKey(env, value)}
            />
          </>
        )}
      </div>
    );
  }

  const modelNames = useMemo(
    () => models.split(",").map((m) => m.trim()).filter(Boolean),
    [models],
  );
  const canFetch = Boolean(name.trim() && baseUrl.trim());

  const protocolField = initial ? (
    <select className="mem-select" value={kind} onChange={(e) => setKind(e.target.value)}>
      {kindOptions.map((k) => (
        <option key={k} value={k}>
          {k === "openai" ? t("settings.providerProtocolOpenAI") : k}
        </option>
      ))}
    </select>
  ) : (
    <div className="provider-readonly-field provider-readonly-field--stacked" aria-readonly="true">
      <strong>{t("settings.providerProtocolOpenAI")}</strong>
      <span>{t("settings.providerProtocolOpenAIHint")}</span>
    </div>
  );

  const advancedFields = (
    <details className="provider-editor-advanced" open={advancedOpen} onToggle={(e) => setAdvancedOpen(e.currentTarget.open)}>
      <summary>{t("settings.providerAdvancedSettings")}</summary>
      <div className="provider-editor-advanced__body">
        <label className="set-label">{t("settings.providerApiKeyEnv")}</label>
        <input
          className="mem-input"
          placeholder={apiKeyEnvFromProviderName(name)}
          value={apiKeyEnv}
          onChange={(e) => setApiKeyEnv(e.target.value)}
        />
        <div className="mem-hint">{t("settings.providerApiKeyEnvHint")}</div>
        <label className="set-label">{t("settings.providerBalanceUrl")}</label>
        <input className="mem-input" placeholder={t("settings.balanceUrlPlaceholder")} value={balanceUrl} onChange={(e) => setBalanceUrl(e.target.value)} />
        <div className="mem-hint">{t("settings.balanceUrlHint")}</div>
        <label className="set-label">{t("settings.providerContextWindow")}</label>
        <input className="mem-input" placeholder={t("settings.contextWindowPlaceholder")} value={ctx} onChange={(e) => setCtx(e.target.value)} inputMode="numeric" />
        <div className="mem-hint">{t("settings.contextWindowHint")}</div>
        <label className="set-label">{t("settings.visionModels")}</label>
        <input
          className="mem-input"
          placeholder={t("settings.providerModels")}
          value={visionModels}
          onChange={(e) => {
            setVisionModelsConfigured(true);
            setVisionModels(e.target.value);
          }}
        />
        <div className="mem-hint">{t("settings.visionModelsHint")}</div>
        <label className="set-label">{t("settings.reasoningProtocol")}</label>
        <select className="mem-select" value={reasoningProtocol} onChange={(e) => setReasoningProtocol(e.target.value)}>
          {REASONING_PROTOCOLS.map((protocol) => (
            <option key={protocol || "auto"} value={protocol}>
              {reasoningProtocolLabel(protocol, t)}
            </option>
          ))}
        </select>
        <div className="mem-hint">{t("settings.reasoningProtocolHint")}</div>
        <label className="set-label">{t("settings.supportedEfforts")}</label>
        {EFFORT_PRESETS.map((level) => (
          <label key={level} className="set-check">
            <input
              type="checkbox"
              checked={presetEfforts.includes(level)}
              onChange={() => togglePreset(level)}
            />
            {level}
          </label>
        ))}
        <div className="set-row">
          <input
            className="mem-input set-grow"
            placeholder={t("settings.supportedEffortsCustomPlaceholder")}
            value={customEffortDraft}
            onChange={(e) => setCustomEffortDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                addCustomEffort();
              }
            }}
          />
          <button
            type="button"
            className="btn btn--small"
            disabled={
              !customEffortDraft.trim() || supportedEfforts.includes(customEffortDraft.trim().toLowerCase())
            }
            onClick={addCustomEffort}
          >
            {t("common.add")}
          </button>
        </div>
        {customEfforts.length > 0 && (
          <div className="set-rules__chips">
            {customEfforts.map((level) => (
              <span className="set-rule" key={level}>
                {level}
                <Tooltip label={t("common.delete")}>
                  <button
                    type="button"
                    className="set-rule__x"
                    disabled={busy}
                    onClick={() => removeCustomEffort(level)}
                  >
                    ×
                  </button>
                </Tooltip>
              </span>
            ))}
          </div>
        )}
        <div className="mem-hint">{t("settings.supportedEffortsHint")}</div>
        <label className="set-label">{t("settings.defaultEffort")}</label>
        {supportedEfforts.length > 0 ? (
          <select
            className="mem-select"
            value={defaultEffort}
            onChange={(e) => setDefaultEffort(e.target.value)}
          >
            <option value="">{t("settings.defaultEffortAuto")}</option>
            {supportedEfforts.map((level) => (
              <option key={level} value={level}>
                {level}
              </option>
            ))}
          </select>
        ) : (
          <select className="mem-select" value="" disabled>
            <option value="">{t("settings.defaultEffortAuto")}</option>
          </select>
        )}
        <div className="mem-hint">{t("settings.defaultEffortHint")}</div>
      </div>
    </details>
  );

  return (
    <div className={`provider-editor${isNewCustomProvider ? " provider-editor--wizard" : ""}`}>
      <label className="set-label">{t("settings.customProviderName")}</label>
      <input className="mem-input" placeholder={t("settings.customProviderNamePlaceholder")} value={name} onChange={(e) => setName(e.target.value)} disabled={!!initial} />
      <label className="set-label">{t("settings.providerProtocol")}</label>
      {protocolField}
      <label className="set-label">{t("settings.providerBaseUrlLabel")}</label>
      <input className="mem-input" placeholder={t("settings.providerBaseUrl")} value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} />
      {!initial && (
        <>
          <label className="set-label">{t("settings.providerKey")}</label>
          <input
            className="mem-input"
            type="password"
            placeholder={t("settings.providerKeyPlaceholder")}
            value={keyDraft}
            onChange={(e) => setKeyDraft(e.target.value)}
          />
        </>
      )}
      {initial && onSaveKey && apiKeyEnv.trim() && (
        <>
          <label className="set-label">{t("settings.providerKey")}</label>
          {initial.keySource && (
            <div className="mem-hint" title={initial.keySourcePath || undefined}>
              {t("settings.keySource", { source: initial.keySource })}
            </div>
          )}
          <KeyField
            apiKeyEnv={apiKeyEnv.trim()}
            busy={busy || fetchingModels}
            keySet={initial.keySet}
            onSet={(env, value) => onSaveKey(env, value)}
          />
        </>
      )}
      <div className="provider-model-fetch-row">
        <button
          type="button"
          className="btn btn--small"
          disabled={busy || fetchingModels || !canFetch}
          onClick={() => void fetchModels()}
        >
          {fetchingModels ? t("settings.fetchingModels") : t("settings.testFetchModels")}
        </button>
        <span>{t("settings.testFetchModelsHint")}</span>
      </div>
      {fetchStatus && <div className="provider-fetch-status provider-fetch-status--ok">{fetchStatus}</div>}
      {fetchErr && <div className="provider-fetch-status provider-fetch-status--error">{fetchErr}</div>}
      {modelNames.length > 0 && (
        <div className="provider-card-block">
          <div className="provider-card-block__label">{t("settings.availableModels")}</div>
          <ModelChips modelNames={modelNames} />
        </div>
      )}
      <label className="set-label">{t("settings.manualModels")}</label>
      <input className="mem-input" placeholder={t("settings.providerModels")} value={models} onChange={(e) => setModels(e.target.value)} />
      <div className="mem-hint">{t("settings.manualModelsHint")}</div>
      {advancedFields}
      <div className="prov-card__actions">
        <button className="btn btn--small" onClick={onCancel} disabled={busy}>
          {t("common.cancel")}
        </button>
        <button className="btn btn--primary btn--small" onClick={() => void save()} disabled={busy || !name.trim() || !baseUrl.trim() || !models.trim()}>
          {t("common.save")}
        </button>
      </div>
    </div>
  );
}

function KeyField({
  apiKeyEnv,
  busy,
  keySet = false,
  onSet,
}: {
  apiKeyEnv: string;
  busy: boolean;
  keySet?: boolean;
  onSet: (apiKeyEnv: string, value: string) => Promise<void>;
}) {
  const t = useT();
  const [val, setVal] = useState("");
  if (!apiKeyEnv) return null;
  return (
    <div className="set-key">
      <input
        className="mem-input"
        type="password"
        placeholder={t(keySet ? "settings.updateKey" : "settings.setKey", { env: apiKeyEnv })}
        value={val}
        onChange={(e) => setVal(e.target.value)}
      />
      <button
        className="btn btn--small"
        disabled={busy || !val.trim()}
        onClick={() => {
          void onSet(apiKeyEnv, val.trim());
          setVal("");
        }}
      >
        {t(keySet ? "settings.updateKeyAction" : "settings.saveKey")}
      </button>
    </div>
  );
}

