package jolpiaf1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ---- helpers ----

func newTestClient(srv *httptest.Server) *Client {
	c := NewClient()
	c.BaseURL = srv.URL
	c.Rate = 0
	c.Retries = 0
	return c
}

// ---- HTTP layer ----

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0

	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	c.Retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

// ---- Results ----

func TestResults(t *testing.T) {
	payload := map[string]any{
		"MRData": map[string]any{
			"RaceTable": map[string]any{
				"Races": []any{
					map[string]any{
						"season":   "2024",
						"round":    "1",
						"raceName": "Bahrain Grand Prix",
						"Results": []any{
							map[string]any{
								"position": "1",
								"points":   "25",
								"grid":     "1",
								"status":   "Finished",
								"Driver": map[string]any{
									"driverId":   "verstappen",
									"code":       "VER",
									"givenName":  "Max",
									"familyName": "Verstappen",
								},
								"Constructor": map[string]any{
									"constructorId": "red_bull",
									"name":          "Red Bull",
								},
								"Time": map[string]any{
									"time": "1:32:23.313",
								},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	results, err := c.Results(context.Background(), "2024", "1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Position != "1" {
		t.Errorf("Position = %q, want 1", r.Position)
	}
	if r.Driver != "Max Verstappen (VER)" {
		t.Errorf("Driver = %q, want Max Verstappen (VER)", r.Driver)
	}
	if r.Constructor != "Red Bull" {
		t.Errorf("Constructor = %q, want Red Bull", r.Constructor)
	}
	if r.Time != "1:32:23.313" {
		t.Errorf("Time = %q, want 1:32:23.313", r.Time)
	}
	if r.Race != "Bahrain Grand Prix" {
		t.Errorf("Race = %q, want Bahrain Grand Prix", r.Race)
	}
}

// ---- Driver Standings ----

func TestDriverStandings(t *testing.T) {
	payload := map[string]any{
		"MRData": map[string]any{
			"StandingsTable": map[string]any{
				"StandingsLists": []any{
					map[string]any{
						"DriverStandings": []any{
							map[string]any{
								"position": "1",
								"points":   "437",
								"wins":     "9",
								"Driver": map[string]any{
									"code":       "VER",
									"givenName":  "Max",
									"familyName": "Verstappen",
								},
								"Constructors": []any{
									map[string]any{"name": "Red Bull"},
								},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	standings, err := c.DriverStandings(context.Background(), "2024")
	if err != nil {
		t.Fatal(err)
	}
	if len(standings) != 1 {
		t.Fatalf("got %d standings, want 1", len(standings))
	}
	s := standings[0]
	if s.Position != "1" {
		t.Errorf("Position = %q, want 1", s.Position)
	}
	if s.Name != "Max Verstappen" {
		t.Errorf("Name = %q, want Max Verstappen", s.Name)
	}
	if s.Code != "VER" {
		t.Errorf("Code = %q, want VER", s.Code)
	}
	if s.Team != "Red Bull" {
		t.Errorf("Team = %q, want Red Bull", s.Team)
	}
	if s.Points != "437" {
		t.Errorf("Points = %q, want 437", s.Points)
	}
}

// ---- Constructor Standings ----

func TestConstructorStandings(t *testing.T) {
	payload := map[string]any{
		"MRData": map[string]any{
			"StandingsTable": map[string]any{
				"StandingsLists": []any{
					map[string]any{
						"ConstructorStandings": []any{
							map[string]any{
								"position": "1",
								"points":   "860",
								"wins":     "14",
								"Constructor": map[string]any{
									"constructorId": "mclaren",
									"name":          "McLaren",
								},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	standings, err := c.ConstructorStandings(context.Background(), "2024")
	if err != nil {
		t.Fatal(err)
	}
	if len(standings) != 1 {
		t.Fatalf("got %d standings, want 1", len(standings))
	}
	s := standings[0]
	if s.Position != "1" {
		t.Errorf("Position = %q, want 1", s.Position)
	}
	if s.Name != "McLaren" {
		t.Errorf("Name = %q, want McLaren", s.Name)
	}
	if s.Code != "mclaren" {
		t.Errorf("Code = %q, want mclaren", s.Code)
	}
	if s.Team != "" {
		t.Errorf("Team = %q, want empty for constructor", s.Team)
	}
}

// ---- Races ----

func TestRaces(t *testing.T) {
	payload := map[string]any{
		"MRData": map[string]any{
			"RaceTable": map[string]any{
				"Races": []any{
					map[string]any{
						"round":    "1",
						"raceName": "Bahrain Grand Prix",
						"Circuit": map[string]any{
							"circuitName": "Bahrain International Circuit",
							"Location": map[string]any{
								"locality": "Sakhir",
								"country":  "Bahrain",
							},
						},
						"date": "2024-03-02",
						"time": "15:00:00Z",
					},
					map[string]any{
						"round":    "2",
						"raceName": "Saudi Arabian Grand Prix",
						"Circuit": map[string]any{
							"circuitName": "Jeddah Corniche Circuit",
							"Location": map[string]any{
								"locality": "Jeddah",
								"country":  "Saudi Arabia",
							},
						},
						"date": "2024-03-09",
						"time": "17:00:00Z",
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	races, err := c.Races(context.Background(), "2024")
	if err != nil {
		t.Fatal(err)
	}
	if len(races) != 2 {
		t.Fatalf("got %d races, want 2", len(races))
	}
	r := races[0]
	if r.Round != "1" {
		t.Errorf("Round = %q, want 1", r.Round)
	}
	if r.Name != "Bahrain Grand Prix" {
		t.Errorf("Name = %q, want Bahrain Grand Prix", r.Name)
	}
	if r.Country != "Bahrain" {
		t.Errorf("Country = %q, want Bahrain", r.Country)
	}
	if r.Date != "2024-03-02" {
		t.Errorf("Date = %q, want 2024-03-02", r.Date)
	}
}

// ---- GetDriver ----

func TestGetDriver(t *testing.T) {
	payload := map[string]any{
		"MRData": map[string]any{
			"DriverTable": map[string]any{
				"Drivers": []any{
					map[string]any{
						"driverId":        "verstappen",
						"permanentNumber": "1",
						"code":            "VER",
						"givenName":       "Max",
						"familyName":      "Verstappen",
						"dateOfBirth":     "1997-09-30",
						"nationality":     "Dutch",
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	d, err := c.GetDriver(context.Background(), "verstappen")
	if err != nil {
		t.Fatal(err)
	}
	if d.ID != "verstappen" {
		t.Errorf("ID = %q, want verstappen", d.ID)
	}
	if d.Name != "Max Verstappen" {
		t.Errorf("Name = %q, want Max Verstappen", d.Name)
	}
	if d.Code != "VER" {
		t.Errorf("Code = %q, want VER", d.Code)
	}
	if d.Number != "1" {
		t.Errorf("Number = %q, want 1", d.Number)
	}
	if d.Nationality != "Dutch" {
		t.Errorf("Nationality = %q, want Dutch", d.Nationality)
	}
	if d.Born != "1997-09-30" {
		t.Errorf("Born = %q, want 1997-09-30", d.Born)
	}
}

func TestGetDriverNotFound(t *testing.T) {
	payload := map[string]any{
		"MRData": map[string]any{
			"DriverTable": map[string]any{
				"Drivers": []any{},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.GetDriver(context.Background(), "unknown_driver")
	if err == nil {
		t.Error("GetDriver with unknown driver should return an error")
	}
}
