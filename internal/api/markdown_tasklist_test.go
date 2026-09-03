package api

import (
	"strings"
	"testing"
)

func TestMarkdownToADF_TaskList(t *testing.T) {
	adf := MarkdownToADF("- [ ] Open item\n- [x] Done item\n- [X] Also done")

	if len(adf.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(adf.Content))
	}

	list := adf.Content[0]
	if list.Type != "taskList" {
		t.Fatalf("expected taskList, got %q", list.Type)
	}
	if list.Attrs == nil || list.Attrs.LocalID == "" {
		t.Error("expected taskList to carry a localId")
	}
	if len(list.Content) != 3 {
		t.Fatalf("expected 3 task items, got %d", len(list.Content))
	}

	wantStates := []string{"TODO", "DONE", "DONE"}
	wantTexts := []string{"Open item", "Done item", "Also done"}
	seenIDs := map[string]bool{}
	for i, item := range list.Content {
		if item.Type != "taskItem" {
			t.Errorf("item %d: expected taskItem, got %q", i, item.Type)
		}
		if item.Attrs == nil {
			t.Fatalf("item %d: expected attrs", i)
		}
		if item.Attrs.State != wantStates[i] {
			t.Errorf("item %d: expected state %q, got %q", i, wantStates[i], item.Attrs.State)
		}
		if item.Attrs.LocalID == "" {
			t.Errorf("item %d: expected a localId", i)
		}
		if seenIDs[item.Attrs.LocalID] {
			t.Errorf("item %d: localId %q is not unique", i, item.Attrs.LocalID)
		}
		seenIDs[item.Attrs.LocalID] = true
		if len(item.Content) != 1 || item.Content[0].Type != "text" {
			t.Fatalf("item %d: expected inline text content, got %+v", i, item.Content)
		}
		if item.Content[0].Text != wantTexts[i] {
			t.Errorf("item %d: expected text %q, got %q", i, wantTexts[i], item.Content[0].Text)
		}
	}
}

func TestMarkdownToADF_TaskListInlineFormatting(t *testing.T) {
	adf := MarkdownToADF("- [ ] A1 – check **bold** and `code`")

	list := adf.Content[0]
	if list.Type != "taskList" {
		t.Fatalf("expected taskList, got %q", list.Type)
	}

	item := list.Content[0]
	if len(item.Content) < 3 {
		t.Fatalf("expected inline nodes with marks, got %+v", item.Content)
	}

	var hasStrong, hasCode bool
	for _, n := range item.Content {
		for _, m := range n.Marks {
			if m.Type == "strong" {
				hasStrong = true
			}
			if m.Type == "code" {
				hasCode = true
			}
		}
	}
	if !hasStrong || !hasCode {
		t.Errorf("expected strong and code marks, got strong=%v code=%v", hasStrong, hasCode)
	}
}

func TestMarkdownToADF_TaskListDoesNotSwallowBulletList(t *testing.T) {
	adf := MarkdownToADF("- plain bullet\n- [ ] task item")

	if len(adf.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(adf.Content))
	}
	if adf.Content[0].Type != "bulletList" {
		t.Errorf("expected first block bulletList, got %q", adf.Content[0].Type)
	}
	if adf.Content[1].Type != "taskList" {
		t.Errorf("expected second block taskList, got %q", adf.Content[1].Type)
	}
}

func TestMarkdownToADF_BulletListDoesNotSwallowTaskList(t *testing.T) {
	adf := MarkdownToADF("- [ ] task item\n- plain bullet")

	if len(adf.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(adf.Content))
	}
	if adf.Content[0].Type != "taskList" {
		t.Errorf("expected first block taskList, got %q", adf.Content[0].Type)
	}
	if adf.Content[1].Type != "bulletList" {
		t.Errorf("expected second block bulletList, got %q", adf.Content[1].Type)
	}
}

func TestADFToText_TaskList(t *testing.T) {
	doc := &ADF{
		Type:    "doc",
		Version: 1,
		Content: []ADFContent{
			{
				Type:  "taskList",
				Attrs: &ADFAttrs{LocalID: "list-1"},
				Content: []ADFContent{
					{
						Type:    "taskItem",
						Attrs:   &ADFAttrs{LocalID: "item-1", State: "TODO"},
						Content: []ADFContent{{Type: "text", Text: "Open item"}},
					},
					{
						Type:    "taskItem",
						Attrs:   &ADFAttrs{LocalID: "item-2", State: "DONE"},
						Content: []ADFContent{{Type: "text", Text: "Done item"}},
					},
				},
			},
		},
	}

	text := ADFToText(doc)

	if !strings.Contains(text, "[ ] Open item") {
		t.Errorf("expected open checkbox marker in output, got %q", text)
	}
	if !strings.Contains(text, "[x] Done item") {
		t.Errorf("expected done checkbox marker in output, got %q", text)
	}
	if strings.Contains(text, "Open itemDone item") {
		t.Errorf("task items were flattened together: %q", text)
	}
}

func TestTaskList_RoundTrip(t *testing.T) {
	src := "- [ ] First\n- [x] Second"

	text := ADFToText(MarkdownToADF(src))

	if !strings.Contains(text, "[ ] First") || !strings.Contains(text, "[x] Second") {
		t.Errorf("round-trip lost checkbox markers: %q", text)
	}

	// The rendered text must parse back into a task list with states intact.
	again := MarkdownToADF(text)
	if len(again.Content) != 1 || again.Content[0].Type != "taskList" {
		t.Fatalf("re-parse: expected a single taskList, got %+v", again.Content)
	}
	items := again.Content[0].Content
	if len(items) != 2 {
		t.Fatalf("re-parse: expected 2 items, got %d", len(items))
	}
	if items[0].Attrs.State != "TODO" || items[1].Attrs.State != "DONE" {
		t.Errorf("re-parse: states lost, got %q and %q", items[0].Attrs.State, items[1].Attrs.State)
	}
}
