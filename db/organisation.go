package db

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
	"github.com/google/uuid"
)

// NewOrganisationStore creates a new OrganisationStore.
func NewOrganisationStore(region, tableName string) (us OrganisationStore, err error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return
	}
	us.Client = dynamodb.New(sess)
	us.TableName = aws.String(tableName)
	us.Now = func() time.Time {
		return time.Now().UTC()
	}
	return
}

// OrganisationStore stores Organisation records in DynamoDB.
type OrganisationStore struct {
	Client    *dynamodb.DynamoDB
	TableName *string
	Now       func() time.Time
}

// Create a new organisation.
func (store OrganisationStore) Create(owner User, name string) (id string, err error) {
	// Create the Organisation.
	id = uuid.New().String()
	now := store.Now()
	org := newOrganisation(id, name)
	or := newOrganisationRecord(org)
	orItem, err := dynamodbattribute.MarshalMap(or)
	if err != nil {
		return
	}
	notOverwrite := expression.And(expression.AttributeNotExists(expression.Name("id")),
		expression.AttributeNotExists(expression.Name("rng")))
	notOverwriteExpr, err := expression.NewBuilder().WithCondition(notOverwrite).Build()
	if err != nil {
		return
	}
	putNewOrganisation := &dynamodb.Put{
		TableName:           store.TableName,
		Item:                orItem,
		ConditionExpression: notOverwriteExpr.KeyCondition(),
	}

	// Assign ownership.
	ogmr := newOrganisationGroupMemberRecord(org, []string{GroupOwner}, owner)
	ogmrItem, err := dynamodbattribute.MarshalMap(ogmr)
	if err != nil {
		return
	}
	putOrganisationGroupMember := &dynamodb.Put{
		TableName: store.TableName,
		Item:      ogmrItem,
	}

	// Include the user side of the ownership relationship.
	userOrganisationRecord := newUserOrganisationRecord(owner, org, now, &now)
	userOrganisationItem, err := dynamodbattribute.ConvertToMap(userOrganisationRecord)
	if err != nil {
		err = fmt.Errorf("userStore.Invite: failed to convert userOrganisationRecord: %w", err)
		return
	}
	putUserOrganisation := &dynamodb.Put{
		TableName: store.TableName,
		Item:      userOrganisationItem,
	}

	_, err = store.Client.TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			&dynamodb.TransactWriteItem{Put: putNewOrganisation},
			&dynamodb.TransactWriteItem{Put: putOrganisationGroupMember},
			&dynamodb.TransactWriteItem{Put: putUserOrganisation},
		},
	})
	return
}

// Put an Organisation.
func (store OrganisationStore) Put(org Organisation) error {
	ur := newOrganisationRecord(org)
	item, err := dynamodbattribute.MarshalMap(ur)
	if err != nil {
		return err
	}
	_, err = store.Client.PutItem(&dynamodb.PutItemInput{
		TableName: store.TableName,
		Item:      item,
	})
	return err
}

// Get an Organisation.
func (store OrganisationStore) Get(id string) (org Organisation, err error) {
	gio, err := store.Client.GetItem(&dynamodb.GetItemInput{
		TableName:      store.TableName,
		ConsistentRead: aws.Bool(true),
		Key:            idAndRng(newOrganisationRecordHashKey(id), newOrganisationRecordRangeKey()),
	})
	if err != nil {
		return
	}
	var record organisationRecord
	err = dynamodbattribute.UnmarshalMap(gio.Item, &record)
	org = newOrganisationFromRecord(record)
	return
}

// GetDetails retrieves all details of an Organisation.
func (store OrganisationStore) GetDetails(id string) (org OrganisationDetails, err error) {
	q := expression.Key("id").Equal(expression.Value(newOrganisationRecordHashKey(id)))
	expr, err := expression.NewBuilder().
		WithKeyCondition(q).
		Build()
	if err != nil {
		err = fmt.Errorf("organisationStore.GetDetails: failed to build query: %v", err)
		return
	}

	qi := &dynamodb.QueryInput{
		TableName:                 store.TableName,
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeValues: expr.Values(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ConsistentRead:            aws.Bool(true),
	}

	var items []map[string]*dynamodb.AttributeValue
	page := func(page *dynamodb.QueryOutput, lastPage bool) bool {
		items = append(items, page.Items...)
		return true
	}
	err = store.Client.QueryPages(qi, page)
	if err != nil {
		err = fmt.Errorf("organisationStore.GetDetails: failed to query pages: %v", err)
		return
	}

	org, err = newOrganisationDetailsFromRecords(items)
	if err != nil {
		err = fmt.Errorf("organisationStore.GetDetails: failed to create OrganisationDetails: %w", err)
	}
	return
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

func newOrganisationGroupMemberRecord(org Organisation, groups []string, u User) organisationGroupMemberRecord {
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
	record.CreatedAt = u.CreatedAt
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
	ServiceID   string `json:"serviceId"`
	ServiceName string `json:"serviceName"`
}

// organisation service group record.
const organisationServiceGroupRecordName = "organisationServiceGroup"

func newOrganisationServiceGroupRecordHashKey(organisationID string) string {
	return newOrganisationRecordHashKey(organisationID)
}

func newOrganisationServiceGroupRecordRangeKey(serviceID, group string) string {
	return organisationServiceGroupRecordName + "/" + serviceID + "/" + group
}

type organisationServiceGroupMemberRecord struct {
	record
	ServiceID string   `json:"serviceId"`
	Group     string   `json:"group"`
	UserIDs   []string `json:"userIds"`
}
