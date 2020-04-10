package db

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func newUserFromRecord(ur userRecord) User {
	return User{
		ID:        ur.Email,
		FirstName: ur.FirstName,
		LastName:  ur.LastName,
		Phone:     ur.Phone,
		CreatedAt: ur.CreatedAt,
	}
}

// User of the system.
type User struct {
	// ID is the user's email address.
	ID        string
	FirstName string
	LastName  string
	Phone     string
	CreatedAt time.Time
}

func newUserDetailsFromRecords(items []map[string]*dynamodb.AttributeValue) (user UserDetails, err error) {
	for _, item := range items {
		recordType, ok := item["typ"]
		if !ok || recordType.S == nil {
			continue
		}
		switch *recordType.S {
		case userRecordName:
			err = dynamodbattribute.ConvertFromMap(item, &user)
			if err != nil {
				err = fmt.Errorf("newUserDetailsFromRecords: failed to convert userRecord: %w", err)
				return
			}
			user.ID = *item["email"].S
			break
		case userOrgnisationRecordName:
			var uor userOrganisationRecord
			err = dynamodbattribute.ConvertFromMap(item, &uor)
			if err != nil {
				err = fmt.Errorf("newUserDetailsFromRecords: failed to convert userOrganisationRecord: %w", err)
				return
			}
			if uor.AcceptedAt == nil {
				user.Invitations = append(user.Invitations, newInvitationFromRecord(uor))
				continue
			}
			user.Organisations = append(user.Organisations, newOrganisation(uor.OrganisationID, uor.OrganisationName))
			break
		}
	}
	return
}

// UserDetails provides all the details of a User.
type UserDetails struct {
	User
	Organisations []Organisation
	Invitations   []Invitation
}

func newInvitationFromRecord(uor userOrganisationRecord) Invitation {
	org := newOrganisation(uor.OrganisationID, uor.OrganisationName)
	return newInvitation(org, uor.InvitedAt, uor.AcceptedAt)
}

func newInvitation(org Organisation, invitedAt time.Time, acceptedAt *time.Time) Invitation {
	return Invitation{
		Organisation: org,
		InvitedAt:    invitedAt,
		AcceptedAt:   acceptedAt,
	}
}

// An Invitation for a user to join an Organisation.
type Invitation struct {
	Organisation Organisation
	InvitedAt    time.Time
	AcceptedAt   *time.Time
}

func newOrganisation(id, name string) Organisation {
	return Organisation{ID: id, Name: name}
}

// An Organisation that can be joined.
type Organisation struct {
	ID   string
	Name string
}

// OrganisationDetails provides all the details of an Organisation.
type OrganisationDetails struct {
	Organisation
	Groups   []Group
	Services []Service
}

// A Service is owned by an Organisation.
type Service struct {
	ID     string
	Name   string
	Groups []Group
}

// A Group contains Users.
type Group struct {
	Name  string
	Users []User
}
