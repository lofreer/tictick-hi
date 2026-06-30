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
const maxRightPriceAxisWidth = parsePositiveInt(process.env.SMOKE_MAX_RIGHT_PRICE_AXIS_WIDTH, 114);
const minAxisLabelInkHeight = parsePositiveInt(process.env.SMOKE_MIN_AXIS_LABEL_INK_HEIGHT, 14);
const maxTimeAxisEdgeInkPixels = parsePositiveInt(process.env.SMOKE_MAX_TIME_AXIS_EDGE_INK, 64);
const totalTimeoutMs = parsePositiveInt(process.env.SMOKE_TOTAL_TIMEOUT_MS, 5 * 60 * 1000);

const viewports = [
  {
    label: "desktop-1440x900",
    metrics: { width: 1440, height: 900, deviceScaleFactor: 1, mobile: false },
  },
  {
    label: "desktop-2048x1152",
    metrics: { width: 2048, height: 1152, deviceScaleFactor: 1, mobile: false },
  },
  {
    label: "narrow-desktop-812x1320",
    metrics: { width: 812, height: 1320, deviceScaleFactor: 2, mobile: false },
  },
  { label: "mobile-390x844", metrics: { width: 390, height: 844, deviceScaleFactor: 2, mobile: true } },
];

let chrome = null;
let chromeProfileDir = null;
let smokeTimedOut = false;
const activeSockets = new Set();

process.once("SIGINT", () => shutdownFromSignal("SIGINT", 130));
process.once("SIGTERM", () => shutdownFromSignal("SIGTERM", 143));

try {
  await withTotalTimeout(runSmoke(), totalTimeoutMs, "research chart height smoke");
} catch (error) {
  console.error("research chart height smoke failed");
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
} finally {
  cleanupChrome();
}

async function runSmoke() {
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

    await evaluate(
      cdp,
      `(() => {
        localStorage.setItem('tictick-hi.theme', 'light');
        document.documentElement.dataset.theme = 'light';
        return true;
      })()`,
    );
    await cdp.send("Page.navigate", { url: `${baseUrl}/research` });
    await waitFor(cdp, "!!document.querySelector('.research-chart-body')", 15000);
    await waitFor(cdp, "!!document.querySelector('.tv-lightweight-charts')", 15000);
    await delay(settleMs);
    const initialSample = await evaluate(cdp, sampleExpression());
    assertChartLayout(viewport.label, initialSample);

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
    await closePage(endpoint, page.id);
  }
}

async function launchChrome() {
  const chromePath = findChromePath();
  const port = await findOpenPort(parsePositiveInt(process.env.CHROME_REMOTE_DEBUGGING_PORT, 9223));
  if (smokeTimedOut) throw new Error("research chart height smoke aborted after total timeout");
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
    if (smokeTimedOut) throw new Error("research chart height smoke aborted after total timeout");
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

async function closePage(endpoint, targetId) {
  if (!targetId) return;
  try {
    await fetch(`${endpoint}/json/close/${encodeURIComponent(targetId)}`);
  } catch {
    // Best-effort cleanup; Chrome process cleanup still runs at script exit.
  }
}

function connect(wsUrl) {
  const ws = new WebSocket(wsUrl);
  activeSockets.add(ws);
  ws.addEventListener("close", () => activeSockets.delete(ws), { once: true });
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
          top: Math.round(rect.top),
          bottom: Math.round(rect.bottom),
          left: Math.round(rect.left),
          right: Math.round(rect.right),
          styleHeight: style.height,
          overflowX: style.overflowX,
          contain: style.contain,
          overflowY: style.overflowY
        };
      };
      const body = read('.research-chart-body');
      const chartInlineStartGutter = cssPixel('.research-chart-body', 'padding-left');
      const chartInlineEndGutter = cssPixel('.research-chart-body', 'padding-right');
      const chartBlockStartGutter = cssPixel('.research-chart-body', 'padding-top');
      const chartBlockEndGutter = cssPixel('.research-chart-body', 'padding-bottom');
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
      const rightAxisEntry = canvasEntries
        .filter((entry) => entry.metrics.rectWidth >= 24 && entry.metrics.rectWidth <= 180)
        .filter((entry) => body ? entry.metrics.rectHeight >= Math.max(120, body.rectHeight - 96) : true)
        .sort((left, right) => right.metrics.right - left.metrics.right)[0] ?? null;
      const rightAxisCanvas = rightAxisEntry?.metrics ?? null;
      const mainPaneCanvases = canvases
        .filter((canvas) => body ? canvas.rectWidth >= Math.max(120, body.rectWidth - 240) : true)
        .filter((canvas) => body ? canvas.rectHeight >= Math.max(120, body.rectHeight - 96) : true);
      const mainPaneCanvas = mainPaneCanvases.sort((left, right) => left.left - right.left)[0] ?? null;
      const mainPaneColorStats = marketColorStats(
        canvasEntries
          .filter((entry) => mainPaneCanvases.some((canvas) => canvas.index === entry.metrics.index))
          .map((entry) => entry.canvas)
      );
      const bottomTimeAxisEntry = canvasEntries
        .filter((entry) => entry.metrics.rectHeight >= 16 && entry.metrics.rectHeight <= 80)
        .filter((entry) => body ? entry.metrics.rectWidth >= Math.max(120, body.rectWidth - 240) : true)
        .sort((left, right) => right.metrics.bottom - left.metrics.bottom)[0] ?? null;
      const bottomTimeAxisCanvas = bottomTimeAxisEntry?.metrics ?? null;
      return {
        href: location.href,
        viewportWidth: innerWidth,
        bodyScrollWidth: document.body.scrollWidth,
        bodyScrollHeight: document.body.scrollHeight,
        docScrollWidth: document.documentElement.scrollWidth,
        docScrollHeight: document.documentElement.scrollHeight,
        scrollX: Math.round(window.scrollX),
        viewportHeight: innerHeight,
        taskPanel: read('.research-tasks-panel'),
        panel: read('.research-chart-panel'),
        body,
        chart: read('.trading-chart'),
        canvas: read('.trading-chart__canvas'),
        tv,
        chartInlineStartGutter,
        chartInlineEndGutter,
        chartBlockStartGutter,
        chartBlockEndGutter,
        canvases,
        mainPaneCanvas,
        mainPaneColorStats,
        rightAxisCanvas,
        priceAxisTextInk: axisTextInkStats(rightAxisEntry?.canvas ?? null),
        bottomTimeAxisCanvas,
        timeAxisTextInk: axisTextInkStats(bottomTimeAxisEntry?.canvas ?? null),
        bottomTimeAxisEdgeInk: edgeInkStats(bottomTimeAxisEntry?.canvas ?? null),
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

      function axisTextInkStats(canvas) {
        if (!canvas || canvas.width <= 0 || canvas.height <= 0) return null;
        const context = canvas.getContext('2d', { willReadFrequently: true });
        if (!context) return null;
        const pixels = context.getImageData(0, 0, canvas.width, canvas.height).data;
        const darkTheme = document.documentElement.dataset.theme === 'dark';
        const scaleY = canvas.height / Math.max(1, canvas.getBoundingClientRect().height);
        const inkRows = [];
        for (let y = 0; y < canvas.height; y += 1) {
          let rowInk = 0;
          for (let x = 0; x < canvas.width; x += 1) {
            if (isAxisTextInk(pixels, (y * canvas.width + x) * 4, darkTheme)) rowInk += 1;
          }
          if (rowInk >= 2) inkRows.push(y);
        }
        const runs = rowRuns(inkRows, 2).map((run) => ({
          start: run.start,
          end: run.end,
          height: run.end - run.start + 1,
          cssHeight: (run.end - run.start + 1) / scaleY
        }));
        return {
          canvasWidth: canvas.width,
          canvasHeight: canvas.height,
          scaleY,
          runCount: runs.length,
          maxRunCssHeight: Math.max(0, ...runs.map((run) => run.cssHeight)),
          runs: runs.slice(0, 8)
        };
      }

      function rowRuns(rows, allowedGap) {
        if (rows.length === 0) return [];
        const runs = [];
        let start = rows[0];
        let previous = rows[0];
        for (let index = 1; index < rows.length; index += 1) {
          const row = rows[index];
          if (row - previous <= allowedGap + 1) {
            previous = row;
            continue;
          }
          runs.push({ start, end: previous });
          start = row;
          previous = row;
        }
        runs.push({ start, end: previous });
        return runs;
      }

      function isAxisTextInk(pixels, index, darkTheme) {
        const red = pixels[index];
        const green = pixels[index + 1];
        const blue = pixels[index + 2];
        const alpha = pixels[index + 3];
        if (alpha < 80) return false;
        if (darkTheme) return red > 145 && green > 145 && blue > 145;
        return red < 150 && green < 150 && blue < 170;
      }

      function isDarkInk(pixels, index) {
        const red = pixels[index];
        const green = pixels[index + 1];
        const blue = pixels[index + 2];
        const alpha = pixels[index + 3];
        return alpha > 40 && red < 180 && green < 180 && blue < 190;
      }

      function cssPixel(selector, property) {
        const element = document.querySelector(selector);
        if (!element) return 0;
        const value = Number.parseFloat(getComputedStyle(element).getPropertyValue(property));
        return Number.isFinite(value) && value > 0 ? value : 0;
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
  if (sample.scrollX !== 0 || sample.docScrollWidth > sample.viewportWidth + 1 || sample.bodyScrollWidth > sample.viewportWidth + 1) {
    throw new Error(
      `${label} page overflowed horizontally and can clip the chart viewport: ${JSON.stringify({
        viewportWidth: sample.viewportWidth,
        scrollX: sample.scrollX,
        docScrollWidth: sample.docScrollWidth,
        bodyScrollWidth: sample.bodyScrollWidth,
        taskPanel: sample.taskPanel,
        panel: sample.panel,
      })}`,
    );
  }
  if (!body || !tv) {
    throw new Error(`${label} missing chart layout nodes`);
  }
  const expectedMinimumPlotHeight = sample.viewportWidth <= 760 ? 540 : sample.viewportWidth <= 980 ? 620 : 600;
  if (tv.rectHeight < expectedMinimumPlotHeight - heightTolerance) {
    throw new Error(
      `${label} chart plot is too short for the viewport: ${JSON.stringify({
        expectedMinimumPlotHeight,
        body,
        tv,
      })}`,
    );
  }
  if (tv.left < body.left - 1 || tv.top < body.top - 1) {
    throw new Error(
      `${label} chart root is clipped before the fixed body: ${JSON.stringify({
        body,
        tv,
      })}`,
    );
  }
  assertConfiguredInset(label, "chart left side", tv.left - body.left, sample.chartInlineStartGutter, { body, tv });
  assertConfiguredInset(label, "chart top side", tv.top - body.top, sample.chartBlockStartGutter, { body, tv });
  if (sample.chartInlineStartGutter < 8 || sample.chartInlineStartGutter > 22) {
    throw new Error(
      `${label} chart left gutter is outside the production range: ${JSON.stringify({
        chartInlineStartGutter: sample.chartInlineStartGutter,
        body,
        tv,
      })}`,
    );
  }
  if (sample.chartInlineEndGutter < 2 || sample.chartInlineEndGutter > 8) {
    throw new Error(
      `${label} chart right gutter is outside the production range: ${JSON.stringify({
        chartInlineEndGutter: sample.chartInlineEndGutter,
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
  const mainPaneShare = mainPaneCanvas.rectWidth / tv.rectWidth;
  const minimumMainPaneShare = sample.viewportWidth <= 760 ? 0.665 : sample.viewportWidth <= 980 ? 0.84 : 0.9;
  if (mainPaneShare < minimumMainPaneShare) {
    throw new Error(
      `${label} main pane does not use enough chart width: ${JSON.stringify({
        minimumMainPaneShare,
        mainPaneShare,
        body,
        mainPaneCanvas,
        rightAxisCanvas,
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
  if (rightAxisCanvas.rectWidth > maxRightPriceAxisWidth) {
    throw new Error(
      `${label} right price-axis is too wide: ${JSON.stringify({
        maxRightPriceAxisWidth,
        body,
        tv,
        rightAxisCanvas,
      })}`,
    );
  }
  assertAxisTextInk(label, "right price-axis", sample.priceAxisTextInk, {
    body,
    tv,
    rightAxisCanvas,
  });
  if (Math.abs(rightAxisCanvas.left - mainPaneCanvas.right) > 1) {
    throw new Error(
      `${label} main chart pane is detached from the right price-axis: ${JSON.stringify({
        body,
        tv,
        mainPaneCanvas,
        rightAxisCanvas,
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
  assertConfiguredInset(label, "chart right side", body.right - tv.right, sample.chartInlineEndGutter, { body, tv });
  assertConfiguredInset(label, "right price-axis", body.right - rightAxisCanvas.right, sample.chartInlineEndGutter, {
    body,
    rightAxisCanvas,
  });
  if (!bottomTimeAxisCanvas) {
    throw new Error(
      `${label} missing bounded bottom time-axis canvas: ${JSON.stringify({
        body,
        tv,
        canvases: sample.canvases,
      })}`,
    );
  }
  assertAxisTextInk(label, "bottom time-axis", sample.timeAxisTextInk, {
    body,
    tv,
    bottomTimeAxisCanvas,
  });
  if (
    !sample.bottomTimeAxisEdgeInk ||
    sample.bottomTimeAxisEdgeInk.leftDarkPixels > maxTimeAxisEdgeInkPixels ||
    sample.bottomTimeAxisEdgeInk.rightDarkPixels > maxTimeAxisEdgeInkPixels
  ) {
    throw new Error(
      `${label} time-axis label touches fixed body edge: ${JSON.stringify({
        maxTimeAxisEdgeInkPixels,
        body,
        bottomTimeAxisCanvas,
        bottomTimeAxisEdgeInk: sample.bottomTimeAxisEdgeInk,
      })}`,
    );
  }
  assertConfiguredInset(label, "chart bottom side", body.bottom - tv.bottom, sample.chartBlockEndGutter, { body, tv });
  assertConfiguredInset(label, "bottom time-axis", body.bottom - bottomTimeAxisCanvas.bottom, sample.chartBlockEndGutter, {
    body,
    bottomTimeAxisCanvas,
  });
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
  for (const key of ["body", "chart", "canvas", "tv"]) {
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
  const expectedBlockStartInset = result.lastFull.chartBlockStartGutter ?? 0;
  const expectedBlockEndInset = result.lastFull.chartBlockEndGutter ?? 0;
  const expectedChartHeight = fixedBodyHeight - expectedBlockStartInset - expectedBlockEndInset;
  for (const key of ["chart", "canvas", "tv"]) {
    const overflow = result.last[key] - expectedChartHeight;
    if (overflow > heightTolerance) {
      throw new Error(
        `${result.label} ${key} height overflowed fixed body by ${overflow}px: ${JSON.stringify({
          body: fixedBodyHeight,
          expectedBlockStartInset,
          expectedBlockEndInset,
          expectedChartHeight,
          last: result.last,
          min: result.min,
          max: result.max,
        })}`,
      );
    }
    const inset = fixedBodyHeight - result.last[key];
    const expectedInset = expectedBlockStartInset + expectedBlockEndInset;
    if (Math.abs(inset - expectedInset) > heightTolerance) {
      throw new Error(
        `${result.label} ${key} height does not match configured fixed body inset: ${JSON.stringify({
          expectedBlockStartInset,
          expectedBlockEndInset,
          expectedInset,
          body: fixedBodyHeight,
          last: result.last,
          min: result.min,
          max: result.max,
        })}`,
      );
    }
  }
}

function assertAxisTextInk(label, name, textInk, context) {
  if (!textInk || textInk.maxRunCssHeight < minAxisLabelInkHeight) {
    throw new Error(
      `${label} ${name} text is too small or missing: ${JSON.stringify({
        minAxisLabelInkHeight,
        textInk,
        ...context,
      })}`,
    );
  }
}

function assertConfiguredInset(label, name, actual, expected, context) {
  if (Math.abs(actual - expected) <= maxViewportInset) return;
  throw new Error(
    `${label} ${name} does not match configured fixed body inset: ${JSON.stringify({
      expected,
      actual,
      tolerance: maxViewportInset,
      ...context,
    })}`,
  );
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

async function withTotalTimeout(promise, timeoutMs, label) {
  let timeout;
  try {
    return await Promise.race([
      promise,
      new Promise((_, reject) => {
        timeout = setTimeout(() => {
          smokeTimedOut = true;
          cleanupChrome();
          reject(new Error(`${label} exceeded total timeout ${timeoutMs}ms`));
        }, timeoutMs);
        timeout.unref?.();
      }),
    ]);
  } finally {
    clearTimeout(timeout);
    promise.catch(() => {
      // Timeout cleanup can reject the still-unwinding browser flow after the race settles.
    });
  }
}

function cleanupChrome() {
  for (const socket of activeSockets) {
    try {
      socket.close();
    } catch {
      // Best-effort cleanup; Chrome process cleanup follows.
    }
  }
  activeSockets.clear();
  if (chrome) {
    chrome.kill("SIGTERM");
    chrome = null;
  }
  if (chromeProfileDir) {
    fs.rmSync(chromeProfileDir, { recursive: true, force: true, maxRetries: 5, retryDelay: 100 });
    chromeProfileDir = null;
  }
}

function shutdownFromSignal(signal, code) {
  console.error(`research chart height smoke interrupted by ${signal}`);
  cleanupChrome();
  process.exit(code);
}
