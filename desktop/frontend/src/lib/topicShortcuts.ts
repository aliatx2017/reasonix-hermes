// useTopicShortcuts - Cmd/Ctrl hold detection plus 1-9 sidebar topic navigation.
//
// When the user holds Cmd (macOS) or Ctrl (Windows/Linux) for a brief moment
// without pressing another key, shortcut badges (⌘1 ... ⌘9) appear over the
// sidebar topic list. Releasing the modifier hides them immediately. Pressing
// Cmd/Ctrl+1-9 navigates to the matching topic.

import { useCallback, useEffect, useRef, useState } from "react";

/** Delay (ms) before showing badges after modifier is held. */
const SHOW_DELAY_MS = 250;

type TopicShortcutEntry = {
  scope: "global" | "project";
  workspaceRoot: string;
  topicId: string;
  sessionPath?: string;
};

type TopicShortcutKeyboardEvent = Pick<globalThis.KeyboardEvent, "key" | "ctrlKey" | "metaKey" | "defaultPrevented">;

export function topicShortcutIndexFromEvent(event: TopicShortcutKeyboardEvent): number | null {
  if (event.defaultPrevented) return null;
  if (!event.metaKey && !event.ctrlKey) return null;
  if (!/^[1-9]$/.test(event.key)) return null;
  return Number(event.key) - 1;
}

export function useTopicShortcuts(
  enabled = true,
) {
  const [showBadges, setShowBadges] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const heldRef = useRef(false);

  const clearTimer = useCallback(() => {
    if (timerRef.current !== null) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  const hideBadges = useCallback(() => {
    clearTimer();
    heldRef.current = false;
    setShowBadges(false);
  }, [clearTimer]);

  useEffect(() => {
    if (!enabled) return;

    const isModifier = (key: string) => key === "Meta" || key === "Control";

    const onKeydown = (event: globalThis.KeyboardEvent) => {
      if (!isModifier(event.key)) return;
      if (heldRef.current) return; // already tracking
      heldRef.current = true;
      clearTimer();
      timerRef.current = setTimeout(() => {
        timerRef.current = null;
        setShowBadges(true);
      }, SHOW_DELAY_MS);
    };

    const onKeyup = (event: globalThis.KeyboardEvent) => {
      if (!isModifier(event.key)) return;
      hideBadges();
    };

    // If the window loses focus, hide badges
    const onBlur = () => hideBadges();

    document.addEventListener("keydown", onKeydown);
    document.addEventListener("keyup", onKeyup);
    window.addEventListener("blur", onBlur);
    return () => {
      document.removeEventListener("keydown", onKeydown);
      document.removeEventListener("keyup", onKeyup);
      window.removeEventListener("blur", onBlur);
      clearTimer();
    };
  }, [enabled, clearTimer, hideBadges]);

  return { showBadges, hideBadges };
}

export type { TopicShortcutEntry };
