package db

import (
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

type User struct {
	// ID is the user's email address.
	ID        string
	FirstName string
	LastName  string
	Phone     string
	CreatedAt time.Time
}

type UserDetails struct {
	User
	Organisations []Organisation
	Invitations   []Invitation
}

type Invitation struct {
	Organisation Organisation
	InvitedAt    time.Time
	AcceptedAt   *time.Time
}

type Organisation struct {
	ID   string
	Name string
}

type OrganisationDetails struct {
	Organisation
	Groups   []Group
	Services []Service
}

type Service struct {
	ID     string
	Name   string
	Groups []Group
}

type Group struct {
	Name  string
	Users []User
}
