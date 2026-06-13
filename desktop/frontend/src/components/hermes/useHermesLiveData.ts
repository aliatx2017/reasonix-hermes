import { useState, useEffect, useCallback } from "react";
import { app } from "../../lib/bridge";
import type { CacheEconomyView, BotLiveStatusView, GoalProgressView, MemoryDashboardView } from "../../lib/types";

interface HermesLiveData {
  cache: CacheEconomyView | null;
  discord: BotLiveStatusView | null;
  goal: GoalProgressView | null;
  memory: MemoryDashboardView | null;
}

const POLL_MS = 5000;

export function useHermesLiveData(tabId: string | undefined, enabled: boolean): HermesLiveData {
  const [data, setData] = useState<HermesLiveData>({
    cache: null, discord: null, goal: null, memory: null,
  });

  const poll = useCallback(async () => {
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

  useEffect(() => {
    poll();
    if (!enabled) return;
    const id = setInterval(poll, POLL_MS);
    return () => clearInterval(id);
  }, [poll, enabled]);

  return data;
}
