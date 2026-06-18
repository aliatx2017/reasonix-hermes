import { useState, useEffect, useCallback } from "react";
import { app } from "../../lib/bridge";
import type {
  CacheEconomyView,
  SessionTokensView,
  CompressStatsView,
  BotLiveStatusView,
  GoalProgressView,
  MemoryDashboardView,
  CostSummaryView,
  ScheduleDashboardView,
  CollabView,
  CouncilDashboardView,
  LearnedPatternView,
  LearnedTrajectoryView,
} from "../../lib/types";

interface HermesLiveData {
  cache: CacheEconomyView | null;
  discord: BotLiveStatusView | null;
  goal: GoalProgressView | null;
  memory: MemoryDashboardView | null;
  cost: CostSummaryView | null;
  tokens: SessionTokensView | null;
  compress: CompressStatsView | null;
  schedule: ScheduleDashboardView | null;
  collab: CollabView | null;
  council: CouncilDashboardView | null;
  learnPatterns: LearnedPatternView[];
  learnTrajectories: LearnedTrajectoryView[];
}

interface HermesDashboardPayload {
  cache: CacheEconomyView | null;
  memory: MemoryDashboardView | null;
  bot: BotLiveStatusView | null;
  goal: GoalProgressView | null;
  cost: CostSummaryView | null;
  tokens: SessionTokensView | null;
  compress: CompressStatsView | null;
  schedule: ScheduleDashboardView | null;
  collab: CollabView | null;
  council: CouncilDashboardView | null;
  learnPatterns: LearnedPatternView[];
  learnTrajectories: LearnedTrajectoryView[];
}

const POLL_MS = 5000;
const EVENT_CHANNEL = "hermes:dashboard";

export function useHermesLiveData(tabId: string | undefined, enabled: boolean): HermesLiveData {
  const [data, setData] = useState<HermesLiveData>({
    cache: null, discord: null, goal: null, memory: null, cost: null, tokens: null, compress: null, schedule: null, collab: null, council: null, learnPatterns: [], learnTrajectories: [],
  });

  // Prefer Wails push events; fall back to polling.
  useEffect(() => {
    if (!enabled) return;

    // Try push-based first.
    try {
      const w = window as any;
      if (w.runtime?.EventsOn) {
        const unsub = w.runtime.EventsOn(EVENT_CHANNEL, (payload: HermesDashboardPayload) => {
          if (!payload) return;
          setData({
            cache: payload.cache ?? null,
            discord: payload.bot ?? null,
            goal: payload.goal ?? null,
            memory: payload.memory ?? null,
            cost: (payload as any).cost ?? null,
            tokens: (payload as any).tokens ?? null,
            compress: (payload as any).compress ?? null,
            schedule: (payload as any).schedule ?? null,
            collab: (payload as any).collab ?? null,
            council: (payload as any).council ?? null,
            learnPatterns: (payload as any).learnPatterns ?? [],
            learnTrajectories: (payload as any).learnTrajectories ?? [],
          });
        });
        // Initial fetch in case the first event hasn't fired yet.
        pollOnce();
        return () => { try { unsub(); } catch { /* ignore */ } };
      }
    } catch {
      // Events not available — fall through to polling.
    }

    // Polling fallback.
    pollOnce();
    const id = setInterval(pollOnce, POLL_MS);
    return () => clearInterval(id);
  }, [enabled, tabId]);

  const pollOnce = useCallback(async () => {
    if (!enabled) return;
    try {
      const tid = tabId ?? "";
      const [cache, discord, goal, memory, cost, tokens, compress, schedule, collab, council, learnPatterns, learnTrajectories] = await Promise.all([
        tid ? app.CacheEconomyForTab(tid) : app.CacheEconomy(),
        app.BotLiveStatus(),
        tid ? app.GoalProgressForTab(tid) : app.GoalProgress(),
        app.MemoryDashboard(),
        tid ? app.CostSummaryForTab(tid) : app.CostSummary(),
        tid ? app.SessionTokensForTab(tid) : app.SessionTokens(),
        tid ? app.CompressStatsForTab(tid) : app.CompressStats(),
        app.ScheduleDashboard(),
        app.CollabDashboard(),
        app.CouncilDashboard(),
        app.LearnedPatterns().then(([p, _t]) => p ?? []),
        app.LearnedPatterns().then(([_p, t]) => t ?? []),
      ]);
      setData({ cache, discord, goal, memory, cost, tokens, compress, schedule, collab, council, learnPatterns, learnTrajectories });
    } catch {
      // silent — bridge may not be ready
    }
  }, [enabled, tabId]);

  return data;
}
