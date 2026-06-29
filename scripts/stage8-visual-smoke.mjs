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
const settleMs = parsePositiveInt(process.env.SMOKE_SETTLE_MS, 1200);
const widthTolerance = parsePositiveInt(process.env.SMOKE_WIDTH_TOLERANCE, 2);
const maxToolbarSymbolWidth = parsePositiveInt(process.env.SMOKE_MAX_SYMBOL_WIDTH, 220);
const maxRightPriceAxisWidth = parsePositiveInt(process.env.SMOKE_MAX_RIGHT_PRICE_AXIS_WIDTH, 60);

const viewports = [
  { label: "desktop-1440x900", metrics: { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false } },
  { label: "narrow-desktop-812x1320", metrics: { width: 812, height: 1320, deviceScaleFactor: 2, mobile: false } },
  { label: "mobile-390x844", metrics: { width: 390, height: 844, deviceScaleFactor: 2, mobile: true } },
];

const themes = ["light", "dark"];
const locales = ["zh-CN", "en-US"];

const pages = [
  { label: "overview", path: "/overview", selector: ".overview-metrics" },
  { label: "research", path: "/research", selector: ".research-workspace", chart: true },
  { label: "backtests", path: "/backtests", selector: ".backtests-panel" },
  { label: "backtests-new", path: "/backtests/new", selector: ".task-form-grid" },
  { label: "trading", path: "/trading", selector: ".trading-panel" },
  { label: "trading-new", path: "/trading/new", selector: ".task-form-grid" },
  { label: "system-notifications", path: "/system/notifications", selector: ".system-panel" },
  { label: "system-exchange-accounts", path: "/system/exchange-accounts", selector: ".system-panel" },
  { label: "system-operators", path: "/system/operators", selector: ".system-panel" },
  { label: "system-sessions", path: "/system/sessions", selector: ".system-panel" },
  { label: "system-audit-events", path: "/system/audit-events", selector: ".system-panel" },
  { label: "system-health", path: "/system/health", selector: ".health-service-list, .health-service" },
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
        results.push(await runVisualPass(endpoint, viewport, theme, locale));
      }
    }
  }

  for (const result of results) {
    console.log(
      `${result.viewport}/${result.theme}/${result.locale}: ${result.pages.length} pages, max document width ${result.maxDocumentWidth}px`,
    );
  }
  console.log("stage8 visual smoke passed");
} catch (error) {
  console.error("stage8 visual smoke failed");
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
} finally {
  cleanupChrome();
}

async function runVisualPass(endpoint, viewport, theme, locale) {
  const page = await createPage(endpoint, `${baseUrl}/`);
  const cdp = await connect(page.webSocketDebuggerUrl);
  const browserErrors = [];
  const pageResults = [];

  cdp.on("Runtime.consoleAPICalled", (event) => {
    if (event.type === "error") {
      browserErrors.push(formatConsoleArgs(event.args));
    }
  });
  cdp.on("Runtime.exceptionThrown", (event) => {
    browserErrors.push(event.exceptionDetails?.text ?? "runtime exception");
  });

  try {
    await cdp.send("Page.enable");
    await cdp.send("Runtime.enable");
    await cdp.send("Network.enable");
    await cdp.send("Emulation.setDeviceMetricsOverride", viewport.metrics);
    await cdp.send("Page.navigate", { url: `${baseUrl}/` });
    await waitFor(cdp, "document.readyState === 'complete' || document.readyState === 'interactive'");
    await login(cdp, viewport.label, theme, locale);
    await setLocale(cdp, locale);

    const passPages = [...pages, ...(await detailPages(cdp))];
    for (const pageConfig of passPages) {
      await setTheme(cdp, theme);
      await setLocale(cdp, locale);
      await cdp.send("Page.navigate", { url: `${baseUrl}${pageConfig.path}` });
      await waitFor(cdp, "!!document.querySelector('.app-shell')", 15000);
      await waitFor(cdp, "!!document.querySelector('.page-title')", 15000);
      await waitFor(cdp, `!!document.querySelector(${JSON.stringify(pageConfig.selector)})`, 15000);
      if (pageConfig.chart) {
        await waitFor(cdp, "!!document.querySelector('.research-chart-body')", 15000);
      }
      if (pageConfig.detailLayout) {
        for (const selector of Object.values(pageConfig.detailLayout)) {
          await waitFor(cdp, `!!document.querySelector(${JSON.stringify(selector)})`, 15000);
        }
        await waitFor(cdp, "!!document.querySelector('.trading-chart')", 15000);
        await waitFor(cdp, "!!document.querySelector('.tv-lightweight-charts')", 15000);
      }
      await delay(settleMs);

      const sample = await evaluate(cdp, visualSampleExpression(pageConfig));
      assertPageLayout(viewport, theme, locale, pageConfig, sample);
      pageResults.push({ label: pageConfig.label, sample });
    }

    if (browserErrors.length > 0) {
      throw new Error(`${viewport.label}/${theme}/${locale} browser errors: ${browserErrors.join(" | ")}`);
    }

    return {
      viewport: viewport.label,
      theme,
      locale,
      pages: pageResults,
      maxDocumentWidth: Math.max(...pageResults.map((result) => result.sample.documentWidth)),
    };
  } finally {
    cdp.close();
  }
}

async function detailPages(cdp) {
  const result = await evaluate(
    cdp,
    `Promise.all([
      fetch('/api/backtests', { credentials: 'include' }).then(async (response) => response.ok ? response.json() : []),
      fetch('/api/trading/tasks', { credentials: 'include' }).then(async (response) => response.ok ? response.json() : [])
    ]).then(([backtests, tradingTasks]) => ({
      backtestId: Array.isArray(backtests) && backtests.length > 0 ? backtests[0].id : '',
      tradingTaskId: Array.isArray(tradingTasks) && tradingTasks.length > 0 ? tradingTasks[0].id : ''
    }))`,
  );
  const detailPages = [];
  if (result.backtestId) {
    detailPages.push({
      label: "backtest-detail",
      path: `/backtests/${encodeURIComponent(result.backtestId)}`,
      selector: ".backtest-detail-workspace",
      detailLayout: {
        chartPanel: ".backtest-chart-panel",
        chartViewport: ".backtest-chart-viewport",
        lowerGrid: ".backtest-detail-lower-grid",
        summary: ".backtest-summary-panel",
        tabs: ".backtest-detail-tabs",
      },
    });
  }
  if (result.tradingTaskId) {
    detailPages.push({
      label: "trading-detail",
      path: `/trading/${encodeURIComponent(result.tradingTaskId)}`,
      selector: ".trading-detail-workspace",
      detailLayout: {
        chartPanel: ".trading-detail-chart",
        chartViewport: ".trading-detail-chart-viewport",
        lowerGrid: ".trading-detail-lower-grid",
        summary: ".trading-detail-summary",
        tabs: ".trading-detail-tabs",
      },
    });
  }
  return detailPages;
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

function visualSampleExpression(pageConfig) {
  return `(() => {
    const read = (selector) => {
      const element = document.querySelector(selector);
      if (!element) return null;
      const rect = element.getBoundingClientRect();
      const style = getComputedStyle(element);
      return {
        selector,
        className: element.className,
        clientWidth: element.clientWidth,
        clientHeight: element.clientHeight,
        scrollWidth: element.scrollWidth,
        scrollHeight: element.scrollHeight,
        rectWidth: Math.round(rect.width),
        rectHeight: Math.round(rect.height),
        left: Math.round(rect.left),
        right: Math.round(rect.right),
        top: Math.round(rect.top),
        bottom: Math.round(rect.bottom),
        overflowX: style.overflowX,
        overflowY: style.overflowY,
        display: style.display,
        visibility: style.visibility
      };
    };
    const detailSelectors = ${JSON.stringify(pageConfig.detailLayout ?? null)};
    const root = document.documentElement;
    const body = document.body;
    const visibleText = i18nVisibleText();
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
      required: read(${JSON.stringify(pageConfig.selector)}),
      toolbarControls: read('.research-source-controls'),
      toolbarSymbol: read('.research-symbol-input'),
      chartBody: read('.research-chart-body'),
      chartViewport: read(detailSelectors ? detailSelectors.chartViewport : '.research-chart-viewport'),
      chartInlineStartGutter: cssPixel(detailSelectors ? detailSelectors.chartPanel : '.research-chart-body', 'padding-left'),
      chartInlineEndGutter: cssPixel(detailSelectors ? detailSelectors.chartPanel : '.research-chart-body', 'padding-right'),
      chart: read('.trading-chart'),
      tv: read('.tv-lightweight-charts'),
      priceAxisCanvas: rightPriceAxisCanvas(),
      detail: detailSelectors ? {
        chartPanel: read(detailSelectors.chartPanel),
        chartViewport: read(detailSelectors.chartViewport),
        lowerGrid: read(detailSelectors.lowerGrid),
        summary: read(detailSelectors.summary),
        tabs: read(detailSelectors.tabs)
      } : null
    };

    function rightPriceAxisCanvas() {
      const chartViewport = document.querySelector(detailSelectors ? detailSelectors.chartViewport : '.research-chart-viewport');
      const viewportRect = chartViewport?.getBoundingClientRect();
      const canvases = Array.from(document.querySelectorAll('.trading-chart__canvas canvas'))
        .map((canvas, index) => ({ index, node: canvas, rect: canvas.getBoundingClientRect() }))
        .filter((entry) => entry.rect.width >= 40 && entry.rect.width <= 180)
        .filter((entry) => !viewportRect || entry.rect.height >= Math.max(120, viewportRect.height - 96))
        .sort((left, right) => right.rect.right - left.rect.right);
      const canvas = canvases[0]?.node;
      return canvas ? readCanvas(canvas, canvases[0].index) : null;
    }

    function readCanvas(canvas, index) {
      const rect = canvas.getBoundingClientRect();
      return {
        selector: '.trading-chart__canvas canvas',
        index,
        clientWidth: canvas.clientWidth,
        clientHeight: canvas.clientHeight,
        scrollWidth: canvas.scrollWidth,
        scrollHeight: canvas.scrollHeight,
        rectWidth: Math.round(rect.width),
        rectHeight: Math.round(rect.height),
        left: Math.round(rect.left),
        right: Math.round(rect.right),
        top: Math.round(rect.top),
        bottom: Math.round(rect.bottom),
        display: getComputedStyle(canvas).display,
        visibility: getComputedStyle(canvas).visibility
      };
    }

    function cssPixel(selector, property) {
      const element = document.querySelector(selector);
      if (!element) return 0;
      const value = Number.parseFloat(getComputedStyle(element).getPropertyValue(property));
      return Number.isFinite(value) && value > 0 ? value : 0;
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

function assertPageLayout(viewport, theme, locale, page, sample) {
  const label = `${viewport.label}/${theme}/${locale}/${page.label}`;
  if (sample.theme !== theme) {
    throw new Error(`${label} theme = ${sample.theme}, want ${theme}`);
  }
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
    if (!node) continue;
    if (node.rectWidth > viewport.metrics.width + widthTolerance || node.right > viewport.metrics.width + widthTolerance) {
      throw new Error(`${label} ${name} escaped viewport: ${JSON.stringify(node)}`);
    }
  }
  if (page.chart) assertResearchChartSmoke(label, sample, viewport.metrics);
  if (page.detailLayout) assertDetailLayoutSmoke(label, sample, viewport.metrics);
}

function assertLocale(label, locale, sample) {
  if (sample.locale !== locale) {
    throw new Error(`${label} html lang = ${sample.locale}, want ${locale}`);
  }
  const expectedNavText = locale === "en-US" ? "Overview" : "概览";
  const unexpectedNavText = locale === "en-US" ? "概览" : "Overview";
  if (!sample.navText.includes(expectedNavText)) {
    throw new Error(
      `${label} top nav did not render the expected locale: ${JSON.stringify({
        expectedNavText,
        navText: sample.navText,
      })}`,
    );
  }
  if (sample.navText.includes(unexpectedNavText)) {
    throw new Error(
      `${label} top nav still contains text from the wrong locale: ${JSON.stringify({
        unexpectedNavText,
        navText: sample.navText,
      })}`,
    );
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

function assertResearchChartSmoke(label, sample, viewport) {
  assertVisibleNode(label, "toolbarControls", sample.toolbarControls);
  assertVisibleNode(label, "toolbarSymbol", sample.toolbarSymbol);
  if (viewport.width > 760 && sample.toolbarSymbol.rectWidth > maxToolbarSymbolWidth) {
    throw new Error(
      `${label} symbol input is too wide for a compact chart toolbar: ${JSON.stringify({
        maxToolbarSymbolWidth,
        symbol: sample.toolbarSymbol,
        controls: sample.toolbarControls,
      })}`,
    );
  }
  assertChartViewportSmoke(label, sample, viewport, 620);
}

function assertChartViewportSmoke(label, sample, viewport, desktopMinimumHeight) {
  for (const [name, node] of [
    ["chartViewport", sample.chartViewport],
    ["chartBody", sample.chartBody],
    ["chart", sample.chart],
    ["tv", sample.tv],
  ]) {
    if (name === "chartBody" && !node) continue;
    assertVisibleNode(label, name, node);
    if (node.rectHeight > viewport.height + widthTolerance) {
      throw new Error(`${label} ${name} exceeded viewport height: ${JSON.stringify(node)}`);
    }
  }
  const minimumHeight = viewport.width <= 760 ? 500 : viewport.width <= 980 ? 600 : desktopMinimumHeight;
  if (sample.chartViewport.rectHeight < minimumHeight - widthTolerance) {
    throw new Error(
      `${label} chart viewport is too short: ${JSON.stringify({
        minimumHeight,
        viewport,
        chartViewport: sample.chartViewport,
      })}`,
    );
  }
  if (sample.chart.rectHeight - sample.chartViewport.rectHeight > widthTolerance) {
    throw new Error(
      `${label} chart height escaped fixed viewport: ${JSON.stringify({
        viewport: sample.chartViewport,
        chart: sample.chart,
      })}`,
    );
  }
  if (!sample.priceAxisCanvas) {
    throw new Error(`${label} missing right price-axis canvas: ${JSON.stringify(sample)}`);
  }
  assertChartGutters(label, sample);
  if (sample.priceAxisCanvas.rectWidth > maxRightPriceAxisWidth) {
    throw new Error(
      `${label} right price-axis is too wide: ${JSON.stringify({
        maxRightPriceAxisWidth,
        priceAxis: sample.priceAxisCanvas,
        chartViewport: sample.chartViewport,
      })}`,
    );
  }
  if (Math.abs(sample.priceAxisCanvas.right - sample.chartViewport.right) > widthTolerance + 4) {
    throw new Error(
      `${label} right price-axis does not sit on the chart viewport edge: ${JSON.stringify({
        priceAxis: sample.priceAxisCanvas,
        chartViewport: sample.chartViewport,
      })}`,
    );
  }
  if (sample.tv.right > sample.chartViewport.right + widthTolerance || sample.chartViewport.right - sample.tv.right > widthTolerance) {
    throw new Error(
      `${label} chart renderer does not fill the fixed viewport width: ${JSON.stringify({
        chartViewport: sample.chartViewport,
        tv: sample.tv,
      })}`,
    );
  }
}

function assertChartGutters(label, sample) {
  const host = sample.detail?.chartPanel ?? sample.chartBody;
  if (!host) return;
  const startGutter = sample.chartViewport.left - host.left;
  const endGutter = host.right - sample.chartViewport.right;
  assertConfiguredGutter(label, "chart left gutter", startGutter, sample.chartInlineStartGutter, { host, chartViewport: sample.chartViewport });
  assertConfiguredGutter(label, "chart right gutter", endGutter, sample.chartInlineEndGutter, { host, chartViewport: sample.chartViewport });
  if (sample.chartInlineStartGutter < 8 || sample.chartInlineStartGutter > 18) {
    throw new Error(
      `${label} chart left gutter is outside the production range: ${JSON.stringify({
        gutter: sample.chartInlineStartGutter,
        host,
        chartViewport: sample.chartViewport,
      })}`,
    );
  }
  if (sample.chartInlineEndGutter < 2 || sample.chartInlineEndGutter > 6) {
    throw new Error(
      `${label} chart right gutter should be tight so the price scale does not create excess whitespace: ${JSON.stringify({
        gutter: sample.chartInlineEndGutter,
        host,
        chartViewport: sample.chartViewport,
      })}`,
    );
  }
}

function assertConfiguredGutter(label, name, actual, expected, context) {
  if (Math.abs(actual - expected) > widthTolerance + 1) {
    throw new Error(
      `${label} ${name} does not match configured chart padding: ${JSON.stringify({
        actual,
        expected,
        ...context,
      })}`,
    );
  }
}

function assertDetailLayoutSmoke(label, sample, viewport) {
  if (!sample.detail) throw new Error(`${label} missing detail layout sample`);
  for (const [name, node] of [
    ["detailChartPanel", sample.detail.chartPanel],
    ["detailChartViewport", sample.detail.chartViewport],
    ["detailLowerGrid", sample.detail.lowerGrid],
    ["detailSummary", sample.detail.summary],
    ["detailTabs", sample.detail.tabs],
    ["chart", sample.chart],
    ["tv", sample.tv],
  ]) {
    assertVisibleNode(label, name, node);
    if (node.rectWidth > viewport.width + widthTolerance || node.right > viewport.width + widthTolerance) {
      throw new Error(`${label} ${name} escaped viewport: ${JSON.stringify(node)}`);
    }
  }
  if (sample.detail.chartPanel.rectHeight < Math.min(500, viewport.height * 0.58)) {
    throw new Error(`${label} detail chart is too short: ${JSON.stringify(sample.detail.chartPanel)}`);
  }
  if (sample.detail.chartPanel.rectHeight > viewport.height + widthTolerance) {
    throw new Error(`${label} detail chart exceeded viewport height: ${JSON.stringify(sample.detail.chartPanel)}`);
  }
  assertChartViewportSmoke(label, sample, viewport, 620);
  if (sample.detail.lowerGrid.top <= sample.detail.chartPanel.bottom) {
    throw new Error(
      `${label} detail lower grid must sit below chart: ${JSON.stringify({
        chart: sample.detail.chartPanel,
        lowerGrid: sample.detail.lowerGrid,
      })}`,
    );
  }
  if (Math.abs(sample.chart.rectHeight - sample.detail.chartViewport.rectHeight) > widthTolerance + 2) {
    throw new Error(
      `${label} detail chart does not fill fixed viewport: ${JSON.stringify({
        viewport: sample.detail.chartViewport,
        chart: sample.chart,
      })}`,
    );
  }
  if (viewport.width > 980) {
    if (sample.detail.summary.rectWidth >= sample.detail.tabs.rectWidth) {
      throw new Error(
        `${label} detail summary must be narrower than tab list: ${JSON.stringify({
          summary: sample.detail.summary,
          tabs: sample.detail.tabs,
        })}`,
      );
    }
    if (sample.detail.summary.rectWidth < 280 || sample.detail.summary.rectWidth > 360) {
      throw new Error(`${label} detail summary width is outside the expected narrow column: ${JSON.stringify(sample.detail.summary)}`);
    }
  } else if (Math.abs(sample.detail.summary.rectWidth - sample.detail.tabs.rectWidth) > widthTolerance + 2) {
    throw new Error(
      `${label} stacked detail panels should share mobile width: ${JSON.stringify({
        summary: sample.detail.summary,
        tabs: sample.detail.tabs,
      })}`,
    );
  }
}

async function launchChrome() {
  const chromePath = findChromePath();
  const port = await findOpenPort(parsePositiveInt(process.env.CHROME_REMOTE_DEBUGGING_PORT, 9233));
  chromeProfileDir = fs.mkdtempSync(path.join(os.tmpdir(), "tictick-hi-visual-smoke-"));
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
  throw new Error("Chrome executable not found. Set CHROME_PATH to run the stage8 visual smoke.");
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
  if (!response.ok) {
    response = await fetch(`${endpoint}/json/new?${encodeURIComponent(url)}`);
  }
  if (!response.ok) {
    throw new Error(`failed to create Chrome target: HTTP ${response.status}`);
  }
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
  if (result.exceptionDetails) {
    throw new Error(result.exceptionDetails.text || JSON.stringify(result.exceptionDetails));
  }
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
  console.error(`stage8 visual smoke interrupted by ${signal}`);
  cleanupChrome();
  process.exit(code);
}
