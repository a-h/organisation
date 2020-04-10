package db

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestMarshalUserRecord(t *testing.T) {
	var r organisationGroupMemberRecord
	r.ID = "id"
	r.Range = "range"
	r.RecordType = organisationGroupMemberRecordName
	r.OrganisationID = "orgID"
	r.OrganisationName = "the org name"
	r.Groups = []string{"grpname"}
	r.Email = "email"
	r.FirstName = "fname"
	r.LastName = "lname"
	r.Phone = "12345"
	r.CreatedAt = time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	actual := string(data)
	expected := `{"id":"id","rng":"range","typ":"organisationGroupMember","v":0,"organisationId":"orgID","organisationName":"the org name","groups":["grpname"],"email":"email","firstName":"fname","lastName":"lname","phone":"12345","createdAt":"2020-01-01T00:00:00Z"}`

	if expected != actual {
		t.Error(cmp.Diff(expected, actual))
		t.Error(expected)
		t.Error(actual)
	}
}
