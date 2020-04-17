package db

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestUserPutIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewUserStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	u := newUser("test@example.com", "Sarah", "Connor", "4476123456789",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	err = s.Put(u)
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}
}

func TestUserGetIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewUserStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	expected := newUser("test@example.com", "Sarah", "Connor", "4476123456789",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	err = s.Put(expected)
	if err != nil {
		t.Errorf("failed to put user: %v", err)
	}
	actual, err := s.Get("test@example.com")
	if err != nil {
		t.Errorf("failed to get user: %v", err)
	}
	if !cmp.Equal(expected, actual) {
		t.Errorf(cmp.Diff(expected, actual))
	}
}

func TestUserInviteIgnore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewUserStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	u := newUser("test@example.com", "Sarah", "Connor", "4476123456789",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	err = s.Put(u)
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}

	orgA := newOrganisation("orgA", "A")
	err = s.Invite(u, orgA, []string{"testGroup"}, nil)
	if err != nil {
		t.Errorf("failed to invite user to group A: %v", err)
	}

	// Get the details and ensure that this is reflected.
	userDetails, err := s.GetDetails("test@example.com")
	if err != nil {
		t.Errorf("failed to get user details: %v", err)
	}

	if diff := cmp.Diff(u, userDetails.User); diff != "" {
		t.Errorf("failed to match user:\n%v", diff)
	}
	if len(userDetails.Organisations) != 0 {
		t.Errorf("expected 0 organisations, got %d", len(userDetails.Organisations))
	}
	if len(userDetails.Invitations) != 1 {
		t.Errorf("expected 1 invitation, got %d", len(userDetails.Invitations))
	}
	if userDetails.Invitations[0].Organisation.ID != "orgA" {
		t.Errorf("the invite from orgA has not been accepted or rejected, but got %q", userDetails.Invitations[0].Organisation.ID)
	}
	if diff := cmp.Diff(orgA, userDetails.Invitations[0].Organisation); diff != "" {
		t.Errorf("invitation organisation fields not correct:\n%v", diff)
	}
}

func TestUserInviteAccept(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewUserStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	u := newUser("test@example.com", "Sarah", "Connor", "4476123456789",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	err = s.Put(u)
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}

	// Invite user to three groups (A, B and C). Ignore A, Accept B, and Reject C.
	orgA := newOrganisation("orgA", "A")
	err = s.Invite(u, orgA, []string{"testGroup"}, nil)
	if err != nil {
		t.Errorf("failed to invite user to group A: %v", err)
	}

	err = s.AcceptInvite(u, orgA)
	if err != nil {
		t.Errorf("failed to accept invite to orgB: %v", err)
	}

	// Get the details and ensure that this is reflected.
	userDetails, err := s.GetDetails("test@example.com")
	if err != nil {
		t.Errorf("failed to get user details: %v", err)
	}

	if diff := cmp.Diff(u, userDetails.User); diff != "" {
		t.Errorf("failed to match user:\n%v", diff)
	}
	if len(userDetails.Organisations) != 1 {
		t.Errorf("expected 1 organisation, got %d", len(userDetails.Organisations))
	}
	if userDetails.Organisations[0].ID != "orgA" {
		t.Errorf("accepted orgA, but it's showing as %q", userDetails.Organisations[0].ID)
	}
	if diff := cmp.Diff(orgA, userDetails.Organisations[0]); diff != "" {
		t.Errorf("organisation fields not correct:\n%v", diff)
	}
	if len(userDetails.Invitations) != 0 {
		t.Errorf("expected 0 invitations, got %d", len(userDetails.Invitations))
	}
}

func TestUserInviteReject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewUserStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	u := newUser("test@example.com", "Sarah", "Connor", "4476123456789",
		time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC))
	err = s.Put(u)
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}

	orgA := newOrganisation("orgA", "A")
	err = s.Invite(u, orgA, []string{"testGroup"}, nil)
	if err != nil {
		t.Errorf("failed to invite user to group A: %v", err)
	}
	err = s.RejectInvite(u, orgA)
	if err != nil {
		t.Errorf("failed to reject invite to orgC: %v", err)
	}

	userDetails, err := s.GetDetails("test@example.com")
	if err != nil {
		t.Errorf("failed to get user details: %v", err)
	}

	if diff := cmp.Diff(u, userDetails.User); diff != "" {
		t.Errorf("failed to match user:\n%v", diff)
	}
	if len(userDetails.Organisations) != 0 {
		t.Errorf("expected no organisations, got %d", len(userDetails.Organisations))
	}
	if len(userDetails.Invitations) != 0 {
		t.Errorf("expected no invitations, got %d", len(userDetails.Invitations))
	}
}
