package db

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// record default fields.
type record struct {
	ID         string `json:"id"`
	Range      string `json:"rng"`
	RecordType string `json:"typ"`
	Version    int    `json:"v"`
}

func idAndRng(id, rng string) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"id":  &dynamodb.AttributeValue{S: aws.String(id)},
		"rng": &dynamodb.AttributeValue{S: aws.String(rng)},
	}
}

// organisation record.
const organisationRecordName = "organisation"

func newOrganisationRecordHashKey(organisationID string) string {
	return organisationRecordName + "/" + organisationID
}

func newOrganisationRecordRangeKey() string {
	return organisationRecordName
}

func newOrganisationRecord(org Organisation) organisationRecord {
	var record organisationRecord
	record.ID = newOrganisationRecordHashKey(org.ID)
	record.Range = newOrganisationRecordRangeKey()
	record.RecordType = organisationRecordName
	record.Version = 0
	record.OrganisationID = org.ID
	record.OrganisationName = org.Name
	return record
}

type organisationRecord struct {
	record
	organisationRecordFields
}

type organisationRecordFields struct {
	OrganisationID   string `json:"organisationId"`
	OrganisationName string `json:"organisationName"`
}

// organisation group member record.
const organisationGroupMemberRecordName = "organisationGroupMember"

func newOrganisationGroupMemberRecordHashKey(organisationID string) string {
	return newOrganisationRecordHashKey(organisationID)
}

func newOrganisationGroupMemberRecordRangeKey(emailAddress string) string {
	return organisationGroupMemberRecordName + "/" + emailAddress
}

func newOrganisationGroupMemberRecord(org Organisation, groups []string, u User, now time.Time) organisationGroupMemberRecord {
	var record organisationGroupMemberRecord
	record.ID = newOrganisationGroupMemberRecordHashKey(org.ID)
	record.Range = newOrganisationGroupMemberRecordRangeKey(u.ID)
	record.RecordType = organisationGroupMemberRecordName
	record.Version = 0
	record.Groups = groups
	record.Email = u.ID
	record.FirstName = u.FirstName
	record.LastName = u.LastName
	record.Phone = u.Phone
	record.CreatedAt = now
	record.OrganisationID = org.ID
	record.OrganisationName = org.Name
	return record
}

type organisationGroupMemberRecord struct {
	record
	organisationRecordFields
	Groups []string `json:"groups"`
	userRecordFields
}

// organisation service record.
const organisationServiceRecordName = "organisationService"

func newOrganisationServiceRecordHashKey(organisationID string) string {
	return newOrganisationRecordHashKey(organisationID)
}

func newOrganisationServiceRecordRangeKey(serviceID string) string {
	return organisationServiceRecordName + "/" + serviceID
}

type organisationServiceRecord struct {
	record
	ServiceName string `json:"serviceName"`
}

// organisation service group member record.
const organisationServiceGroupMemberRecordName = "organisationServiceGroupMember"

func newOrganisationServiceGroupMemberRecordHashKey(organisationID string) string {
	return newOrganisationRecordHashKey(organisationID)
}

func newOrganisationServiceGroupMemberRecordRangeKey(serviceID, groupName, emailAddress string) string {
	return organisationServiceGroupMemberRecordName + "/" + serviceID + "/" + groupName + "/" + emailAddress
}

type organisationServiceGroupMemberRecord struct {
	record
	userRecordFields
}
