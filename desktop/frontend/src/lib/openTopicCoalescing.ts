export type PendingOpenTopicRequest = {
  seq: number;
  scope: string;
  workspaceRoot: string;
  topicId: string;
  sessionPath?: string;
  resolve: () => void;
};

export type OpenTopicRequestInput = Omit<PendingOpenTopicRequest, "seq" | "resolve">;

export type OpenTopicCoalescingRefs = {
  seqRef: { current: number };
  runningRef: { current: boolean };
  pendingRef: { current: PendingOpenTopicRequest | null };
};

export function enqueueOpenTopicRequest(
  refs: OpenTopicCoalescingRefs,
  input: OpenTopicRequestInput,
  run: (request: PendingOpenTopicRequest) => Promise<void>,
): Promise<void> {
  const seq = refs.seqRef.current + 1;
  refs.seqRef.current = seq;

  let resolve!: () => void;
  const promise = new Promise<void>((res) => {
    resolve = res;
  });
  const request: PendingOpenTopicRequest = { ...input, seq, resolve };

  const start = (next: PendingOpenTopicRequest) => {
    refs.runningRef.current = true;
    void (async () => {
      try {
        await run(next);
      } catch {
        // run() is expected to handle user-visible failures. The scheduler keeps
        // navigation promises non-rejecting so event handlers cannot leak
        // unhandled rejections.
      } finally {
        next.resolve();
        const pending = refs.pendingRef.current;
        if (pending) {
          refs.pendingRef.current = null;
          start(pending);
        } else {
          refs.runningRef.current = false;
        }
      }
    })();
  };

  if (refs.runningRef.current) {
    refs.pendingRef.current?.resolve();
    refs.pendingRef.current = request;
    return promise;
  }

  start(request);
  return promise;
}
