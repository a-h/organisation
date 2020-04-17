package db

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/expression"
)

// NewUserStore creates a new UserStore.
func NewUserStore(region, tableName string) (us UserStore, err error) {
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

// UserStore stores User records in DynamoDB.
type UserStore struct {
	Client    *dynamodb.DynamoDB
	TableName *string
	Now       func() time.Time
}

// Put a User.
func (store UserStore) Put(user User) error {
	ur := newUserRecord(user)
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

// Get a User.
func (store UserStore) Get(id string) (user User, err error) {
	gio, err := store.Client.GetItem(&dynamodb.GetItemInput{
		TableName:      store.TableName,
		ConsistentRead: aws.Bool(true),
		Key:            idAndRng(newUserRecordHashKey(id), newUserRecordRangeKey()),
	})
	if err != nil {
		return
	}
	var record userRecord
	err = dynamodbattribute.UnmarshalMap(gio.Item, &record)
	user = newUserFromRecord(record)
	return
}

// GetDetails gets the full details of a User.
func (store UserStore) GetDetails(id string) (user UserDetails, err error) {
	q := expression.Key("id").Equal(expression.Value(newUserRecordHashKey(id)))
	expr, err := expression.NewBuilder().
		WithKeyCondition(q).
		Build()
	if err != nil {
		err = fmt.Errorf("userStore.GetDetails: failed to build query: %v", err)
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
		err = fmt.Errorf("userStore.GetDetails: failed to query pages: %v", err)
		return
	}

	user, err = newUserDetailsFromRecords(items)
	if err != nil {
		err = fmt.Errorf("userStore.GetDetails: failed to create UserDetails: %w", err)
	}
	return
}

// Invite a User to an Organisation, optionally inviting to Organisation and Service groups.
func (store UserStore) Invite(u User, org Organisation, groups []string, serviceGroups map[string][]string) error {
	now := store.Now()
	organisationMemberRecord := newOrganisationMemberRecord(org, groups, serviceGroups, u)
	organisationGroupMemberItem, err := dynamodbattribute.ConvertToMap(organisationMemberRecord)
	if err != nil {
		return fmt.Errorf("userStore.Invite: failed to convert organisationMemberRecord: %w", err)
	}
	userOrganisationRecord := newUserOrganisationRecord(u, org, now, nil)
	userOrganisationItem, err := dynamodbattribute.ConvertToMap(userOrganisationRecord)
	if err != nil {
		return fmt.Errorf("userStore.Invite: failed to convert userOrganisationRecord: %w", err)
	}
	_, err = store.Client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			*store.TableName: {
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: organisationGroupMemberItem,
					},
				},
				&dynamodb.WriteRequest{
					PutRequest: &dynamodb.PutRequest{
						Item: userOrganisationItem,
					},
				},
			},
		},
	})
	return err
}

// AcceptInvite accepts an invitation to join an Organisation.
func (store UserStore) AcceptInvite(u User, org Organisation) error {
	update := expression.Set(expression.Name("acceptedAt"), expression.Value(store.Now()))
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()
	if err != nil {
		return fmt.Errorf("userStore.AcceptInvite: failed to build query: %v", err)
	}

	_, err = store.Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 store.TableName,
		Key:                       idAndRng(newUserOrganisationRecordHashKey(u.ID), newUserOrganisationRecordRangeKey(org.ID)),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeValues: expr.Values(),
		ExpressionAttributeNames:  expr.Names(),
	})
	return err
}

// RejectInvite rejects an invitation to join an Organisation.
func (store UserStore) RejectInvite(u User, org Organisation) error {
	organisationGroupMemberKey := idAndRng(newOrganisationMemberRecordHashKey(org.ID),
		newOrganisationMemberRecordRangeKey(u.ID))
	userOrganisationRecordKey := idAndRng(newUserOrganisationRecordHashKey(u.ID),
		newUserOrganisationRecordRangeKey(org.ID))
	_, err := store.Client.BatchWriteItem(&dynamodb.BatchWriteItemInput{
		RequestItems: map[string][]*dynamodb.WriteRequest{
			*store.TableName: {
				&dynamodb.WriteRequest{
					DeleteRequest: &dynamodb.DeleteRequest{
						Key: organisationGroupMemberKey,
					},
				},
				&dynamodb.WriteRequest{
					DeleteRequest: &dynamodb.DeleteRequest{
						Key: userOrganisationRecordKey,
					},
				},
			},
		},
	})
	return err
}

// user record.
const userRecordName = "user"

func newUserRecord(user User) userRecord {
	var ur userRecord
	ur.ID = newUserRecordHashKey(user.ID)
	ur.Range = newUserRecordRangeKey()
	ur.Version = 0
	ur.RecordType = userRecordName
	ur.Email = user.ID
	ur.FirstName = user.FirstName
	ur.LastName = user.LastName
	ur.Phone = user.Phone
	ur.CreatedAt = user.CreatedAt
	return ur
}

func newUserRecordHashKey(email string) string {
	return userRecordName + "/" + email
}

func newUserRecordRangeKey() string {
	return userRecordName
}

type userRecord struct {
	record
	userRecordFields
}

type userRecordFields struct {
	Email     string    `json:"email"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"createdAt"`
}

// user organisation record.
const userOrgnisationRecordName = "userOrganisation"

func newUserOrganisationRecordHashKey(email string) string {
	return newUserRecordHashKey(email)
}

func newUserOrganisationRecordRangeKey(organisationID string) string {
	return userOrgnisationRecordName + "/" + organisationID
}

func newUserOrganisationRecord(u User, org Organisation, invitedAt time.Time, acceptedAt *time.Time) userOrganisationRecord {
	var record userOrganisationRecord
	record.ID = newUserOrganisationRecordHashKey(u.ID)
	record.Range = newUserOrganisationRecordRangeKey(org.ID)
	record.RecordType = userOrgnisationRecordName
	record.Version = 0
	record.Email = u.ID
	record.OrganisationID = org.ID
	record.OrganisationName = org.Name
	record.InvitedAt = invitedAt
	record.AcceptedAt = acceptedAt
	return record
}

type userOrganisationRecord struct {
	record
	Email string `json:"email"`
	organisationRecordFields
	InvitedAt  time.Time  `json:"invitedAt"`
	AcceptedAt *time.Time `json:"acceptedAt"`
}
