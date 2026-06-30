import { onBeforeUnmount } from "vue";

type PollingOptions = {
  intervalMs?: number;
  maxAttempts?: number;
};

type StartPollingOptions = {
  immediate?: boolean;
};

export function useResearchRepairTaskPolling(loadTasks: () => Promise<void>, options: PollingOptions = {}) {
  const intervalMs = options.intervalMs ?? 4_000;
  const maxAttempts = options.maxAttempts ?? 6;
  let timer: number | null = null;
  let generation = 0;
  let attempts = 0;

  function clearTimer() {
    if (timer === null) return;
    window.clearTimeout(timer);
    timer = null;
  }

  function stopRepairTaskPolling() {
    generation += 1;
    attempts = 0;
    clearTimer();
  }

  function startRepairTaskPolling(startOptions: StartPollingOptions = {}) {
    generation += 1;
    attempts = 0;
    clearTimer();
    if (maxAttempts <= 0) return;

    const currentGeneration = generation;
    if (startOptions.immediate === false) {
      scheduleNext(currentGeneration);
      return;
    }
    void runAttempt(currentGeneration);
  }

  function scheduleNext(currentGeneration: number) {
    clearTimer();
    timer = window.setTimeout(() => {
      timer = null;
      void runAttempt(currentGeneration);
    }, intervalMs);
  }

  async function runAttempt(currentGeneration: number) {
    if (currentGeneration !== generation) return;
    attempts += 1;
    try {
      await loadTasks();
    } finally {
      if (currentGeneration !== generation) return;
      if (attempts < maxAttempts) scheduleNext(currentGeneration);
    }
  }

  onBeforeUnmount(stopRepairTaskPolling);

  return {
    startRepairTaskPolling,
    stopRepairTaskPolling,
  };
}
