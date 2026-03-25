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
