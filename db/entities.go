package db

import (
	"fmt"
	"strings"
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

func newUser(email, first, last, phone string, createdAt time.Time) User {
	return User{
		ID:        strings.ToLower(email),
		FirstName: first,
		LastName:  last,
		Phone:     phone,
		CreatedAt: createdAt,
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
			err = dynamodbattribute.UnmarshalMap(item, &user)
			if err != nil {
				err = fmt.Errorf("newUserDetailsFromRecords: failed to convert userRecord: %w", err)
				return
			}
			user.ID = *item["email"].S
			break
		case userOrgnisationRecordName:
			var uor userOrganisationRecord
			err = dynamodbattribute.UnmarshalMap(item, &uor)
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

func newOrganisationFromRecord(or organisationRecord) Organisation {
	return newOrganisation(or.OrganisationID, or.OrganisationName)
}

// An Organisation that can be joined.
type Organisation struct {
	ID   string
	Name string
}

func newOrganisationDetailsFromRecords(items []map[string]*dynamodb.AttributeValue) (org OrganisationDetails, err error) {
	serviceIDToService := make(map[string]Service)
	userIDToUser := make(map[string]User)
	userIDToGroups := make(map[string]*groupSet)

	for _, item := range items {
		recordType, ok := item["typ"]
		if !ok || recordType.S == nil {
			continue
		}
		switch *recordType.S {
		case organisationRecordName:
			var or organisationRecord
			err = dynamodbattribute.UnmarshalMap(item, &or)
			if err != nil {
				err = fmt.Errorf("newOrganisationDetailsFromRecords: failed to convert organisationRecord: %w", err)
				return
			}
			org.ID = or.OrganisationID
			org.Name = or.OrganisationName
		case organisationMemberRecordName:
			// Extract the member record details.
			var omr organisationMemberRecord
			err = dynamodbattribute.UnmarshalMap(item, &omr)
			if err != nil {
				err = fmt.Errorf("newOrganisationDetailsFromRecords: failed to convert organisationMember: %w", err)
				return
			}

			// Extract the user details.
			var ur userRecord
			err = dynamodbattribute.UnmarshalMap(item, &ur)
			if err != nil {
				err = fmt.Errorf("newOrganisationDetailsFromRecords: failed to convert organisationServiceGroupMemberRecord: %w", err)
				return
			}
			u := newUserFromRecord(ur)
			userIDToUser[u.ID] = u

			// Collate the user groups.
			userIDToGroups[omr.Email] = omr.Groups
		case organisationServiceRecordName:
			var osr organisationServiceRecord
			err = dynamodbattribute.UnmarshalMap(item, &osr)
			if err != nil {
				err = fmt.Errorf("newOrganisationDetailsFromRecords: failed to convert organisationServiceRecord: %w", err)
				return
			}
			service := serviceIDToService[osr.ServiceID]
			service.ID = osr.ServiceID
			service.Name = osr.ServiceName
			serviceIDToService[osr.ServiceID] = service
		}
	}
	// Now that all of the records have been read, populate the organisation groups and the services.
	for userID, groups := range userIDToGroups {
		user := userIDToUser[userID]
		for _, g := range groups.OrganisationGroups() {
			if org.Groups == nil {
				org.Groups = make(map[GroupName][]User)
			}
			org.Groups[GroupName(g)] = append(org.Groups[GroupName(g)], user)
		}

		for serviceID, groups := range groups.ServiceGroups() {
			service := serviceIDToService[serviceID]
			if service.Groups == nil {
				service.Groups = make(map[GroupName][]User)
			}
			for _, g := range groups {
				service.Groups[GroupName(g)] = append(service.Groups[GroupName(g)], user)
			}
			serviceIDToService[serviceID] = service
		}
	}
	// Now copy the services to the organisation.
	for serviceID := range serviceIDToService {
		org.Services = append(org.Services, serviceIDToService[serviceID])
	}
	return
}

func newOrganisationDetails(org Organisation, groups map[GroupName][]User, services []Service) OrganisationDetails {
	return OrganisationDetails{
		Organisation: org,
		Groups:       groups,
		Services:     services,
	}
}

// OrganisationDetails provides all the details of an Organisation.
type OrganisationDetails struct {
	Organisation
	Groups   map[GroupName][]User
	Services []Service
}

type GroupName string

// A Service is owned by an Organisation.
type Service struct {
	ID     string
	Name   string
	Groups map[GroupName][]User
}

const (
	GroupOwner  = "owner"
	GroupMember = "member"
)
