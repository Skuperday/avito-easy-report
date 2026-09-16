package service

import (
	models "avito-easy-report/internal/struct"
	"encoding/json"
	"testing"
)

func TestGroupedListingCount(t *testing.T) {
	offers := []models.Offer{
		{Employee: "Анна", Object: "Офис", ListingNumber: "same", Shows: 12},
		{Employee: "Анна", Object: "Офис", ListingNumber: "same"},
		{Employee: "Борис", Object: "Склад", Shows: 3},
		{Employee: "", Object: ""},
	}
	for _, group := range []string{"employee", "object"} {
		t.Run(group, func(t *testing.T) {
			rows := GetResultStats(GetGroupedStats(offers, group))
			if len(rows) != 2 {
				t.Fatalf("got %d groups, want 2", len(rows))
			}
			for _, row := range rows {
				raw, _ := json.Marshal(row)
				var fields map[string]any
				if err := json.Unmarshal(raw, &fields); err != nil {
					t.Fatal(err)
				}
				want := float64(1)
				if row.Key == "Анна" || row.Key == "Офис" {
					want = 2
					if row.Shows != 12 {
						t.Fatal("shows changed")
					}
				}
				if fields["listingCount"] != want {
					t.Errorf("%s listingCount = %v, want %v", row.Key, fields["listingCount"], want)
				}
			}
		})
	}
	if got := GetResultStats(GetGroupedStats(nil, "employee")); len(got) != 0 {
		t.Fatal(got)
	}
}
