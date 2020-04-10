package db

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
)

const region = "eu-west-1"

func createLocalTable(t *testing.T) (name string) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		t.Fatalf("failed to create test db session: %v", err)
		return
	}
	name = uuid.New().String()
	client := dynamodb.New(sess)
	client.Endpoint = "http://localhost:8000"
	_, err = client.CreateTable(&dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("rng"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("rng"),
				KeyType:       aws.String("RANGE"),
			},
		},
		BillingMode: aws.String(dynamodb.BillingModePayPerRequest),
		TableName:   aws.String(name),
	})
	if err != nil {
		t.Fatalf("failed to create local table: %v", err)
	}
	return
}

func deleteLocalTable(t *testing.T, name string) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return
	}
	client := dynamodb.New(sess)
	client.Endpoint = "http://localhost:8000"
	_, err = client.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(name),
	})
	if err != nil {
		t.Fatalf("failed to delete table: %v", err)
	}
}

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
	u := User{
		ID:        "test@example.com",
		FirstName: "Sarah",
		LastName:  "Connor",
		CreatedAt: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		Phone:     "4476123456789",
	}
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
	expected := User{
		ID:        "test@example.com",
		FirstName: "Sarah",
		LastName:  "Connor",
		CreatedAt: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		Phone:     "4476123456789",
	}
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

func TestUserInviteIntegration(t *testing.T) {
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
	u := User{
		ID:        "test@example.com",
		FirstName: "Sarah",
		LastName:  "Connor",
		CreatedAt: time.Date(2020, time.January, 1, 0, 0, 0, 0, time.UTC),
		Phone:     "4476123456789",
	}
	err = s.Put(u)
	if err != nil {
		t.Errorf("failed to create user: %v", err)
	}

	// Invite user to three groups (A, B and C). Ignore A, Accept B, and Reject C.
	orgA := newOrganisation("orgA", "A")
	orgB := newOrganisation("orgB", "B")
	orgC := newOrganisation("orgC", "C")
	err = s.Invite(u, orgA, []string{"testGroup"})
	if err != nil {
		t.Errorf("failed to invite user to group A: %v", err)
	}
	err = s.Invite(u, orgB, []string{"testGroup"})
	if err != nil {
		t.Errorf("failed to invite user to group B: %v", err)
	}
	err = s.Invite(u, orgC, []string{"testGroup"})
	if err != nil {
		t.Errorf("failed to invite user to group C: %v", err)
	}

	err = s.AcceptInvite(u, orgB)
	if err != nil {
		t.Errorf("failed to accept invite to orgB: %v", err)
	}
	err = s.RejectInvite(u, orgC)
	if err != nil {
		t.Errorf("failed to reject invite to orgC: %v", err)
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
	if userDetails.Organisations[0].ID != "orgB" {
		t.Errorf("accepted orgB, but it's showing as %q", userDetails.Organisations[0].ID)
	}
	if diff := cmp.Diff(orgB, userDetails.Organisations[0]); diff != "" {
		t.Errorf("organisation fields not correct:\n%v", diff)
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
