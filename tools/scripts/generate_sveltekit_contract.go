package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type routeRow struct {
	Method           string   `json:"method"`
	Path             string   `json:"path"`
	OperationID      string   `json:"operation_id"`
	ResponseContract string   `json:"response_contract"`
	ErrorContracts   []string `json:"error_contracts"`
}

func main() {
	outputPath := flag.String("output", filepath.FromSlash("examples/sveltekit-api-only/generated/goship-contract.ts"), "output file")
	jsonOutputPath := flag.String("json-output", filepath.FromSlash("examples/sveltekit-api-only/generated/goship-contract.json"), "json manifest output file")
	flag.Parse()

	rows, err := loadRoutes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate_sveltekit_contract: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*outputPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "generate_sveltekit_contract: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*outputPath, []byte(renderContract(rows)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "generate_sveltekit_contract: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*jsonOutputPath, []byte(renderContractJSON(rows)), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "generate_sveltekit_contract: %v\n", err)
		os.Exit(1)
	}
}

func loadRoutes() ([]routeRow, error) {
	cmd := exec.Command("go", "run", "./tools/cli/ship/cmd/ship", "routes", "--json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run routes --json: %w\n%s", err, out)
	}
	var rows []routeRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("decode routes output: %w", err)
	}
	filtered := make([]routeRow, 0, len(rows))
	for _, row := range rows {
		if strings.HasPrefix(strings.TrimSpace(row.Path), "/api/") {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

func renderContract(routes []routeRow) string {
	var b bytes.Buffer
	b.WriteString("export type GoshipAPIError = {\n")
	b.WriteString("  field?: string;\n")
	b.WriteString("  message: string;\n")
	b.WriteString("  code: string;\n")
	b.WriteString("};\n\n")
	b.WriteString("export type GoshipResponseEnvelope<T> = {\n")
	b.WriteString("  data: T;\n")
	b.WriteString("  errors?: GoshipAPIError[];\n")
	b.WriteString("  meta?: Record<string, unknown>;\n")
	b.WriteString("};\n\n")
	b.WriteString("export type GoshipRouteContract = {\n")
	b.WriteString("  method: string;\n")
	b.WriteString("  path: string;\n")
	b.WriteString("  operation_id: string;\n")
	b.WriteString("  response_contract?: string;\n")
	b.WriteString("  error_contracts?: string[];\n")
	b.WriteString("};\n\n")
	b.WriteString("export const goshipContractVersion = \"api-only-same-origin-sveltekit-v1\";\n\n")
	b.WriteString("export const goshipBrowserContract = {\n")
	b.WriteString("  authMode: \"same-origin auth/session\",\n")
	b.WriteString("  csrfHeaderName: \"X-CSRF-Token\",\n")
	b.WriteString("  cookieMode: \"include\",\n")
	b.WriteString("} as const;\n\n")
	b.WriteString("export const goshipRoutes: GoshipRouteContract[] = [\n")
	for _, route := range routes {
		fmt.Fprintf(&b, "  {\n    method: %q,\n    path: %q,\n    operation_id: %q,\n", route.Method, route.Path, route.OperationID)
		if route.ResponseContract != "" {
			fmt.Fprintf(&b, "    response_contract: %q,\n", route.ResponseContract)
		}
		if len(route.ErrorContracts) > 0 {
			raw, _ := json.Marshal(route.ErrorContracts)
			fmt.Fprintf(&b, "    error_contracts: %s,\n", string(raw))
		} else {
			b.WriteString("    error_contracts: [],\n")
		}
		b.WriteString("  },\n")
	}
	b.WriteString("];\n\n")
	b.WriteString("export type GoshipFetchOptions = {\n")
	b.WriteString("  method?: string;\n")
	b.WriteString("  body?: BodyInit | null;\n")
	b.WriteString("  headers?: Record<string, string>;\n")
	b.WriteString("  csrfToken?: string;\n")
	b.WriteString("};\n\n")
	b.WriteString("export async function goshipFetch<T>(\n")
	b.WriteString("  fetchImpl: typeof fetch,\n")
	b.WriteString("  input: RequestInfo | URL,\n")
	b.WriteString("  opts: GoshipFetchOptions = {},\n")
	b.WriteString("): Promise<GoshipResponseEnvelope<T>> {\n")
	b.WriteString("  const headers: Record<string, string> = {\n")
	b.WriteString("    Accept: \"application/json\",\n")
	b.WriteString("    ...(opts.headers ?? {}),\n")
	b.WriteString("  };\n\n")
	b.WriteString("  if (opts.csrfToken && !headers[\"X-CSRF-Token\"]) {\n")
	b.WriteString("    headers[\"X-CSRF-Token\"] = opts.csrfToken;\n")
	b.WriteString("  }\n\n")
	b.WriteString("  const res = await fetchImpl(input, {\n")
	b.WriteString("    method: opts.method,\n")
	b.WriteString("    body: opts.body,\n")
	b.WriteString("    headers,\n")
	b.WriteString("    credentials: \"include\",\n")
	b.WriteString("  });\n\n")
	b.WriteString("  return (await res.json()) as GoshipResponseEnvelope<T>;\n")
	b.WriteString("}\n")
	return b.String()
}

func renderContractJSON(routes []routeRow) string {
	payload := map[string]any{
		"contract_version": "api-only-same-origin-sveltekit-v1",
		"browser_contract": map[string]any{
			"authMode":       "same-origin auth/session",
			"csrfHeaderName": "X-CSRF-Token",
			"cookieMode":     "include",
		},
		"routes": routes,
	}
	body, _ := json.MarshalIndent(payload, "", "  ")
	return string(body) + "\n"
}
