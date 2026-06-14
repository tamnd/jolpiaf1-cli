package jolpiaf1

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes jolpiaf1 as a kit Domain so a multi-domain host (ant)
// can enable it with a single blank import:
//
//	import _ "github.com/tamnd/jolpiaf1-cli/jolpiaf1"
//
// The same Domain builds the standalone jolpiaf1 binary (see cli.NewApp),
// so the binary and any host share one source of truth.
func init() { kit.Register(Domain{}) }

// Domain is the jolpiaf1 driver. It carries no state; the per-run client
// is built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched
// against, and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "jolpiaf1",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "jolpiaf1",
			Short:  "Formula 1 data from the Jolpica F1 API.",
			Long: `jolpiaf1 reads public Formula 1 data from the Jolpica F1 API
(https://api.jolpi.ca/ergast/f1/), an open-source mirror of the
Ergast API. No API key or sign-up required.

Race results, driver and constructor standings, season schedules,
and driver profiles — all shaped into clean records that pipe
into the rest of your tools.`,
			Site: Host,
			Repo: "https://github.com/tamnd/jolpiaf1-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// results: race results for a given year + round
	kit.Handle(app, kit.OpMeta{
		Name:    "results",
		Group:   "read",
		Single:  false,
		Summary: "Get race results for a specific grand prix",
		Args: []kit.Arg{
			{Name: "year", Help: "season year, e.g. 2024"},
			{Name: "round", Help: "round number, e.g. 1"},
		},
	}, getResults)

	// standings: driver or constructor standings
	kit.Handle(app, kit.OpMeta{
		Name:    "standings",
		Group:   "read",
		Single:  false,
		Summary: "Get driver or constructor championship standings",
	}, getStandings)

	// races: season race schedule
	kit.Handle(app, kit.OpMeta{
		Name:    "races",
		Group:   "read",
		Single:  false,
		Summary: "List all races in a season",
	}, getRaces)

	// driver: driver profile
	kit.Handle(app, kit.OpMeta{
		Name:    "driver",
		Group:   "read",
		Single:  true,
		Summary: "Get information about a driver",
		URIType: "driver",
		Resolver: true,
		Args: []kit.Arg{
			{Name: "driver_id", Help: "driver ID, e.g. verstappen, hamilton, albon"},
		},
	}, getDriver)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// ---- input structs ----

type resultsIn struct {
	Year   string  `kit:"arg" help:"season year, e.g. 2024"`
	Round  string  `kit:"arg" help:"round number, e.g. 1"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

type standingsIn struct {
	Year   string  `kit:"flag" help:"season year" default:"2024"`
	Type   string  `kit:"flag" help:"standings type: driver or constructor" default:"driver"`
	Client *Client `kit:"inject"`
}

type racesIn struct {
	Year   string  `kit:"flag" help:"season year" default:"2024"`
	Client *Client `kit:"inject"`
}

type driverIn struct {
	DriverID string  `kit:"arg" help:"driver ID, e.g. verstappen"`
	Client   *Client `kit:"inject"`
}

// ---- handlers ----

func getResults(ctx context.Context, in resultsIn, emit func(*RaceResult) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	results, err := in.Client.Results(ctx, in.Year, in.Round, limit)
	if err != nil {
		return mapErr(err)
	}
	for _, r := range results {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func getStandings(ctx context.Context, in standingsIn, emit func(*Standing) error) error {
	typ := strings.ToLower(in.Type)
	if typ == "" {
		typ = "driver"
	}
	year := in.Year
	if year == "" {
		year = "2024"
	}

	var standings []*Standing
	var err error
	switch typ {
	case "constructor", "constructors":
		standings, err = in.Client.ConstructorStandings(ctx, year)
	default:
		standings, err = in.Client.DriverStandings(ctx, year)
	}
	if err != nil {
		return mapErr(err)
	}
	for _, s := range standings {
		if err := emit(s); err != nil {
			return err
		}
	}
	return nil
}

func getRaces(ctx context.Context, in racesIn, emit func(*Race) error) error {
	year := in.Year
	if year == "" {
		year = "2024"
	}
	races, err := in.Client.Races(ctx, year)
	if err != nil {
		return mapErr(err)
	}
	for _, r := range races {
		if err := emit(r); err != nil {
			return err
		}
	}
	return nil
}

func getDriver(ctx context.Context, in driverIn, emit func(*Driver) error) error {
	d, err := in.Client.GetDriver(ctx, in.DriverID)
	if err != nil {
		return mapErr(err)
	}
	return emit(d)
}

// ---- Resolver: pure string functions, network-free ----

// Classify turns any accepted input into the canonical (type, id).
func (Domain) Classify(input string) (uriType, id string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errs.Usage("empty jolpiaf1 reference")
	}
	return "driver", input, nil
}

// Locate is the inverse: the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "driver":
		return BaseURL + "/ergast/f1/drivers/" + id + ".json", nil
	default:
		return "", errs.Usage("jolpiaf1 has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind that carries
// the right exit code.
func mapErr(err error) error {
	return err
}
