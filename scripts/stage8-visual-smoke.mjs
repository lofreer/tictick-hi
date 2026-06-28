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

const viewports = [
  { label: "desktop-1440x900", metrics: { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false } },
  { label: "mobile-390x844", metrics: { width: 390, height: 844, deviceScaleFactor: 2, mobile: true } },
];

const themes = ["light", "dark"];

const pages = [
  { label: "overview", path: "/overview", selector: ".overview-metrics" },
  { label: "research", path: "/research", selector: ".research-workspace", chart: true },
  { label: "backtests", path: "/backtests", selector: ".backtests-panel" },
  { label: "trading", path: "/trading", selector: ".trading-panel" },
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
      results.push(await runVisualPass(endpoint, viewport, theme));
    }
  }

  for (const result of results) {
    console.log(
      `${result.viewport}/${result.theme}: ${result.pages.length} pages, max document width ${result.maxDocumentWidth}px`,
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

async function runVisualPass(endpoint, viewport, theme) {
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
    await login(cdp, viewport.label, theme);

    for (const pageConfig of pages) {
      await setTheme(cdp, theme);
      await cdp.send("Page.navigate", { url: `${baseUrl}${pageConfig.path}` });
      await waitFor(cdp, "!!document.querySelector('.app-shell')", 15000);
      await waitFor(cdp, "!!document.querySelector('.page-title')", 15000);
      await waitFor(cdp, `!!document.querySelector(${JSON.stringify(pageConfig.selector)})`, 15000);
      if (pageConfig.chart) {
        await waitFor(cdp, "!!document.querySelector('.research-chart-body')", 15000);
      }
      await delay(settleMs);

      const sample = await evaluate(cdp, visualSampleExpression(pageConfig.selector));
      assertPageLayout(viewport, theme, pageConfig, sample);
      pageResults.push({ label: pageConfig.label, sample });
    }

    if (browserErrors.length > 0) {
      throw new Error(`${viewport.label}/${theme} browser errors: ${browserErrors.join(" | ")}`);
    }

    return {
      viewport: viewport.label,
      theme,
      pages: pageResults,
      maxDocumentWidth: Math.max(...pageResults.map((result) => result.sample.documentWidth)),
    };
  } finally {
    cdp.close();
  }
}

async function login(cdp, viewportLabel, theme) {
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
    throw new Error(`${viewportLabel}/${theme} login failed: HTTP ${loginResult.status} ${loginResult.body}`);
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

function visualSampleExpression(requiredSelector) {
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
    const root = document.documentElement;
    const body = document.body;
    return {
      href: location.href,
      theme: document.documentElement.dataset.theme || '',
      viewportWidth: innerWidth,
      viewportHeight: innerHeight,
      documentWidth: Math.max(root.scrollWidth, body.scrollWidth, root.clientWidth, body.clientWidth),
      documentHeight: Math.max(root.scrollHeight, body.scrollHeight, root.clientHeight, body.clientHeight),
      shell: read('.app-shell'),
      header: read('.app-header'),
      main: read('.app-main'),
      page: read('.page'),
      title: read('.page-title'),
      required: read(${JSON.stringify(requiredSelector)}),
      chartBody: read('.research-chart-body'),
      chart: read('.trading-chart'),
      tv: read('.tv-lightweight-charts')
    };
  })()`;
}

function assertPageLayout(viewport, theme, page, sample) {
  const label = `${viewport.label}/${theme}/${page.label}`;
  if (sample.theme !== theme) {
    throw new Error(`${label} theme = ${sample.theme}, want ${theme}`);
  }
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
  if (page.chart) assertChartSmoke(label, sample, viewport.metrics.height);
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

function assertChartSmoke(label, sample, viewportHeight) {
  for (const [name, node] of [
    ["chartBody", sample.chartBody],
    ["chart", sample.chart],
    ["tv", sample.tv],
  ]) {
    assertVisibleNode(label, name, node);
    if (node.rectHeight > viewportHeight + widthTolerance) {
      throw new Error(`${label} ${name} exceeded viewport height: ${JSON.stringify(node)}`);
    }
  }
  if (sample.chart.rectHeight - sample.chartBody.rectHeight > widthTolerance) {
    throw new Error(
      `${label} chart height escaped fixed body: ${JSON.stringify({
        body: sample.chartBody,
        chart: sample.chart,
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
