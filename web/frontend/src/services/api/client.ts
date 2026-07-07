export type ApiClientOptions = {
  baseUrl?: string;
};

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly code?: string,
    readonly payload?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type JsonBody = Record<string, unknown> | unknown[];
type JsonRequestInit = Omit<RequestInit, "body"> & { body?: JsonBody };
const csrfCookieName = "tictick_hi_csrf";
const csrfHeaderName = "X-CSRF-Token";

export class ApiClient {
  private readonly baseUrl: string;

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? "/api";
  }

  get<T>(path: string) {
    return this.request<T>(path, { method: "GET" });
  }

  post<T>(path: string, body?: JsonBody) {
    return this.request<T>(path, { method: "POST", body });
  }

  put<T>(path: string, body?: JsonBody) {
    return this.request<T>(path, { method: "PUT", body });
  }

  delete<T>(path: string) {
    return this.request<T>(path, { method: "DELETE" });
  }

  private async request<T>(path: string, init: JsonRequestInit) {
    const headers = requestHeaders(init);
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers,
      credentials: "same-origin",
      body: init.body === undefined ? undefined : JSON.stringify(init.body),
    });

    const payload = await readJson(response);
    if (!response.ok) {
      const error = apiErrorDetails(payload, response.statusText || "Request failed");
      throw new ApiError(error.message, response.status, error.code, payload);
    }

    return payload as T;
  }
}

async function readJson(response: Response) {
  const text = await response.text();
  return text.length > 0 ? JSON.parse(text) : null;
}

function apiErrorDetails(payload: unknown, fallback: string) {
  if (isRecord(payload)) {
    const message = stringField(payload, "message") || stringField(payload, "error");
    return {
      code: stringField(payload, "code") || undefined,
      message: message || fallback,
    };
  }
  return { code: undefined, message: fallback };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function stringField(value: Record<string, unknown>, key: string) {
  const field = value[key];
  return typeof field === "string" ? field : "";
}

function requestHeaders(init: JsonRequestInit) {
  const headers = new Headers({
    Accept: "application/json",
    "Content-Type": "application/json",
  });
  new Headers(init.headers).forEach((value, key) => headers.set(key, value));
  if (!isSafeMethod(init.method ?? "GET")) {
    const token = readCookie(csrfCookieName);
    if (token !== "") {
      headers.set(csrfHeaderName, token);
    }
  }
  return headers;
}

function isSafeMethod(method: string) {
  const normalized = method.toUpperCase();
  return normalized === "GET" || normalized === "HEAD" || normalized === "OPTIONS";
}

function readCookie(name: string) {
  if (typeof document === "undefined") {
    return "";
  }
  const prefix = `${encodeURIComponent(name)}=`;
  return document.cookie
    .split(";")
    .map((part) => part.trim())
    .find((part) => part.startsWith(prefix))
    ?.slice(prefix.length) ?? "";
}

export const apiClient = new ApiClient();
