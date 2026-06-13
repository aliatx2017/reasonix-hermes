import { useState, useEffect, useCallback } from "react";
import { app } from "../../lib/bridge";
import type { CacheEconomyView, BotLiveStatusView, GoalProgressView, MemoryDashboardView } from "../../lib/types";

interface HermesLiveData {
  cache: CacheEconomyView | null;
  discord: BotLiveStatusView | null;
  goal: GoalProgressView | null;
  memory: MemoryDashboardView | null;
}

interface HermesDashboardPayload {
  cache: CacheEconomyView | null;
  memory: MemoryDashboardView | null;
  bot: BotLiveStatusView | null;
  goal: GoalProgressView | null;
}

const POLL_MS = 5000;
const EVENT_CHANNEL = "hermes:dashboard";

export function useHermesLiveData(tabId: string | undefined, enabled: boolean): HermesLiveData {
  const [data, setData] = useState<HermesLiveData>({
    cache: null, discord: null, goal: null, memory: null,
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
      const [cache, discord, goal, memory] = await Promise.all([
        tid ? app.CacheEconomyForTab(tid) : app.CacheEconomy(),
        app.BotLiveStatus(),
        tid ? app.GoalProgressForTab(tid) : app.GoalProgress(),
        app.MemoryDashboard(),
      ]);
      setData({ cache, discord, goal, memory });
    } catch {
      // silent — bridge may not be ready
    }
  }, [enabled, tabId]);

  return data;
}
