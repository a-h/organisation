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
	// defer deleteLocalTable(t, name)
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
