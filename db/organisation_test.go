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
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
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
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
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
		GroupOwner: {owner},
	}
	var services []Service
	expected := newOrganisationDetails(org, groups, services)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}

func TestOrganisationGroupIntegration(t *testing.T) {
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

	// Add the owner to some groups.
	err = s.AddUserToOrganisationGroups(organisationID, owner, "hipsters", "gin_fans", "tricycle_riders")
	if err != nil {
		t.Errorf("failed to add owner to Organisation group: %v", err)
	}

	// Add a user that doesn't exist to some groups.
	other := newUser("other@example.com", "Other F", "Other L", "1567", createdAt)
	err = s.AddUserToOrganisationGroups(organisationID, other, "hipsters", "tricycle_riders")
	if err != nil {
		t.Errorf("failed to add other to Organisation group: %v", err)
	}

	// Remove the owner from the "gin_fans" group.
	err = s.RemoveUserFromOrganisationGroups(organisationID, "test@example.com", "tricycle_riders", "gin_fans")
	if err != nil {
		t.Errorf("failed to remove owner from groups: %v", err)
	}

	// Now get it back.
	org := newOrganisation(organisationID, "Organisation Name")
	groups := map[GroupName][]User{
		GroupOwner:                   {owner},
		GroupName("hipsters"):        {other, owner},
		GroupName("tricycle_riders"): {other},
	}
	var services []Service
	expected := newOrganisationDetails(org, groups, services)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if hipsterCount := len(actual.Groups[GroupName("hipsters")]); hipsterCount != 2 {
		t.Errorf("expected two users in hipster group, got %v", hipsterCount)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}

func TestServiceGroupsIntegration(t *testing.T) {
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

	// Create a service.
	serviceID, err := s.CreateService(organisationID, "old_service_name")
	if err != nil {
		t.Errorf("failed to create a service: %v", err)
	}

	// Rename it.
	err = s.PutService(organisationID, serviceID, "new_service_name")
	if err != nil {
		t.Errorf("failed to rename service: %v", err)
	}

	// Add the owner to service groups.
	err = s.AddUserToServiceGroups(organisationID, owner, serviceID, "service_group_1", "service_group_2", "service_group_3")
	if err != nil {
		t.Errorf("failed to add user to service: %v", err)
	}

	// Delete the owner from one of the groups.
	err = s.RemoveUserFromServiceGroups(organisationID, owner.ID, serviceID, "service_group_2", "non-existent-group")
	if err != nil {
		t.Errorf("failed to remove user from service: %v", err)
	}

	// Now get it back.
	org := newOrganisation(organisationID, "Organisation Name")
	groups := map[GroupName][]User{
		GroupOwner: {owner},
	}
	services := []Service{
		{
			ID:   serviceID,
			Name: "new_service_name",
			Groups: map[GroupName][]User{
				GroupName("service_group_1"): {owner},
				GroupName("service_group_3"): {owner},
			},
		},
	}
	expected := newOrganisationDetails(org, groups, services)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}

func TestServiceGroupDeleteIntegration(t *testing.T) {
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

	// Create a service.
	serviceID, err := s.CreateService(organisationID, "old_service_name")
	if err != nil {
		t.Errorf("failed to create a service: %v", err)
	}

	// Create a new service and delete it.
	err = s.DeleteService(organisationID, serviceID)
	if err != nil {
		t.Errorf("failed to delete service: %v", err)
	}

	// Check that it's been deleted.
	org := newOrganisation(organisationID, "Organisation Name")
	groups := map[GroupName][]User{
		GroupOwner: {owner},
	}
	expected := newOrganisationDetails(org, groups, nil)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}

func TestOrganisationUpdateUserIntegration(t *testing.T) {
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

	// Add another user to organisation groups.
	other := newUser("other@example.com", "Other F", "Other L", "1567", createdAt)
	err = s.AddUserToOrganisationGroups(organisationID, other, "hipsters")
	if err != nil {
		t.Errorf("failed to add other to Organisation group: %v", err)
	}

	// Update their details.
	other.FirstName = "updated_firstname"
	other.LastName = "updated_lastname"
	other.Phone = "updated_phone"
	err = s.UpdateUserDetails(organisationID, other.ID, other.FirstName, other.LastName, other.Phone)
	if err != nil {
		t.Errorf("failed to update user: %v", err)
	}

	// Check that they're updated.
	org := newOrganisation(organisationID, "Organisation Name")
	groups := map[GroupName][]User{
		GroupOwner:            {owner},
		GroupName("hipsters"): {other},
	}
	expected := newOrganisationDetails(org, groups, nil)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}

func TestOrganisationDeleteUserIntegration(t *testing.T) {
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

	// Add another user to organisation groups.
	other := newUser("other@example.com", "Other F", "Other L", "1567", createdAt)
	err = s.AddUserToOrganisationGroups(organisationID, other, "hipsters")
	if err != nil {
		t.Errorf("failed to add other to Organisation group: %v", err)
	}

	// Delete the user.
	err = s.RemoveUser(organisationID, other.ID)
	if err != nil {
		t.Errorf("failed to delete user: %v", err)
	}

	// Check that they're not in the list.
	org := newOrganisation(organisationID, "Organisation Name")
	groups := map[GroupName][]User{
		GroupOwner: {owner},
	}
	expected := newOrganisationDetails(org, groups, nil)
	actual, err := s.GetDetails(organisationID)
	if err != nil {
		t.Errorf("failed to get organisation: %v", err)
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Error(diff)
	}
}
