package jolpiaf1

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the URI driver's pure string
// functions and the host wiring (Mint, Body, ResolveOn), which need no
// network. HTTP behaviour is covered in jolpiaf1_test.go.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "jolpiaf1" {
		t.Errorf("Scheme = %q, want jolpiaf1", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "jolpiaf1" {
		t.Errorf("Identity.Binary = %q, want jolpiaf1", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct{ in, typ, id string }{
		{"verstappen", "driver", "verstappen"},
		{"hamilton", "driver", "hamilton"},
		{"albon", "driver", "albon"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Error("Classify(\"\") should return an error")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("driver", "verstappen")
	want := BaseURL + "/ergast/f1/drivers/verstappen.json"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestLocateUnknownType(t *testing.T) {
	_, err := Domain{}.Locate("bogus", "foo")
	if err == nil {
		t.Error("Locate with unknown type should return an error")
	}
}

// TestHostWiring mounts the driver in a kit Host and checks the round trip:
// a Driver record mints to its URI, its body is readable, and a bare id
// resolves back to the same URI.
func TestHostWiring(t *testing.T) {
	h, err := kit.Open()
	if err != nil {
		t.Fatal(err)
	}

	d := &Driver{
		ID:          "verstappen",
		Name:        "Max Verstappen",
		Code:        "VER",
		Number:      "1",
		Nationality: "Dutch",
		Born:        "1997-09-30",
	}
	u, err := h.Mint(d)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	if want := "jolpiaf1://driver/verstappen"; u.String() != want {
		t.Errorf("Mint = %q, want %q", u.String(), want)
	}

	got, err := h.ResolveOn("jolpiaf1", "hamilton")
	if err != nil || got.String() != "jolpiaf1://driver/hamilton" {
		t.Errorf("ResolveOn = (%q, %v), want jolpiaf1://driver/hamilton", got.String(), err)
	}
}
