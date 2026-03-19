package generators

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type MakeIslandOptions struct {
	Name string
}

type MakeIslandDeps struct {
	Out io.Writer
	Err io.Writer
	Cwd string
}

func RunMakeIsland(args []string, d MakeIslandDeps) int {
	opts, err := ParseMakeIslandArgs(args)
	if err != nil {
		fmt.Fprintf(d.Err, "invalid make:island arguments: %v\n", err)
		return 1
	}

	cwd := d.Cwd
	if strings.TrimSpace(cwd) == "" {
		var wdErr error
		cwd, wdErr = os.Getwd()
		if wdErr != nil {
			fmt.Fprintf(d.Err, "failed to resolve working directory: %v\n", wdErr)
			return 1
		}
	}

	tokens := splitWords(opts.Name)
	if len(tokens) == 0 {
		fmt.Fprintln(d.Err, "invalid make:island arguments: usage: ship make:island <Name>")
		return 1
	}

	basePascal := toPascalFromParts(tokens)
	baseSnake := strings.Join(tokens, "_")
	baseKebab := strings.Join(tokens, "-")
	componentName := basePascal + "Island"
	componentKebab := baseKebab + "-island"

	islandRel := filepath.ToSlash(filepath.Join("frontend", "islands", basePascal+".js"))
	templRel := filepath.ToSlash(filepath.Join("app", "views", "web", "components", baseSnake+"_island.templ"))
	islandPath := filepath.Join(cwd, filepath.FromSlash(islandRel))
	templPath := filepath.Join(cwd, filepath.FromSlash(templRel))

	if _, err := os.Stat(islandPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing island file: %s\n", islandPath)
		return 1
	}
	if _, err := os.Stat(templPath); err == nil {
		fmt.Fprintf(d.Err, "refusing to overwrite existing island mount component: %s\n", templPath)
		return 1
	}

	if err := os.MkdirAll(filepath.Dir(islandPath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create islands directory: %v\n", err)
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(templPath), 0o755); err != nil {
		fmt.Fprintf(d.Err, "failed to create components directory: %v\n", err)
		return 1
	}

	if err := os.WriteFile(islandPath, []byte(renderIslandFile(basePascal, componentKebab)), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write island file: %v\n", err)
		return 1
	}
	if err := os.WriteFile(templPath, []byte(renderIslandTemplFile(componentName, componentKebab, basePascal)), 0o644); err != nil {
		fmt.Fprintf(d.Err, "failed to write island mount component: %v\n", err)
		return 1
	}

	writeGeneratorReport(
		d.Out,
		"island",
		false,
		[]string{islandRel, templRel},
		nil,
		[]generatorPreview{
			{
				Title: "Templ usage snippet",
				Body: fmt.Sprintf(`@components.%s(map[string]any{
	"label": "%s",
	"initialCount": 0,
})`, componentName, basePascal),
			},
		},
		[]string{
			fmt.Sprintf("ship templ generate --file %s", templRel),
			"make build-js",
			fmt.Sprintf("render @components.%s(...) from the page or component that should mount this island", componentName),
		},
	)
	return 0
}

func ParseMakeIslandArgs(args []string) (MakeIslandOptions, error) {
	opts := MakeIslandOptions{}
	if len(args) == 0 {
		return opts, errors.New("usage: ship make:island <Name>")
	}
	opts.Name = strings.TrimSpace(args[0])
	if opts.Name == "" || strings.HasPrefix(opts.Name, "-") {
		return opts, errors.New("usage: ship make:island <Name>")
	}
	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			return opts, fmt.Errorf("unknown option: %s", args[i])
		}
		return opts, fmt.Errorf("unexpected argument: %s", args[i])
	}
	return opts, nil
}

func renderIslandFile(basePascal, componentKebab string) string {
	return fmt.Sprintf(`function toInitialCount(value) {
  const parsed = Number(value);
  if (Number.isFinite(parsed)) {
    return parsed;
  }
  return 0;
}

function resolveLabel(props = {}, fallback) {
  return typeof props.label === "string" && props.label.length > 0 ? props.label : fallback;
}

export function mount(el, props = {}) {
  const state = {
    count: toInitialCount(props.initialCount),
    label: resolveLabel(props, %q),
  };

  const container = document.createElement("section");
  container.dataset.component = %q;
  container.className = "rounded-xl border border-base-300 bg-base-100 p-4 space-y-3";

  const title = document.createElement("h3");
  title.className = "text-lg font-semibold";

  const value = document.createElement("p");
  value.className = "text-3xl font-bold leading-none";
  value.setAttribute("data-slot", "count");

  const note = document.createElement("p");
  note.className = "text-xs font-medium uppercase tracking-wide opacity-70";
  note.textContent = "Generated island scaffold";

  const controls = document.createElement("div");
  controls.className = "flex gap-2";

  const decrement = document.createElement("button");
  decrement.type = "button";
  decrement.className = "btn btn-sm";
  decrement.textContent = "-1";

  const increment = document.createElement("button");
  increment.type = "button";
  increment.className = "btn btn-sm";
  increment.textContent = "+1";

  const render = () => {
    title.textContent = state.label;
    value.textContent = String(state.count);
  };

  decrement.addEventListener("click", () => {
    state.count -= 1;
    render();
  });

  increment.addEventListener("click", () => {
    state.count += 1;
    render();
  });

  controls.appendChild(decrement);
  controls.appendChild(increment);
  container.appendChild(title);
  container.appendChild(value);
  container.appendChild(note);
  container.appendChild(controls);

  el.replaceChildren(container);
  render();
}
`, basePascal, componentKebab)
}

func renderIslandTemplFile(componentName, componentKebab, islandName string) string {
	return fmt.Sprintf(`package components

import "github.com/a-h/templ"

// Renders: mount target and fallback shell for the %s frontend island
// Route(s): embedded in web layouts/pages
templ %s(props map[string]any) {
	<div
		data-component=%q
		data-island=%q
		data-props={ templ.JSONString(props) }
		class="contents"
	>
		<section class="rounded-xl border border-dashed border-base-300 bg-base-100 p-4 space-y-2">
			<h3 class="text-lg font-semibold">%s</h3>
			<p data-slot="count" class="text-3xl font-bold leading-none">0</p>
			<p class="text-xs font-medium uppercase tracking-wide opacity-70">Generated island fallback</p>
		</section>
	</div>
}
`, islandName, componentName, componentKebab, islandName, islandName)
}
