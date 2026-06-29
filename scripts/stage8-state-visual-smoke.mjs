#!/usr/bin/env node
import { spawn } from "node:child_process";
import fs from "node:fs";
import net from "node:net";
import os from "node:os";
import path from "node:path";

loadDotEnv();

const baseUrl = process.env.BASE_URL ?? `http://127.0.0.1:${process.env.HTTP_PORT ?? "8080"}`;
const username = process.env.SMOKE_USERNAME ?? process.env.BOOTSTRAP_OPERATOR_USERNAME ?? "admin";
const password = process.env.SMOKE_PASSWORD ?? process.env.BOOTSTRAP_OPERATOR_PASSWORD ?? "tictick-local-admin-password";
const settleMs = parsePositiveInt(process.env.SMOKE_SETTLE_MS, 600);
const widthTolerance = parsePositiveInt(process.env.SMOKE_WIDTH_TOLERANCE, 2);

const viewports = [
  { label: "desktop-1440x900", metrics: { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false } },
  { label: "mobile-390x844", metrics: { width: 390, height: 844, deviceScaleFactor: 2, mobile: true } },
];

const themes = ["light", "dark"];
const locales = ["zh-CN", "en-US"];

const stateCases = [
  {
    label: "research-empty",
    path: "/research",
    selector: ".research-workspace",
    mocks: "empty",
    states: [".research-tasks-panel .state-block", ".research-chart-viewport .state-block"],
    chartViewport: ".research-chart-viewport",
  },
  {
    label: "research-tasks-error",
    path: "/research",
    selector: ".research-workspace",
    mocks: "researchTasksError",
    states: [".research-tasks-panel .state-block"],
  },
  {
    label: "research-chart-error",
    path: "/research",
    selector: ".research-workspace",
    mocks: "researchChartError",
    states: [".research-chart-viewport .state-block"],
    chartViewport: ".research-chart-viewport",
  },
  {
    label: "backtests-empty",
    path: "/backtests",
    selector: ".backtests-panel",
    mocks: "empty",
    states: [".backtests-panel .state-block"],
  },
  {
    label: "backtests-error",
    path: "/backtests",
    selector: ".backtests-panel",
    mocks: "backtestsError",
    states: [".backtests-panel .state-block"],
  },
  {
    label: "trading-empty",
    path: "/trading",
    selector: ".trading-panel",
    mocks: "empty",
    states: [".trading-panel .state-block"],
  },
  {
    label: "trading-error",
    path: "/trading",
    selector: ".trading-panel",
    mocks: "tradingError",
    states: [".trading-panel .state-block"],
  },
  {
    label: "notifications-empty",
    path: "/system/notifications",
    selector: ".system-panel",
    mocks: "empty",
    states: [".system-panel .state-block"],
    minStateBlocks: 2,
  },
  {
    label: "notifications-error",
    path: "/system/notifications",
    selector: ".system-panel",
    mocks: "notificationsError",
    states: [".system-panel .state-block"],
  },
  {
    label: "system-empty",
    path: "/system/exchange-accounts",
    selector: ".system-panel",
    mocks: "empty",
    states: [".system-panel .state-block"],
  },
  {
    label: "backtest-detail-empty",
    path: "/backtests/visual-state-backtest",
    selector: ".backtest-detail-workspace",
    mocks: "empty",
    states: [".backtest-chart-viewport .state-block"],
    chartViewport: ".backtest-chart-viewport",
    detailLayout: {
      chartPanel: ".backtest-chart-panel",
      lowerGrid: ".backtest-detail-lower-grid",
      summary: ".backtest-summary-panel",
      tabs: ".backtest-detail-tabs",
    },
  },
  {
    label: "backtest-detail-chart-error",
    path: "/backtests/visual-state-backtest",
    selector: ".backtest-detail-workspace",
    mocks: "detailChartError",
    states: [".backtest-chart-viewport .state-block"],
    chartViewport: ".backtest-chart-viewport",
    detailLayout: {
      chartPanel: ".backtest-chart-panel",
      lowerGrid: ".backtest-detail-lower-grid",
      summary: ".backtest-summary-panel",
      tabs: ".backtest-detail-tabs",
    },
  },
  {
    label: "trading-detail-empty",
    path: "/trading/visual-state-trading",
    selector: ".trading-detail-workspace",
    mocks: "empty",
    states: [".trading-detail-chart-viewport .state-block", ".trading-detail-tabs .state-block"],
    chartViewport: ".trading-detail-chart-viewport",
    detailLayout: {
      chartPanel: ".trading-detail-chart",
      lowerGrid: ".trading-detail-lower-grid",
      summary: ".trading-detail-summary",
      tabs: ".trading-detail-tabs",
    },
  },
  {
    label: "trading-detail-chart-error",
    path: "/trading/visual-state-trading",
    selector: ".trading-detail-workspace",
    mocks: "detailChartError",
    states: [".trading-detail-chart-viewport .state-block"],
    chartViewport: ".trading-detail-chart-viewport",
    detailLayout: {
      chartPanel: ".trading-detail-chart",
      lowerGrid: ".trading-detail-lower-grid",
      summary: ".trading-detail-summary",
      tabs: ".trading-detail-tabs",
    },
  },
];

let chrome = null;
let chromeProfileDir = null;

process.once("SIGINT", () => shutdownFromSignal("SIGINT", 130));
process.once("SIGTERM", () => shutdownFromSignal("SIGTERM", 143));

try {
  const endpoint = process.env.CDP_ENDPOINT ?? (await launchChrome());
  const results = [];
  for (const viewport of viewports) {
    for (const theme of themes) {
      for (const locale of locales) {
        results.push(await runStatePass(endpoint, viewport, theme, locale));
      }
    }
  }

  for (const result of results) {
    console.log(
      `${result.viewport}/${result.theme}/${result.locale}: ${result.cases.length} state cases, max document width ${result.maxDocumentWidth}px`,
    );
  }
  console.log("stage8 state visual smoke passed");
} catch (error) {
  console.error("stage8 state visual smoke failed");
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
} finally {
  cleanupChrome();
}

async function runStatePass(endpoint, viewport, theme, locale) {
  const page = await createPage(endpoint, `${baseUrl}/`);
  const cdp = await connect(page.webSocketDebuggerUrl);
  const browserErrors = [];
  const caseResults = [];
  let currentCase = null;

  cdp.on("Runtime.consoleAPICalled", (event) => {
    if (event.type === "error") browserErrors.push(formatConsoleArgs(event.args));
  });
  cdp.on("Runtime.exceptionThrown", (event) => {
    browserErrors.push(event.exceptionDetails?.text ?? "runtime exception");
  });
  cdp.on("Fetch.requestPaused", (event) => {
    void handlePausedRequest(cdp, currentCase, event);
  });

  try {
    await cdp.send("Page.enable");
    await cdp.send("Runtime.enable");
    await cdp.send("Network.enable");
    await cdp.send("Fetch.enable", {
      patterns: [{ urlPattern: `${baseUrl}/api/*`, requestStage: "Request" }],
    });
    await cdp.send("Emulation.setDeviceMetricsOverride", viewport.metrics);
    await cdp.send("Page.navigate", { url: `${baseUrl}/` });
    await waitFor(cdp, "document.readyState === 'complete' || document.readyState === 'interactive'");
    await login(cdp, viewport.label, theme, locale);
    await setLocale(cdp, locale);

    for (const stateCase of stateCases) {
      currentCase = stateCase;
      await setTheme(cdp, theme);
      await setLocale(cdp, locale);
      await cdp.send("Page.navigate", { url: `${baseUrl}${stateCase.path}` });
      await waitFor(cdp, "!!document.querySelector('.app-shell')", 15000);
      await waitFor(cdp, "!!document.querySelector('.page-title')", 15000);
      await waitFor(cdp, `!!document.querySelector(${JSON.stringify(stateCase.selector)})`, 15000);
      for (const selector of stateCase.states) {
        await waitFor(cdp, `!!document.querySelector(${JSON.stringify(selector)})`, 15000);
      }
      await delay(settleMs);

      const sample = await evaluate(cdp, stateSampleExpression(stateCase));
      assertStateLayout(viewport, theme, locale, stateCase, sample);
      caseResults.push({ label: stateCase.label, sample });
    }

    if (browserErrors.length > 0) {
      throw new Error(`${viewport.label}/${theme}/${locale} browser errors: ${browserErrors.join(" | ")}`);
    }

    return {
      viewport: viewport.label,
      theme,
      locale,
      cases: caseResults,
      maxDocumentWidth: Math.max(...caseResults.map((result) => result.sample.documentWidth)),
    };
  } finally {
    cdp.close();
  }
}

async function handlePausedRequest(cdp, stateCase, event) {
  const response = stateCase ? mockResponseFor(stateCase.mocks, event.request) : null;
  try {
    if (!response) {
      await cdp.send("Fetch.continueRequest", { requestId: event.requestId });
      return;
    }
    await cdp.send("Fetch.fulfillRequest", {
      requestId: event.requestId,
      responseCode: response.status,
      responseHeaders: [
        { name: "content-type", value: "application/json; charset=utf-8" },
        { name: "cache-control", value: "no-store" },
      ],
      body: Buffer.from(JSON.stringify(response.body)).toString("base64"),
    });
  } catch {
    // The page may have navigated while a mocked request was in flight.
  }
}

function mockResponseFor(profile, request) {
  if (request.method !== "GET") return null;
  const url = new URL(request.url);
  const apiPath = url.pathname.replace(/^\/api/, "");

  if (profile === "researchTasksError" && apiPath === "/data/tasks") return forcedError();
  if (profile === "researchChartError" && apiPath === "/candles") return forcedError();
  if (profile === "backtestsError" && apiPath === "/backtests") return forcedError();
  if (profile === "tradingError" && apiPath === "/trading/tasks") return forcedError();
  if (profile === "notificationsError" && apiPath === "/system/notifications") return forcedError();
  if (profile === "detailChartError" && apiPath === "/candles") return forcedError();

  if (apiPath === "/data/tasks") return ok([]);
  if (apiPath === "/candles") return ok(emptyCandleResult(url.searchParams.get("interval") ?? "1m"));
  if (apiPath === "/market/candle-gaps") return ok(emptyMarketGapScan());
  if (apiPath === "/market/instruments/status") return ok([]);
  if (apiPath === "/backtests") return ok([]);
  if (apiPath === "/backtests/visual-state-backtest") return ok(mockBacktestTask());
  if (apiPath === "/backtests/visual-state-backtest/orders") return ok([]);
  if (apiPath === "/backtests/visual-state-backtest/intents") return ok([]);
  if (apiPath === "/trading/tasks") return ok([]);
  if (apiPath === "/trading/tasks/visual-state-trading") return ok(mockTradingTask());
  if (/^\/trading\/tasks\/visual-state-trading\/(?:positions|intents|orders|executions|notifications)$/.test(apiPath)) {
    return ok([]);
  }
  if (apiPath === "/system/notifications") return ok([]);
  if (apiPath === "/system/notifications/channels") return ok([]);
  if (apiPath === "/system/exchange-accounts") return ok([]);
  if (apiPath === "/system/operators") return ok([]);
  if (apiPath === "/auth/sessions") return ok([]);
  if (apiPath === "/system/audit-events") return ok([]);

  return null;
}

function ok(body) {
  return { status: 200, body };
}

function forcedError() {
  return {
    status: 503,
    body: {
      code: "visual_state_forced_error",
      message: "Stage 8 visual state forced error",
    },
  };
}

function emptyCandleResult(interval) {
  return {
    candles: [],
    source: "none",
    requestedInterval: interval,
    baseInterval: interval,
    health: "insufficient",
    gaps: [],
    issues: [],
    coverage: {
      requestedLimit: 1000,
      returnedCandles: 0,
      limitedByBaseWindow: false,
    },
    window: {
      count: 0,
    },
    pagination: {
      hasPrevious: false,
      hasNext: false,
    },
  };
}

function emptyMarketGapScan() {
  return {
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    gaps: [],
    totalCount: 0,
    returnedCount: 0,
    limited: false,
    window: {
      count: 0,
    },
  };
}

function mockBacktestTask() {
  return {
    id: "visual-state-backtest",
    name: "visual-state-backtest",
    exchange: "binance",
    symbol: "BTCUSDT",
    interval: "1m",
    startTime: "2026-01-01T00:00:00Z",
    endTime: "2026-01-01T02:00:00Z",
    strategyId: "ema-cross",
    strategyParams: { fastPeriod: 2, slowPeriod: 5, orderSize: 0.1, signalMode: "order" },
    initialBalance: "10000",
    feeBps: "1",
    slippageBps: "1",
    triggerMode: "closed_candle",
    resultSummary: {},
    status: "succeeded",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}

function mockTradingTask() {
  return {
    id: "visual-state-trading",
    name: "visual-state-trading",
    type: "paper",
    exchange: "binance",
    accountId: "paper",
    symbol: "BTCUSDT",
    interval: "1m",
    strategyId: "ema-cross",
    strategyParams: { fastPeriod: 2, slowPeriod: 5, orderSize: 0.1, signalMode: "order" },
    intentPolicy: { orderIntent: "execute", notificationChannel: "default" },
    status: "paused",
    lockedBy: "",
    heartbeatAt: "",
    lastError: "",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}

async function login(cdp, viewportLabel, theme, locale) {
  const loginResult = await evaluate(
    cdp,
    `fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ username: ${JSON.stringify(username)}, password: ${JSON.stringify(password)} }),
      credentials: 'include'
    }).then(async (response) => ({ ok: response.ok, status: response.status, body: await response.text() }))`,
  );
  if (!loginResult.ok) {
    throw new Error(`${viewportLabel}/${theme}/${locale} login failed: HTTP ${loginResult.status} ${loginResult.body}`);
  }
}

async function setTheme(cdp, theme) {
  await evaluate(
    cdp,
    `(() => {
      localStorage.setItem('tictick-hi.theme', ${JSON.stringify(theme)});
      document.documentElement.dataset.theme = ${JSON.stringify(theme)};
      return true;
    })()`,
  );
}

async function setLocale(cdp, locale) {
  await evaluate(
    cdp,
    `(() => {
      localStorage.setItem('tictick-hi.locale', ${JSON.stringify(locale)});
      document.documentElement.lang = ${JSON.stringify(locale)};
      return true;
    })()`,
  );
}

function stateSampleExpression(stateCase) {
  return `(() => {
    const root = document.documentElement;
    const body = document.body;
    const visibleText = i18nVisibleText();
    const states = ${JSON.stringify(stateCase.states)}.map((selector) => read(selector));
    return {
      href: location.href,
      theme: document.documentElement.dataset.theme || '',
      locale: document.documentElement.lang || '',
      navText: document.querySelector('.top-nav')?.innerText ?? '',
      i18nLeaks: Array.from(new Set(
        visibleText.match(/\\b(?:auth|backtests|common|nav|overview|page|research|strategy|system|trading)\\.[A-Za-z0-9_.-]+\\b/g) ?? []
      )).slice(0, 8),
      viewportWidth: innerWidth,
      viewportHeight: innerHeight,
      documentWidth: Math.max(root.scrollWidth, body.scrollWidth, root.clientWidth, body.clientWidth),
      documentHeight: Math.max(root.scrollHeight, body.scrollHeight, root.clientHeight, body.clientHeight),
      shell: read('.app-shell'),
      header: read('.app-header'),
      main: read('.app-main'),
      page: read('.page'),
      title: read('.page-title'),
      required: read(${JSON.stringify(stateCase.selector)}),
      states,
      stateBlockCount: document.querySelectorAll('.state-block').length,
      chartViewport: ${stateCase.chartViewport ? `read(${JSON.stringify(stateCase.chartViewport)})` : "null"},
      chart: read('.trading-chart'),
      detail: ${stateCase.detailLayout ? `{
        chartPanel: read(${JSON.stringify(stateCase.detailLayout.chartPanel)}),
        lowerGrid: read(${JSON.stringify(stateCase.detailLayout.lowerGrid)}),
        summary: read(${JSON.stringify(stateCase.detailLayout.summary)}),
        tabs: read(${JSON.stringify(stateCase.detailLayout.tabs)})
      }` : "null"}
    };

    function read(selector) {
      const element = document.querySelector(selector);
      if (!element) return null;
      const rect = element.getBoundingClientRect();
      const style = getComputedStyle(element);
      return {
        selector,
        text: element.innerText || '',
        rectWidth: Math.round(rect.width),
        rectHeight: Math.round(rect.height),
        left: Math.round(rect.left),
        right: Math.round(rect.right),
        top: Math.round(rect.top),
        bottom: Math.round(rect.bottom),
        display: style.display,
        visibility: style.visibility
      };
    }

    function i18nVisibleText() {
      const ignoredSelectors = '.audit-code, .session-id';
      const textNodes = [];
      const walker = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
      while (walker.nextNode()) {
        const node = walker.currentNode;
        const parent = node.parentElement;
        if (!parent || parent.closest(ignoredSelectors)) continue;
        const text = node.nodeValue?.trim();
        if (text) textNodes.push(text);
      }
      return textNodes.join('\\n');
    }
  })()`;
}

function assertStateLayout(viewport, theme, locale, stateCase, sample) {
  const label = `${viewport.label}/${theme}/${locale}/${stateCase.label}`;
  if (sample.theme !== theme) throw new Error(`${label} theme = ${sample.theme}, want ${theme}`);
  assertLocale(label, locale, sample);
  for (const [name, node] of [
    ["shell", sample.shell],
    ["header", sample.header],
    ["main", sample.main],
    ["page", sample.page],
    ["title", sample.title],
    ["required", sample.required],
  ]) {
    assertVisibleNode(label, name, node);
  }
  if (sample.documentWidth > viewport.metrics.width + widthTolerance) {
    throw new Error(
      `${label} document overflowed horizontally: ${JSON.stringify({
        documentWidth: sample.documentWidth,
        viewportWidth: viewport.metrics.width,
      })}`,
    );
  }
  for (const [name, node] of [
    ["shell", sample.shell],
    ["main", sample.main],
    ["page", sample.page],
    ["required", sample.required],
  ]) {
    if (node.right > viewport.metrics.width + widthTolerance) {
      throw new Error(`${label} ${name} escaped viewport: ${JSON.stringify(node)}`);
    }
  }
  if (sample.stateBlockCount < (stateCase.minStateBlocks ?? stateCase.states.length)) {
    throw new Error(`${label} rendered too few state blocks: ${JSON.stringify(sample)}`);
  }
  for (const [index, node] of sample.states.entries()) {
    assertVisibleNode(label, `state[${index}]`, node);
    if (!node.text.trim()) throw new Error(`${label} state[${index}] rendered no readable text: ${JSON.stringify(node)}`);
    if (node.right > viewport.metrics.width + widthTolerance) {
      throw new Error(`${label} state[${index}] escaped viewport: ${JSON.stringify(node)}`);
    }
    if (sample.chartViewport && node.selector.includes("chart-viewport")) {
      assertVisibleNode(label, "chartViewport", sample.chartViewport);
      if (node.top < sample.chartViewport.top - widthTolerance || node.bottom > sample.chartViewport.bottom + widthTolerance) {
        throw new Error(`${label} chart state escaped chart viewport: ${JSON.stringify({ state: node, chartViewport: sample.chartViewport })}`);
      }
    }
  }
  if (sample.chartViewport) {
    assertVisibleNode(label, "chartViewport", sample.chartViewport);
    if (sample.chartViewport.rectHeight < (viewport.metrics.width <= 760 ? 500 : 600) - widthTolerance) {
      throw new Error(`${label} chart state viewport is too short: ${JSON.stringify(sample.chartViewport)}`);
    }
  }
  if (sample.detail) assertDetailStateLayout(label, sample, viewport.metrics);
}

function assertDetailStateLayout(label, sample, viewport) {
  for (const [name, node] of [
    ["detailChartPanel", sample.detail.chartPanel],
    ["detailLowerGrid", sample.detail.lowerGrid],
    ["detailSummary", sample.detail.summary],
    ["detailTabs", sample.detail.tabs],
  ]) {
    assertVisibleNode(label, name, node);
    if (node.right > viewport.width + widthTolerance) throw new Error(`${label} ${name} escaped viewport: ${JSON.stringify(node)}`);
  }
  if (sample.detail.lowerGrid.top <= sample.detail.chartPanel.bottom) {
    throw new Error(`${label} detail lower grid must stay below chart: ${JSON.stringify(sample.detail)}`);
  }
  if (viewport.width > 980 && sample.detail.summary.rectWidth >= sample.detail.tabs.rectWidth) {
    throw new Error(`${label} detail summary must remain narrower than tab list: ${JSON.stringify(sample.detail)}`);
  }
}

function assertLocale(label, locale, sample) {
  if (sample.locale !== locale) throw new Error(`${label} html lang = ${sample.locale}, want ${locale}`);
  const expectedNavText = locale === "en-US" ? "Overview" : "概览";
  const unexpectedNavText = locale === "en-US" ? "概览" : "Overview";
  if (!sample.navText.includes(expectedNavText)) {
    throw new Error(`${label} top nav did not render expected locale: ${JSON.stringify(sample.navText)}`);
  }
  if (sample.navText.includes(unexpectedNavText)) {
    throw new Error(`${label} top nav still contains wrong locale text: ${JSON.stringify(sample.navText)}`);
  }
  if (sample.i18nLeaks.length > 0) {
    throw new Error(`${label} leaked i18n keys into visible text: ${sample.i18nLeaks.join(", ")}`);
  }
}

function assertVisibleNode(label, name, node) {
  if (!node) throw new Error(`${label} missing ${name}`);
  if (node.display === "none" || node.visibility === "hidden") {
    throw new Error(`${label} ${name} is hidden: ${JSON.stringify(node)}`);
  }
  if (node.rectWidth <= 0 || node.rectHeight <= 0) {
    throw new Error(`${label} ${name} has empty bounds: ${JSON.stringify(node)}`);
  }
}

async function launchChrome() {
  const chromePath = findChromePath();
  const port = await findOpenPort(parsePositiveInt(process.env.CHROME_REMOTE_DEBUGGING_PORT, 9235));
  chromeProfileDir = fs.mkdtempSync(path.join(os.tmpdir(), "tictick-hi-state-visual-smoke-"));
  chrome = spawn(
    chromePath,
    [
      "--headless=new",
      `--remote-debugging-port=${port}`,
      `--user-data-dir=${chromeProfileDir}`,
      "--disable-background-networking",
      "--disable-default-apps",
      "--disable-gpu",
      "--disable-sync",
      "--no-first-run",
      "--no-default-browser-check",
      "about:blank",
    ],
    { stdio: "ignore" },
  );

  const endpoint = `http://127.0.0.1:${port}`;
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${endpoint}/json/version`);
      if (response.ok) return endpoint;
    } catch {
      // Chrome is still starting.
    }
    await delay(150);
  }
  throw new Error(`Chrome DevTools endpoint did not start on ${endpoint}`);
}

function findChromePath() {
  const candidates = [
    process.env.CHROME_PATH,
    "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    "/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
    "/Applications/Chromium.app/Contents/MacOS/Chromium",
    "/usr/bin/google-chrome",
    "/usr/bin/chromium",
    "/usr/bin/chromium-browser",
  ].filter(Boolean);
  for (const candidate of candidates) {
    if (fs.existsSync(candidate)) return candidate;
  }
  throw new Error("Chrome executable not found. Set CHROME_PATH to run the stage8 state visual smoke.");
}

async function findOpenPort(startPort) {
  for (let port = startPort; port < startPort + 50; port += 1) {
    if (await canListen(port)) return port;
  }
  throw new Error(`no open DevTools port found from ${startPort}`);
}

function canListen(port) {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once("error", () => resolve(false));
    server.once("listening", () => {
      server.close(() => resolve(true));
    });
    server.listen(port, "127.0.0.1");
  });
}

async function createPage(endpoint, url) {
  let response = await fetch(`${endpoint}/json/new?${encodeURIComponent(url)}`, { method: "PUT" });
  if (!response.ok) response = await fetch(`${endpoint}/json/new?${encodeURIComponent(url)}`);
  if (!response.ok) throw new Error(`failed to create Chrome target: HTTP ${response.status}`);
  return response.json();
}

function connect(wsUrl) {
  const ws = new WebSocket(wsUrl);
  let nextId = 0;
  const pending = new Map();
  const handlers = new Map();

  ws.addEventListener("message", (event) => {
    const message = JSON.parse(event.data);
    if (message.id && pending.has(message.id)) {
      const { resolve, reject } = pending.get(message.id);
      pending.delete(message.id);
      if (message.error) reject(new Error(JSON.stringify(message.error)));
      else resolve(message.result);
      return;
    }
    if (message.method && handlers.has(message.method)) {
      for (const handler of handlers.get(message.method)) handler(message.params ?? {});
    }
  });

  return new Promise((resolve, reject) => {
    ws.addEventListener(
      "open",
      () => {
        resolve({
          send(method, params = {}) {
            const id = ++nextId;
            ws.send(JSON.stringify({ id, method, params }));
            return new Promise((resolve, reject) => pending.set(id, { resolve, reject }));
          },
          on(method, handler) {
            const current = handlers.get(method) ?? [];
            current.push(handler);
            handlers.set(method, current);
          },
          close() {
            ws.close();
          },
        });
      },
      { once: true },
    );
    ws.addEventListener("error", reject, { once: true });
  });
}

async function waitFor(cdp, expression, timeoutMs = 10000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    if (await evaluate(cdp, expression)) return;
    await delay(200);
  }
  throw new Error(`timeout waiting for: ${expression}`);
}

async function evaluate(cdp, expression) {
  const result = await cdp.send("Runtime.evaluate", {
    expression,
    awaitPromise: true,
    returnByValue: true,
    userGesture: true,
  });
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.text || JSON.stringify(result.exceptionDetails));
  return result.result.value;
}

function formatConsoleArgs(args) {
  return args
    .map((arg) => arg.value ?? arg.description ?? arg.type ?? "")
    .filter(Boolean)
    .join(" ");
}

function loadDotEnv() {
  const envPath = path.resolve(".env");
  if (!fs.existsSync(envPath)) return;
  const lines = fs.readFileSync(envPath, "utf8").split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) continue;
    const match = /^([A-Za-z_][A-Za-z0-9_]*)=(.*)$/.exec(trimmed);
    if (!match || process.env[match[1]] !== undefined) continue;
    process.env[match[1]] = match[2].replace(/^['"]|['"]$/g, "");
  }
}

function parsePositiveInt(value, fallback) {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function cleanupChrome() {
  if (chrome) {
    chrome.kill("SIGTERM");
    chrome = null;
  }
  if (chromeProfileDir) {
    fs.rmSync(chromeProfileDir, { recursive: true, force: true });
    chromeProfileDir = null;
  }
}

function shutdownFromSignal(signal, code) {
  console.error(`stage8 state visual smoke interrupted by ${signal}`);
  cleanupChrome();
  process.exit(code);
}
