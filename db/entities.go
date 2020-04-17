package db

import (
	"strings"
	"time"
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
