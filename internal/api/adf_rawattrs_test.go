package api

import (
	"encoding/json"
	"strings"
	"testing"
)

// Shape taken verbatim from production comment SD-25403/555819.
const realMediaNode = `{"type":"mediaSingle","attrs":{"width":1066,"widthType":"pixel","localId":"cf1a68f326ac","layout":"align-start"},"content":[{"type":"media","attrs":{"type":"file","id":"0df3c2f3-721e-43e7-b481-a5cd54ac45e8","alt":"shot.png","collection":"","localId":"d1dcfffd3144","height":126,"width":2256}}]}`

func TestADFContent_RoundTripPreservesMediaAttrs(t *testing.T) {
	var node ADFContent
	if err := json.Unmarshal([]byte(realMediaNode), &node); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)

	// The empty-but-required key Jira rejects the document without.
	if !strings.Contains(got, `"collection":""`) {
		t.Errorf(`expected "collection":"" to survive, got %s`, got)
	}
	for _, want := range []string{"cf1a68f326ac", "d1dcfffd3144", "pixel", "align-start"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q to survive round-trip, got %s", want, got)
		}
	}
}

func TestADFContent_RoundTripIsSemanticallyIdentical(t *testing.T) {
	var node ADFContent
	if err := json.Unmarshal([]byte(realMediaNode), &node); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var before, after any
	if err := json.Unmarshal([]byte(realMediaNode), &before); err != nil {
		t.Fatalf("unmarshal before: %v", err)
	}
	if err := json.Unmarshal(out, &after); err != nil {
		t.Fatalf("unmarshal after: %v", err)
	}

	b, _ := json.Marshal(before)
	a, _ := json.Marshal(after)
	if string(a) != string(b) {
		t.Errorf("round-trip changed the document:\n before: %s\n after:  %s", b, a)
	}
}

func TestADFContent_ProducerBuiltNodeStillMarshals(t *testing.T) {
	// A node built in Go has no RawAttrs; the typed Attrs must still be emitted.
	node := ADFContent{
		Type:    "heading",
		Attrs:   &ADFAttrs{Level: 2},
		Content: []ADFContent{{Type: "text", Text: "hi"}},
	}
	out, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"level":2`) {
		t.Errorf("expected typed attrs to be emitted, got %s", out)
	}
}

func TestADFContent_NodeWithoutAttrsEmitsNoAttrsKey(t *testing.T) {
	var node ADFContent
	if err := json.Unmarshal([]byte(`{"type":"paragraph","content":[{"type":"text","text":"x"}]}`), &node); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "attrs") {
		t.Errorf("expected no attrs key, got %s", out)
	}
}

func TestADFMark_RoundTripPreservesAttrs(t *testing.T) {
	var mark ADFMark
	if err := json.Unmarshal([]byte(`{"type":"textColor","attrs":{"color":"#ff0000"}}`), &mark); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(mark)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(out), `"color":"#ff0000"`) {
		t.Errorf("expected unmodelled mark attr to survive, got %s", out)
	}
}

func TestADFContent_ExplicitNullAttrsEmitsNoAttrsKey(t *testing.T) {
	var node ADFContent
	if err := json.Unmarshal([]byte(`{"type":"paragraph","attrs":null}`), &node); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(node)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "attrs") {
		t.Errorf(`expected no attrs key for explicit null, got %s`, out)
	}
}

func TestADFMark_ExplicitNullAttrsEmitsNoAttrsKey(t *testing.T) {
	var mark ADFMark
	if err := json.Unmarshal([]byte(`{"type":"em","attrs":null}`), &mark); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(mark)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "attrs") {
		t.Errorf(`expected no attrs key for explicit null, got %s`, out)
	}
}
