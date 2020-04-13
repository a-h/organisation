package db

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestOrganisationPutIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewOrganisationStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	// Create an organisation.
	createdAt := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	owner := newUser("test@example.com", "First", "Last", "447901234567", createdAt)
	organisationID, err := s.Create(owner, "Organisation Name")
	if err != nil {
		t.Errorf("failed to create organisation: %v", err)
	}

	// Update it.
	expected := newOrganisation(organisationID, "New Organisation Name")
	err = s.Put(expected)
	if err != nil {
		t.Errorf("failed to put organisation: %v", err)
	}

	// Get the updated one.
	actual, err := s.Get(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if !cmp.Equal(expected, actual) {
		t.Errorf(cmp.Diff(expected, actual))
	}
}

func TestOrganisationGetIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewOrganisationStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	// Create an organisation.
	createdAt := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	owner := newUser("test@example.com", "First", "Last", "447901234567", createdAt)
	organisationID, err := s.Create(owner, "Organisation Name")
	if err != nil {
		t.Errorf("failed to create organisation: %v", err)
	}

	// Now get it back.
	expected := newOrganisation(organisationID, "Organisation Name")
	actual, err := s.Get(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if !cmp.Equal(expected, actual) {
		t.Errorf(cmp.Diff(expected, actual))
	}
}

func TestOrganisationGetDetailsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	name := createLocalTable(t)
	defer deleteLocalTable(t, name)
	s, err := NewOrganisationStore(region, name)
	s.Client.Endpoint = "http://localhost:8000"
	if err != nil {
		t.Errorf("failed to create store: %v", err)
	}
	// Create an organisation.
	createdAt := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	owner := newUser("test@example.com", "First", "Last", "447901234567", createdAt)
	organisationID, err := s.Create(owner, "Organisation Name")
	if err != nil {
		t.Errorf("failed to create organisation: %v", err)
	}

	// Now get it back.
	org := newOrganisation(organisationID, "Organisation Name")
	groups := map[GroupName][]User{
		GroupOwner: []User{owner},
	}
	var services []Service
	expected := newOrganisationDetails(org, groups, services)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if !cmp.Equal(expected, actual) {
		t.Errorf(cmp.Diff(expected, actual))
	}
}
