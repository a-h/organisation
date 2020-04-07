package db

import (
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

func newOrganisationGroupMemberRecordRangeKey(groupName, emailAddress string) string {
	return organisationGroupMemberRecordName + "/" + groupName + "/" + emailAddress
}

type organisationGroupMemberRecord struct {
	record
	organisationRecordFields
	GroupName string `json:"groupName"`
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
