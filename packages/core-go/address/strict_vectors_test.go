package address

import (
	"encoding/json"
	"errors"
	"os"
	"testing"
)

// strkeyVectorFile mirrors the shared spec/vectors/*.json files consumed by
// DecodeStrKeyStrict.
type strkeyVectorFile struct {
	Cases []struct {
		Description string `json:"description"`
		Input       struct {
			Address        string `json:"address"`
			AllowLowercase bool   `json:"allowLowercase"`
		} `json:"input"`
		Expected struct {
			OK      bool   `json:"ok"`
			Error   string `json:"error"`
			Kind    string `json:"kind"`
			Address string `json:"address"`
		} `json:"expected"`
	} `json:"cases"`
}

func loadStrkeyVectors(t *testing.T, name string) strkeyVectorFile {
	t.Helper()
	raw, err := os.ReadFile("../../../spec/vectors/" + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var f strkeyVectorFile
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	if len(f.Cases) == 0 {
		t.Fatalf("%s has no cases", name)
	}
	return f
}

// TestTruncatedAddressVectors runs spec/vectors/truncated-addresses.json.
func TestTruncatedAddressVectors(t *testing.T) {
	f := loadStrkeyVectors(t, "truncated-addresses.json")

	seen := map[int]bool{}
	for _, tc := range f.Cases {
		tc := tc
		if !tc.Expected.OK && len(tc.Input.Address) < 69 {
			seen[len(tc.Input.Address)] = true
		}
		t.Run(tc.Description, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("DecodeStrKeyStrict(%q) panicked: %v", tc.Input.Address, r)
				}
			}()

			kind, canonical, _, err := DecodeStrKeyStrict(tc.Input.Address)
			if tc.Expected.OK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(kind) != tc.Expected.Kind || canonical != tc.Expected.Address {
					t.Fatalf("got (%s, %s), want (%s, %s)", kind, canonical, tc.Expected.Kind, tc.Expected.Address)
				}
				return
			}

			var re RoutingError
			if !errors.As(err, &re) {
				t.Fatalf("error %v is not a RoutingError", err)
			}
			if string(re.Code) != tc.Expected.Error {
				t.Fatalf("code = %s, want %s", re.Code, tc.Expected.Error)
			}
		})
	}

	for n := 10; n <= 68; n++ {
		if !seen[n] {
			t.Errorf("missing truncated vector of length %d", n)
		}
	}
}
