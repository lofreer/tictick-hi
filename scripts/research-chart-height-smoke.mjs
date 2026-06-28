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
const samplesPerViewport = parsePositiveInt(process.env.SMOKE_SAMPLES, 30);
const sampleIntervalMs = parsePositiveInt(process.env.SMOKE_INTERVAL_MS, 250);
const settleMs = parsePositiveInt(process.env.SMOKE_SETTLE_MS, 2000);
const heightTolerance = parsePositiveInt(process.env.SMOKE_HEIGHT_TOLERANCE, 1);
const maxViewportInset = parsePositiveInt(process.env.SMOKE_MAX_VIEWPORT_INSET, 2);

const viewports = [
  { label: "desktop-1440x900", metrics: { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false } },
  {
    label: "narrow-desktop-812x1320",
    metrics: { width: 812, height: 1320, deviceScaleFactor: 2, mobile: false },
    requireInitialChartFit: true,
  },
  { label: "mobile-390x844", metrics: { width: 390, height: 844, deviceScaleFactor: 2, mobile: true } },
];

let chrome = null;
let chromeProfileDir = null;

process.once("SIGINT", () => shutdownFromSignal("SIGINT", 130));
process.once("SIGTERM", () => shutdownFromSignal("SIGTERM", 143));

try {
  const endpoint = process.env.CDP_ENDPOINT ?? (await launchChrome());
  const results = [];
  for (const viewport of viewports) {
    results.push(await runViewport(endpoint, viewport));
  }

  for (const result of results) {
    console.log(
      [
        `${result.label}: stable`,
        `doc ${result.first.doc}->${result.last.doc}`,
        `panel ${result.first.panel}->${result.last.panel}`,
        `body ${result.first.body}->${result.last.body}`,
        `chart ${result.first.chart}->${result.last.chart}`,
        `tv ${result.first.tv}->${result.last.tv}`,
      ].join(", "),
    );
  }
  console.log("research chart height smoke passed");
} catch (error) {
  console.error("research chart height smoke failed");
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
} finally {
  cleanupChrome();
}

async function runViewport(endpoint, viewport) {
  const page = await createPage(endpoint, `${baseUrl}/`);
  const cdp = await connect(page.webSocketDebuggerUrl);
  const browserErrors = [];

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

    const login = await evaluate(
      cdp,
      `fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ username: ${JSON.stringify(username)}, password: ${JSON.stringify(password)} }),
        credentials: 'include'
      }).then(async (response) => ({ ok: response.ok, status: response.status, body: await response.text() }))`,
    );
    if (!login.ok) {
      throw new Error(`login failed for ${viewport.label}: HTTP ${login.status} ${login.body}`);
    }

    await cdp.send("Page.navigate", { url: `${baseUrl}/research` });
    await waitFor(cdp, "!!document.querySelector('.research-chart-body')", 15000);
    await waitFor(cdp, "!!document.querySelector('.tv-lightweight-charts')", 15000);
    await delay(settleMs);
    const initialSample = await evaluate(cdp, sampleExpression());
    assertChartLayout(viewport.label, initialSample);
    if (viewport.requireInitialChartFit) {
      assertInitialChartFit(viewport.label, initialSample);
    }

    const samples = [];
    for (let index = 0; index < samplesPerViewport; index += 1) {
      await polluteInternalChartHeights(cdp);
      samples.push(await evaluate(cdp, sampleExpression()));
      await delay(sampleIntervalMs);
    }

    const result = summarizeSamples(viewport.label, samples);
    assertStable(result);
    if (browserErrors.length > 0) {
      throw new Error(`${viewport.label} browser errors: ${browserErrors.join(" | ")}`);
    }
    return result;
  } finally {
    cdp.close();
  }
}

async function launchChrome() {
  const chromePath = findChromePath();
  const port = await findOpenPort(parsePositiveInt(process.env.CHROME_REMOTE_DEBUGGING_PORT, 9223));
  chromeProfileDir = fs.mkdtempSync(path.join(os.tmpdir(), "tictick-hi-chart-smoke-"));
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
  throw new Error("Chrome executable not found. Set CHROME_PATH to run the chart height smoke.");
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

function sampleExpression() {
  return `(() => {
      const read = (selector) => {
        const element = document.querySelector(selector);
        if (!element) return null;
        const rect = element.getBoundingClientRect();
        const style = getComputedStyle(element);
        return {
          className: element.className,
          classList: Array.from(element.classList),
          clientHeight: element.clientHeight,
          clientWidth: element.clientWidth,
          offsetHeight: element.offsetHeight,
          offsetWidth: element.offsetWidth,
          scrollHeight: element.scrollHeight,
          scrollWidth: element.scrollWidth,
          rectWidth: Math.round(rect.width),
          rectHeight: Math.round(rect.height),
          left: Math.round(rect.left),
          right: Math.round(rect.right),
          styleHeight: style.height,
          overflowX: style.overflowX,
          contain: style.contain,
          overflowY: style.overflowY
        };
      };
      const body = read('.research-chart-body');
      const tv = read('.tv-lightweight-charts');
      const canvasEntries = Array.from(document.querySelectorAll('.trading-chart__canvas canvas')).map((canvas, index) => {
        const rect = canvas.getBoundingClientRect();
        const style = getComputedStyle(canvas);
        return {
          canvas,
          metrics: {
            index,
            rectWidth: Math.round(rect.width),
            rectHeight: Math.round(rect.height),
            top: Math.round(rect.top),
            bottom: Math.round(rect.bottom),
            left: Math.round(rect.left),
            right: Math.round(rect.right),
            styleWidth: style.width,
            styleHeight: style.height
          }
        };
      });
      const canvases = canvasEntries.map((entry) => entry.metrics);
      const rightAxisCanvas = canvases
        .filter((canvas) => canvas.rectWidth >= 72 && canvas.rectWidth <= 180)
        .filter((canvas) => body ? canvas.rectHeight >= Math.max(120, body.rectHeight - 96) : true)
        .sort((left, right) => right.right - left.right)[0] ?? null;
      const mainPaneCanvases = canvases
        .filter((canvas) => body ? canvas.rectWidth >= Math.max(120, body.rectWidth - 240) : true)
        .filter((canvas) => body ? canvas.rectHeight >= Math.max(120, body.rectHeight - 96) : true);
      const mainPaneCanvas = mainPaneCanvases.sort((left, right) => left.left - right.left)[0] ?? null;
      const mainPaneColorStats = marketColorStats(
        canvasEntries
          .filter((entry) => mainPaneCanvases.some((canvas) => canvas.index === entry.metrics.index))
          .map((entry) => entry.canvas)
      );
      const bottomTimeAxisCanvas = canvases
        .filter((canvas) => canvas.rectHeight >= 16 && canvas.rectHeight <= 80)
        .filter((canvas) => body ? canvas.rectWidth >= Math.max(120, body.rectWidth - 240) : true)
        .sort((left, right) => right.bottom - left.bottom)[0] ?? null;
      const bottomTimeAxisElement = canvasEntries.find((entry) => entry.metrics.index === bottomTimeAxisCanvas?.index)?.canvas ?? null;
      return {
        href: location.href,
        viewportWidth: innerWidth,
        bodyScrollHeight: document.body.scrollHeight,
        docScrollHeight: document.documentElement.scrollHeight,
        viewportHeight: innerHeight,
        taskPanel: read('.research-tasks-panel'),
        panel: read('.research-chart-panel'),
        body,
        chart: read('.trading-chart'),
        canvas: read('.trading-chart__canvas'),
        tv,
        canvases,
        mainPaneCanvas,
        mainPaneColorStats,
        rightAxisCanvas,
        bottomTimeAxisCanvas,
        bottomTimeAxisEdgeInk: edgeInkStats(bottomTimeAxisElement),
        chartCount: document.querySelectorAll('.tv-lightweight-charts').length
      };

      function marketColorStats(canvasElements) {
        const rows = new Set();
        const columns = new Set();
        let coloredPixels = 0;
        for (const canvas of canvasElements) {
          const width = canvas.width;
          const height = canvas.height;
          if (width <= 0 || height <= 0) continue;
          const context = canvas.getContext('2d', { willReadFrequently: true });
          if (!context) continue;
          const pixels = context.getImageData(0, 0, width, height).data;
          for (let index = 0; index < pixels.length; index += 4) {
            const red = pixels[index];
            const green = pixels[index + 1];
            const blue = pixels[index + 2];
            const alpha = pixels[index + 3];
            const up = alpha > 40 && green > 120 && red < 110 && blue < 190;
            const down = alpha > 40 && red > 180 && green < 150 && blue < 170;
            if (!up && !down) continue;
            const pixel = index / 4;
            coloredPixels += 1;
            rows.add(Math.floor(pixel / width));
            columns.add(pixel % width);
          }
        }
        return {
          coloredColumns: columns.size,
          coloredPixels,
          coloredRows: rows.size
        };
      }

      function edgeInkStats(canvas, edgeWidth = 8) {
        if (!canvas || canvas.width <= edgeWidth * 2 || canvas.height <= 0) {
          return null;
        }
        const context = canvas.getContext('2d', { willReadFrequently: true });
        if (!context) return null;
        const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data;
        let leftDarkPixels = 0;
        let rightDarkPixels = 0;
        for (let y = 0; y < canvas.height; y += 1) {
          for (let x = 0; x < edgeWidth; x += 1) {
            if (isDarkInk(pixels, (y * canvas.width + x) * 4)) leftDarkPixels += 1;
            if (isDarkInk(pixels, (y * canvas.width + (canvas.width - 1 - x)) * 4)) rightDarkPixels += 1;
          }
        }
        return { edgeWidth, leftDarkPixels, rightDarkPixels };
      }

      function isDarkInk(pixels, index) {
        const red = pixels[index];
        const green = pixels[index + 1];
        const blue = pixels[index + 2];
        const alpha = pixels[index + 3];
        return alpha > 40 && red < 180 && green < 180 && blue < 190;
      }
    })()`;
}

async function polluteInternalChartHeights(cdp) {
  await evaluate(
    cdp,
    `(() => {
      for (const selector of [
        '.research-chart-body',
        '.tv-lightweight-charts',
        '.tv-lightweight-charts table',
        '.tv-lightweight-charts tbody',
        '.tv-lightweight-charts tr',
        '.tv-lightweight-charts td',
        '.trading-chart__canvas canvas'
      ]) {
        for (const element of document.querySelectorAll(selector)) {
          element.style.height = '9000px';
          element.style.maxHeight = '9000px';
          element.style.blockSize = '9000px';
          element.style.maxBlockSize = '9000px';
        }
      }
      return true;
    })()`,
  );
}

function summarizeSamples(label, samples) {
  const firstSample = samples[0];
  const lastSample = samples[samples.length - 1];
  const keys = ["doc", "panel", "body", "chart", "canvas", "tv"];
  const values = samples.map((sample) => compactSample(sample));
  const min = {};
  const max = {};
  for (const key of keys) {
    min[key] = Math.min(...values.map((value) => value[key]));
    max[key] = Math.max(...values.map((value) => value[key]));
  }
  return {
    label,
    first: compactSample(firstSample),
    last: compactSample(lastSample),
    min,
    max,
    samples: samples.length,
    chartCount: lastSample.chartCount,
    firstFull: firstSample,
    lastFull: lastSample,
  };
}

function compactSample(sample) {
  return {
    doc: sample.docScrollHeight,
    panel: sample.panel?.rectHeight ?? 0,
    body: sample.body?.rectHeight ?? 0,
    chart: sample.chart?.rectHeight ?? 0,
    canvas: sample.canvas?.rectHeight ?? 0,
    tv: sample.tv?.rectHeight ?? 0,
  };
}

function assertChartLayout(label, sample) {
  const { body, tv, mainPaneCanvas, rightAxisCanvas, bottomTimeAxisCanvas } = sample;
  if (sample.panel?.classList?.includes("chart-panel")) {
    throw new Error(`${label} research chart panel must not inherit the global chart-panel sizing contract`);
  }
  if (sample.taskPanel?.overflowX === "hidden" || sample.taskPanel?.overflowY === "hidden") {
    throw new Error(`${label} research task panel must expose scrollable overflow instead of clipping table columns`);
  }
  if (!body || !tv) {
    throw new Error(`${label} missing chart layout nodes`);
  }
  if (tv.left < body.left - 1 || tv.top < body.top - 1) {
    throw new Error(
      `${label} chart root is clipped before the fixed body: ${JSON.stringify({
        body,
        tv,
      })}`,
    );
  }
  if (!mainPaneCanvas) {
    throw new Error(
      `${label} missing bounded main pane canvas: ${JSON.stringify({
        body,
        tv,
        canvases: sample.canvases,
      })}`,
    );
  }
  if (
    mainPaneCanvas.left < body.left - 1 ||
    mainPaneCanvas.top < body.top - 1 ||
    mainPaneCanvas.right > body.right + 1 ||
    mainPaneCanvas.bottom > body.bottom + 1
  ) {
    throw new Error(
      `${label} chart main pane canvas is clipped by fixed body: ${JSON.stringify({
        body,
        tv,
        mainPaneCanvas,
      })}`,
    );
  }
  if (
    !sample.mainPaneColorStats ||
    sample.mainPaneColorStats.coloredPixels < 80 ||
    sample.mainPaneColorStats.coloredRows < 12 ||
    sample.mainPaneColorStats.coloredColumns < 12
  ) {
    throw new Error(
      `${label} main pane has no visible candle pixels: ${JSON.stringify({
        body,
        tv,
        mainPaneCanvas,
        mainPaneColorStats: sample.mainPaneColorStats,
      })}`,
    );
  }
  if (!rightAxisCanvas) {
    throw new Error(
      `${label} missing bounded right price-axis canvas: ${JSON.stringify({
        body,
        tv,
        canvases: sample.canvases,
      })}`,
    );
  }
  if (rightAxisCanvas.right > body.right + 1 || rightAxisCanvas.bottom > body.bottom + 1 || tv.right > body.right + 1) {
    throw new Error(
      `${label} chart right edge overflowed fixed body: ${JSON.stringify({
        body,
        tv,
        rightAxisCanvas,
      })}`,
    );
  }
  if (body.right - rightAxisCanvas.right > maxViewportInset || body.right - tv.right > maxViewportInset) {
    throw new Error(
      `${label} chart right side leaves unused fixed body space: ${JSON.stringify({
        maxInset: maxViewportInset,
        body,
        tv,
        rightAxisCanvas,
      })}`,
    );
  }
  if (!bottomTimeAxisCanvas) {
    throw new Error(
      `${label} missing bounded bottom time-axis canvas: ${JSON.stringify({
        body,
        tv,
        canvases: sample.canvases,
      })}`,
    );
  }
  if (
    !sample.bottomTimeAxisEdgeInk ||
    sample.bottomTimeAxisEdgeInk.leftDarkPixels > 0 ||
    sample.bottomTimeAxisEdgeInk.rightDarkPixels > 0
  ) {
    throw new Error(
      `${label} time-axis label touches fixed body edge: ${JSON.stringify({
        body,
        bottomTimeAxisCanvas,
        bottomTimeAxisEdgeInk: sample.bottomTimeAxisEdgeInk,
      })}`,
    );
  }
  if (body.bottom - bottomTimeAxisCanvas.bottom > maxViewportInset || body.bottom - tv.bottom > maxViewportInset) {
    throw new Error(
      `${label} chart bottom side leaves unused fixed body space: ${JSON.stringify({
        maxInset: maxViewportInset,
        body,
        tv,
        bottomTimeAxisCanvas,
      })}`,
    );
  }
  if (bottomTimeAxisCanvas.bottom > body.bottom + 1 || bottomTimeAxisCanvas.right > body.right + 1 || tv.bottom > body.bottom + 1) {
    throw new Error(
      `${label} chart bottom edge overflowed fixed body: ${JSON.stringify({
        body,
        tv,
        bottomTimeAxisCanvas,
      })}`,
    );
  }
  for (const [name, node] of [
    ["panel", sample.panel],
    ["body", sample.body],
    ["chart", sample.chart],
    ["canvas", sample.canvas],
    ["tv", sample.tv],
  ]) {
    if (!node) continue;
    if (node.scrollWidth > node.clientWidth + 1 || node.rectWidth > sample.viewportWidth + 1) {
      throw new Error(
        `${label} ${name} overflowed horizontally: ${JSON.stringify({
          viewportWidth: sample.viewportWidth,
          node,
        })}`,
      );
    }
  }
}

function assertInitialChartFit(label, sample) {
  const { body, bottomTimeAxisCanvas, viewportHeight } = sample;
  if (!body || !bottomTimeAxisCanvas) return;

  const bottomPadding = 16;
  const maxBottom = viewportHeight - bottomPadding;
  if (body.bottom > maxBottom || bottomTimeAxisCanvas.bottom > maxBottom) {
    throw new Error(
      `${label} chart bottom axis is clipped from the initial viewport: ${JSON.stringify({
        viewportHeight,
        maxBottom,
        body,
        bottomTimeAxisCanvas,
      })}`,
    );
  }
}

function assertStable(result) {
  if (result.chartCount !== 1) {
    throw new Error(`${result.label} expected one chart, got ${result.chartCount}`);
  }
  for (const key of Object.keys(result.max)) {
    const spread = result.max[key] - result.min[key];
    if (spread > heightTolerance) {
      throw new Error(
        `${result.label} ${key} height changed by ${spread}px: ${JSON.stringify({
          first: result.first,
          last: result.last,
          min: result.min,
          max: result.max,
        })}`,
      );
    }
  }

  const viewportCap = result.lastFull.viewportHeight + heightTolerance;
  for (const key of ["panel", "body", "chart", "canvas", "tv"]) {
    if (result.max[key] > viewportCap) {
      throw new Error(
        `${result.label} ${key} height exceeded viewport cap: ${JSON.stringify({
          viewportHeight: result.lastFull.viewportHeight,
          max: result.max,
        })}`,
      );
    }
  }

  const fixedBodyHeight = result.last.body;
  for (const key of ["chart", "canvas", "tv"]) {
    const overflow = result.last[key] - fixedBodyHeight;
    if (overflow > heightTolerance) {
      throw new Error(
        `${result.label} ${key} height overflowed fixed body by ${overflow}px: ${JSON.stringify({
          body: fixedBodyHeight,
          last: result.last,
          min: result.min,
          max: result.max,
        })}`,
      );
    }
    const inset = fixedBodyHeight - result.last[key];
    if (inset > maxViewportInset) {
      throw new Error(
        `${result.label} ${key} left too much unused fixed body height: ${JSON.stringify({
          maxInset: maxViewportInset,
          body: fixedBodyHeight,
          last: result.last,
          min: result.min,
          max: result.max,
        })}`,
      );
    }
  }
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
  console.error(`research chart height smoke interrupted by ${signal}`);
  cleanupChrome();
  process.exit(code);
}
