package assets

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

func TestAttributeValuesSkipsNullValues(t *testing.T) {
	values := []api.AssetAttributeValue{
		{DisplayValue: "Customer 145166", Value: json.RawMessage(`"145166"`)},
		{Value: json.RawMessage(`42`)},
		{},
	}

	want := []string{"Customer 145166", "42"}
	if got := attributeValues(values); !reflect.DeepEqual(got, want) {
		t.Fatalf("attributeValues() = %#v, want %#v", got, want)
	}
}

func TestAttributeValuesPreservesLargeNumber(t *testing.T) {
	values := []api.AssetAttributeValue{{Value: json.RawMessage(`9007199254740993`)}}
	if got, want := attributeValues(values), []string{"9007199254740993"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("attributeValues() = %#v, want %#v", got, want)
	}
}

func TestTerminalTextReplacesControlCharacters(t *testing.T) {
	if got, want := terminalText("Customer\n\x1b]52;c;payload\a"), "Customer  ]52;c;payload "; got != want {
		t.Fatalf("terminalText() = %q, want %q", got, want)
	}
}
