export type ApiClientOptions = {
  baseUrl?: string;
};

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly payload?: unknown,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

type JsonBody = Record<string, unknown> | unknown[];
type JsonRequestInit = Omit<RequestInit, "body"> & { body?: JsonBody };

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

  delete<T>(path: string) {
    return this.request<T>(path, { method: "DELETE" });
  }

  private async request<T>(path: string, init: JsonRequestInit) {
    const response = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers: {
        Accept: "application/json",
        "Content-Type": "application/json",
        ...init.headers,
      },
      credentials: "same-origin",
      body: init.body === undefined ? undefined : JSON.stringify(init.body),
    });

    const payload = await readJson(response);
    if (!response.ok) {
      throw new ApiError(response.statusText || "Request failed", response.status, payload);
    }

    return payload as T;
  }
}

async function readJson(response: Response) {
  const text = await response.text();
  return text.length > 0 ? JSON.parse(text) : null;
}

export const apiClient = new ApiClient();
