package geo

import (
	"math"
	"testing"
)

func TestLocationValidation(t *testing.T) {
	for _, p := range []LocationProps{
		{Latitude: 91},
		{Longitude: -181},
		{Latitude: math.NaN()},
		{Longitude: math.Inf(1)},
	} {
		if _, err := NewLocation(p); err == nil {
			t.Fatalf("accepted %v", p)
		}
	}

	a, err := NewLocation(LocationProps{})
	if err != nil {
		t.Fatal(err)
	}

	if a.Validate() != nil {
		t.Fatal("zero coordinates are valid")
	}

	if (Location{}).Validate() == nil {
		t.Fatal("missing location is invalid")
	}

	b, _ := NewLocation(LocationProps{Longitude: 1})
	if math.Abs(a.DistanceKM(b)-111.195) > 0.01 {
		t.Fatal("incorrect distance")
	}
}
