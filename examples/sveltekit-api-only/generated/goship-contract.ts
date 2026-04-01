export type GoshipAPIError = {
  field?: string;
  message: string;
  code: string;
};

export type GoshipResponseEnvelope<T> = {
  data: T;
  errors?: GoshipAPIError[];
  meta?: Record<string, unknown>;
};

export type GoshipRouteContract = {
  method: string;
  path: string;
  operation_id: string;
  response_contract?: string;
  error_contracts?: string[];
};

export const goshipContractVersion = "api-only-same-origin-sveltekit-v1";

export const goshipBrowserContract = {
  authMode: "same-origin auth/session",
  csrfHeaderName: "X-CSRF-Token",
  cookieMode: "include",
} as const;

export const goshipRoutes: GoshipRouteContract[] = [
  {
    method: "GET",
    path: "/api/v1/status",
    operation_id: "get_api_v1_status",
    response_contract: "api.status.v1",
    error_contracts: [],
  },
];

export type GoshipFetchOptions = {
  method?: string;
  body?: BodyInit | null;
  headers?: Record<string, string>;
  csrfToken?: string;
};

export async function goshipFetch<T>(
  fetchImpl: typeof fetch,
  input: RequestInfo | URL,
  opts: GoshipFetchOptions = {},
): Promise<GoshipResponseEnvelope<T>> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    ...(opts.headers ?? {}),
  };

  if (opts.csrfToken && !headers["X-CSRF-Token"]) {
    headers["X-CSRF-Token"] = opts.csrfToken;
  }

  const res = await fetchImpl(input, {
    method: opts.method,
    body: opts.body,
    headers,
    credentials: "include",
  });

  return (await res.json()) as GoshipResponseEnvelope<T>;
}
