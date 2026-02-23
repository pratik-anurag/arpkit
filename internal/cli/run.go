package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/pratik-anurag/arpkit/internal/platform"
	"github.com/pratik-anurag/arpkit/internal/posture"
	"github.com/pratik-anurag/arpkit/internal/render"
	"github.com/pratik-anurag/arpkit/internal/topology"
	"github.com/pratik-anurag/arpkit/internal/util"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func Run(args []string, stdout io.Writer, stderr io.Writer, build BuildInfo) int {
	fs := flag.NewFlagSet("arpkit", flag.ContinueOnError)
	fs.SetOutput(stderr)

	format := fs.String("format", "pretty", "Output format: pretty|json|dot")
	jsonAlias := fs.Bool("json", false, "Alias for --format=json")
	colorMode := fs.String("color", "auto", "Color mode: auto|always|never")
	colorTheme := fs.String("color-theme", "auto", "Pill color theme: auto|distro|mono")
	profileName := fs.String("profile", "default", "Output profile: min|default|verbose")
	only := fs.String("only", "", "Only sections: summary,topology,cache,freq,microarch,power,posture,distance,llc,isolation,pcie,mem,memtop,notes")
	debug := fs.Bool("debug", false, "Enable debug output")
	showVersion := fs.Bool("version", false, "Print version")
	help := fs.Bool("help", false, "Show help")
	noDiagram := fs.Bool("no-diagram", false, "Disable topology diagram")
	noPill := fs.Bool("no-pill", false, "Disable chip pill header")
	compact := fs.Bool("compact", false, "Compact pretty output")
	wide := fs.Bool("wide", false, "Do not truncate topology diagram")
	unicode := fs.Bool("unicode", false, "Enable Unicode section icons and line styles")
	showMem := fs.Bool("mem", false, "Show memory distribution")
	showMicroarch := fs.Bool("microarch", false, "Show microarchitecture and ISA feature summary section")
	showDistance := fs.Bool("distance", false, "Show NUMA distance matrix section")
	showPCIe := fs.Bool("pcie", false, "Show PCIe NUMA affinity section")
	showPosture := fs.Bool("posture", false, "Show architecture posture section")

	fs.Usage = func() {
		fmt.Fprintln(stderr, "arpkit - Architecture Profiling Kit")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  arpkit [flags]")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Flags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if *help {
		fs.Usage()
		return 0
	}
	if *showVersion {
		fmt.Fprintf(stdout, "arpkit %s\n", build.Version)
		fmt.Fprintf(stdout, "commit: %s\n", build.Commit)
		fmt.Fprintf(stdout, "built: %s\n", build.Date)
		return 0
	}
	if *jsonAlias {
		*format = "json"
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "arpkit: unexpected positional arguments: %v\n", fs.Args())
		return 1
	}

	if !oneOf(*format, "pretty", "json", "dot") {
		fmt.Fprintf(stderr, "arpkit: invalid --format %q\n", *format)
		return 1
	}
	if !oneOf(*colorMode, "auto", "always", "never") {
		fmt.Fprintf(stderr, "arpkit: invalid --color %q\n", *colorMode)
		return 1
	}
	if !oneOf(*colorTheme, "auto", "distro", "mono") {
		fmt.Fprintf(stderr, "arpkit: invalid --color-theme %q\n", *colorTheme)
		return 1
	}
	if !oneOf(*profileName, "min", "default", "verbose") {
		fmt.Fprintf(stderr, "arpkit: invalid --profile %q\n", *profileName)
		return 1
	}

	onlySet := render.ParseOnly(*only)
	if err := validateOnly(onlySet); err != nil {
		fmt.Fprintf(stderr, "arpkit: %v\n", err)
		return 1
	}

	machine, err := platform.Collect(platform.Options{Debug: *debug})
	if err != nil {
		if errors.Is(err, platform.ErrUnsupported) {
			fmt.Fprintln(stderr, "arpkit: unsupported platform")
			return 2
		}
		fmt.Fprintf(stderr, "arpkit: %v\n", err)
		return 1
	}
	machine.Metadata.ToolVersion = build.Version
	if err := topology.Normalize(machine); err != nil {
		fmt.Fprintf(stderr, "arpkit: normalize profile: %v\n", err)
		return 1
	}
	machine.Posture = posture.Compute(machine)

	opts := render.Options{
		ColorMode:  *colorMode,
		ColorTheme: *colorTheme,
		IsTTY:      util.IsTTY(os.Stdout),
		Width:      render.DetectWidth(os.Stdout),
		Profile:    *profileName,
		Only:       onlySet,
		NoDiagram:  *noDiagram,
		NoPill:     *noPill,
		Compact:    *compact,
		Wide:       *wide,
		Unicode:    *unicode,
		Mem:        *showMem,
		Microarch:  *showMicroarch,
		Distance:   *showDistance,
		PCIe:       *showPCIe,
		Posture:    *showPosture,
		Debug:      *debug,
		Version:    build.Version,
	}

	var output string
	switch *format {
	case "json":
		output, err = render.RenderJSON(machine)
	case "dot":
		output, err = render.RenderDOT(machine)
	default:
		output, err = render.RenderPretty(machine, opts)
	}
	if err != nil {
		fmt.Fprintf(stderr, "arpkit: render output: %v\n", err)
		return 1
	}
	if _, err := io.WriteString(stdout, output); err != nil {
		fmt.Fprintf(stderr, "arpkit: write output: %v\n", err)
		return 1
	}

	if machine.Partial {
		return 2
	}
	return 0
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func validateOnly(only map[string]struct{}) error {
	if len(only) == 0 {
		return nil
	}
	allowed := map[string]struct{}{
		"summary":   {},
		"topology":  {},
		"cache":     {},
		"freq":      {},
		"microarch": {},
		"power":     {},
		"posture":   {},
		"distance":  {},
		"llc":       {},
		"isolation": {},
		"pcie":      {},
		"mem":       {},
		"memtop":    {},
		"notes":     {},
	}
	invalid := make([]string, 0)
	for section := range only {
		if _, ok := allowed[section]; !ok {
			invalid = append(invalid, section)
		}
	}
	if len(invalid) == 0 {
		return nil
	}
	sort.Strings(invalid)
	return fmt.Errorf("invalid --only value(s): %v", invalid)
}
