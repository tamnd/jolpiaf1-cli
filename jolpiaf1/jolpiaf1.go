// Package jolpiaf1 is the library behind the jolpiaf1 command line:
// the HTTP client, request shaping, and typed data models for the
// Jolpica F1 API (https://api.jolpi.ca/ergast/f1/), an open-source
// mirror of the Ergast Formula 1 API. No API key required.
package jolpiaf1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Host is the API host this client talks to.
const Host = "api.jolpi.ca"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// DefaultUserAgent identifies the client to the Jolpica API.
const DefaultUserAgent = "jolpiaf1-cli/0.1 (tamnd87@gmail.com)"

// Client talks to the Jolpica F1 API over HTTPS.
type Client struct {
	HTTP      *http.Client
	BaseURL   string
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	last time.Time
}

// NewClient returns a Client with sensible defaults: 15s timeout,
// 200ms minimum gap between requests, and 3 retries on transient errors.
func NewClient() *Client {
	return &Client{
		HTTP:      &http.Client{Timeout: 15 * time.Second},
		BaseURL:   BaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
	}
}

// Get fetches url and returns the response body. It paces and retries
// according to the client's settings.
func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, url)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", url, lastErr)
}

func (c *Client) do(ctx context.Context, url string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// ---- typed data models ----

// RaceResult is one driver's result in a race.
type RaceResult struct {
	Position    string `kit:"id" json:"position"`
	Driver      string `json:"driver"` // "Max Verstappen (VER)"
	Constructor string `json:"constructor"`
	Grid        string `json:"grid"`
	Points      string `json:"points"`
	Status      string `json:"status"`
	Time        string `json:"time"` // Results[].Time.time or ""
	Race        string `json:"race"` // raceName
}

// Standing is one row in a driver or constructor standings table.
type Standing struct {
	Position string `kit:"id" json:"position"`
	Name     string `json:"name"`
	Code     string `json:"code"` // driver code e.g. "VER" or constructorId
	Team     string `json:"team"`
	Points   string `json:"points"`
	Wins     string `json:"wins"`
}

// Race is one race weekend in a season schedule.
type Race struct {
	Round   string `kit:"id" json:"round"`
	Name    string `json:"name"`
	Circuit string `json:"circuit"`
	Country string `json:"country"`
	Date    string `json:"date"`
	Time    string `json:"time"`
}

// Driver holds information about a Formula 1 driver.
type Driver struct {
	ID          string `kit:"id" json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Number      string `json:"number"`
	Nationality string `json:"nationality"`
	Born        string `json:"born"`
}

// ---- API response shapes (unexported) ----

type mrData struct {
	RaceTable      *raceTableResp      `json:"RaceTable"`
	StandingsTable *standingsTableResp `json:"StandingsTable"`
	DriverTable    *driverTableResp    `json:"DriverTable"`
}

type apiResp struct {
	MRData mrData `json:"MRData"`
}

type raceTableResp struct {
	Races []raceResp `json:"Races"`
}

type raceResp struct {
	Season   string       `json:"season"`
	Round    string       `json:"round"`
	RaceName string       `json:"raceName"`
	Circuit  circuitResp  `json:"Circuit"`
	Date     string       `json:"date"`
	Time     string       `json:"time"`
	Results  []resultResp `json:"Results"`
}

type circuitResp struct {
	CircuitName string       `json:"circuitName"`
	Location    locationResp `json:"Location"`
}

type locationResp struct {
	Locality string `json:"locality"`
	Country  string `json:"country"`
}

type resultResp struct {
	Position    string          `json:"position"`
	Points      string          `json:"points"`
	Grid        string          `json:"grid"`
	Status      string          `json:"status"`
	Driver      driverResp      `json:"Driver"`
	Constructor constructorResp `json:"Constructor"`
	Time        *raceTime       `json:"Time"`
}

type raceTime struct {
	Time string `json:"time"`
}

type driverResp struct {
	DriverID        string `json:"driverId"`
	Code            string `json:"code"`
	GivenName       string `json:"givenName"`
	FamilyName      string `json:"familyName"`
	DateOfBirth     string `json:"dateOfBirth"`
	Nationality     string `json:"nationality"`
	PermanentNumber string `json:"permanentNumber"`
}

type constructorResp struct {
	ConstructorID string `json:"constructorId"`
	Name          string `json:"name"`
	Nationality   string `json:"nationality"`
}

type standingsTableResp struct {
	StandingsLists []standingsListResp `json:"StandingsLists"`
}

type standingsListResp struct {
	DriverStandings      []driverStandingResp      `json:"DriverStandings"`
	ConstructorStandings []constructorStandingResp `json:"ConstructorStandings"`
}

type driverStandingResp struct {
	Position     string            `json:"position"`
	Points       string            `json:"points"`
	Wins         string            `json:"wins"`
	Driver       driverResp        `json:"Driver"`
	Constructors []constructorResp `json:"Constructors"`
}

type constructorStandingResp struct {
	Position    string          `json:"position"`
	Points      string          `json:"points"`
	Wins        string          `json:"wins"`
	Constructor constructorResp `json:"Constructor"`
}

type driverTableResp struct {
	Drivers []driverResp `json:"Drivers"`
}

// ---- client methods ----

// Results fetches race results for the given year and round.
func (c *Client) Results(ctx context.Context, year, round string, limit int) ([]*RaceResult, error) {
	url := fmt.Sprintf("%s/ergast/f1/%s/%s/results.json?limit=%d", c.BaseURL, year, round, limit)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if resp.MRData.RaceTable == nil || len(resp.MRData.RaceTable.Races) == 0 {
		return nil, nil
	}
	race := resp.MRData.RaceTable.Races[0]
	out := make([]*RaceResult, 0, len(race.Results))
	for _, r := range race.Results {
		t := ""
		if r.Time != nil {
			t = r.Time.Time
		}
		out = append(out, &RaceResult{
			Position:    r.Position,
			Driver:      r.Driver.GivenName + " " + r.Driver.FamilyName + " (" + r.Driver.Code + ")",
			Constructor: r.Constructor.Name,
			Grid:        r.Grid,
			Points:      r.Points,
			Status:      r.Status,
			Time:        t,
			Race:        race.RaceName,
		})
	}
	return out, nil
}

// DriverStandings fetches the driver championship standings for the given year.
func (c *Client) DriverStandings(ctx context.Context, year string) ([]*Standing, error) {
	url := fmt.Sprintf("%s/ergast/f1/%s/driverStandings.json", c.BaseURL, year)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if resp.MRData.StandingsTable == nil || len(resp.MRData.StandingsTable.StandingsLists) == 0 {
		return nil, nil
	}
	list := resp.MRData.StandingsTable.StandingsLists[0]
	out := make([]*Standing, 0, len(list.DriverStandings))
	for _, s := range list.DriverStandings {
		team := ""
		if len(s.Constructors) > 0 {
			team = s.Constructors[0].Name
		}
		out = append(out, &Standing{
			Position: s.Position,
			Name:     s.Driver.GivenName + " " + s.Driver.FamilyName,
			Code:     s.Driver.Code,
			Team:     team,
			Points:   s.Points,
			Wins:     s.Wins,
		})
	}
	return out, nil
}

// ConstructorStandings fetches the constructor championship standings for the given year.
func (c *Client) ConstructorStandings(ctx context.Context, year string) ([]*Standing, error) {
	url := fmt.Sprintf("%s/ergast/f1/%s/constructorStandings.json", c.BaseURL, year)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if resp.MRData.StandingsTable == nil || len(resp.MRData.StandingsTable.StandingsLists) == 0 {
		return nil, nil
	}
	list := resp.MRData.StandingsTable.StandingsLists[0]
	out := make([]*Standing, 0, len(list.ConstructorStandings))
	for _, s := range list.ConstructorStandings {
		out = append(out, &Standing{
			Position: s.Position,
			Name:     s.Constructor.Name,
			Code:     s.Constructor.ConstructorID,
			Team:     "",
			Points:   s.Points,
			Wins:     s.Wins,
		})
	}
	return out, nil
}

// Races fetches the race schedule for the given season year.
func (c *Client) Races(ctx context.Context, year string) ([]*Race, error) {
	url := fmt.Sprintf("%s/ergast/f1/%s.json", c.BaseURL, year)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if resp.MRData.RaceTable == nil {
		return nil, nil
	}
	out := make([]*Race, 0, len(resp.MRData.RaceTable.Races))
	for _, r := range resp.MRData.RaceTable.Races {
		out = append(out, &Race{
			Round:   r.Round,
			Name:    r.RaceName,
			Circuit: r.Circuit.CircuitName,
			Country: r.Circuit.Location.Country,
			Date:    r.Date,
			Time:    r.Time,
		})
	}
	return out, nil
}

// GetDriver fetches information about a driver by their Ergast driver ID.
func (c *Client) GetDriver(ctx context.Context, driverID string) (*Driver, error) {
	url := fmt.Sprintf("%s/ergast/f1/drivers/%s.json", c.BaseURL, driverID)
	body, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	var resp apiResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if resp.MRData.DriverTable == nil || len(resp.MRData.DriverTable.Drivers) == 0 {
		return nil, fmt.Errorf("driver not found: %s", driverID)
	}
	d := resp.MRData.DriverTable.Drivers[0]
	return &Driver{
		ID:          d.DriverID,
		Name:        d.GivenName + " " + d.FamilyName,
		Code:        d.Code,
		Number:      d.PermanentNumber,
		Nationality: d.Nationality,
		Born:        d.DateOfBirth,
	}, nil
}
