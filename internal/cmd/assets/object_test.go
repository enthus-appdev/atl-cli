package assets

import (
	"reflect"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

func TestAttributeValuesSkipsNullValues(t *testing.T) {
	values := []api.AssetAttributeValue{
		{DisplayValue: "Customer 145166", Value: "145166"},
		{Value: float64(42)},
		{},
	}

	want := []string{"Customer 145166", "42"}
	if got := attributeValues(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("attributeValues() = %#v, want %#v", got, want)
	}
}
