package sm

import (
	"reflect"
	"testing"

	"github.com/enthus-appdev/atl-cli/internal/api"
)

func TestRequestTypeRows_IncludesIssueType(t *testing.T) {
	rows := requestTypeRows([]*api.RequestType{
		{ID: "3", Name: "Änderungs-Ticket", IssueTypeID: "10035", Description: "Change request"},
	})

	want := [][]string{{"3", "Änderungs-Ticket", "10035", "Change request"}}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("rows = %v, want %v", rows, want)
	}
}
