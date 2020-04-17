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
	var serviceGroups map[string][]string
	omr := newOrganisationMemberRecord(org, []string{GroupOwner}, serviceGroups, owner)
	omrItem, err := dynamodbattribute.MarshalMap(omr)
	if err != nil {
		return
	}
	putOrganisationGroupMember := &dynamodb.Put{
		TableName: store.TableName,
		Item:      omrItem,
	}

	// Include the user side of the ownership relationship.
	userOrganisationRecord := newUserOrganisationRecord(owner, org, now, &now)
	userOrganisationItem, err := dynamodbattribute.MarshalMap(userOrganisationRecord)
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
			{Put: putNewOrganisation},
			{Put: putOrganisationGroupMember},
			{Put: putUserOrganisation},
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

// CreateService creates a new service.
func (store OrganisationStore) CreateService(id string, serviceName string) (serviceID string, err error) {
	serviceID = uuid.New().String()
	err = store.PutService(id, serviceID, serviceName)
	return
}

// PutService creates a new service or updates an existing service's name.
func (store OrganisationStore) PutService(id string, serviceID, serviceName string) (err error) {
	organisationServiceRecord := newOrganisationServiceRecord(id, serviceID, serviceName)
	item, err := dynamodbattribute.MarshalMap(organisationServiceRecord)
	if err != nil {
		return
	}
	_, err = store.Client.PutItem(&dynamodb.PutItemInput{
		TableName: store.TableName,
		Item:      item,
	})
	return
}

// DeleteService deletes a service from the Organisation. It does not remove assignments to the deleted service.
// These could be removed by a separate process if required.
func (store OrganisationStore) DeleteService(id, serviceID string) (err error) {
	key := idAndRng(newOrganisationServiceRecordHashKey(id), newOrganisationServiceRecordRangeKey(serviceID))
	_, err = store.Client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: store.TableName,
		Key:       key,
	})
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

// AddUserToOrganisationGroups puts a user into groups within the Organisation. If they already exist, the user is added to the group.
func (store OrganisationStore) AddUserToOrganisationGroups(organisationID string, user User, groups ...string) error {
	return store.AddUserToGroups(organisationID, user, groups, nil)
}

// AddUserToServiceGroups puts a user into groups within an Organisation Service.
func (store OrganisationStore) AddUserToServiceGroups(organisationID string, user User, serviceID string, groups ...string) error {
	return store.AddUserToGroups(organisationID, user, nil, map[string][]string{
		serviceID: groups,
	})
}

// AddUserToGroups adds a user to Organisation and Service Groups.
func (store OrganisationStore) AddUserToGroups(organisationID string, user User, groups []string, serviceIDToGroups map[string][]string) error {
	gs := newGroupSet(groups, serviceIDToGroups)
	update := expression.
		Set(expression.Name("typ"), expression.Value(organisationMemberRecordName)).
		Set(expression.Name("v"), expression.Value(0)).
		Set(expression.Name("organisationId"), expression.Value(organisationID)).
		Add(expression.Name("groups"), expression.Value(gs)).
		Set(expression.Name("email"), expression.Value(user.ID)).
		Set(expression.Name("firstName"), expression.Value(user.FirstName)).
		Set(expression.Name("lastName"), expression.Value(user.LastName)).
		Set(expression.Name("phone"), expression.Value(user.Phone)).
		Set(expression.Name("createdAt"), expression.Value(user.CreatedAt))
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()
	if err != nil {
		return err
	}
	_, err = store.Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 store.TableName,
		Key:                       idAndRng(newOrganisationMemberRecordHashKey(organisationID), newOrganisationMemberRecordRangeKey(user.ID)),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	return err
}

// RemoveUserFromOrganisationGroups removes a user from a set of Organisation level groups.
func (store OrganisationStore) RemoveUserFromOrganisationGroups(organisationID, userID string, groups ...string) error {
	return store.RemoveUserFromGroups(organisationID, userID, groups, nil)
}

// RemoveUserFromServiceGroups removes a user from a set of Service-level groups.
func (store OrganisationStore) RemoveUserFromServiceGroups(organisationID, userID, serviceID string, groups ...string) error {
	return store.RemoveUserFromGroups(organisationID, userID, nil, map[string][]string{
		serviceID: groups,
	})
}

// RemoveUserFromGroups removes a user from Organisation and Service groups.
func (store OrganisationStore) RemoveUserFromGroups(organisationID, userID string, groups []string, serviceIDToGroups map[string][]string) error {
	gs := newGroupSet(groups, serviceIDToGroups)
	update := expression.Delete(expression.Name("groups"), expression.Value(gs))
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()
	if err != nil {
		return err
	}
	_, err = store.Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 store.TableName,
		Key:                       idAndRng(newOrganisationMemberRecordHashKey(organisationID), newOrganisationMemberRecordRangeKey(userID)),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	return err
}

// RemoveUser from the Organisation.
func (store OrganisationStore) RemoveUser(organisationID string, userID string) error {
	key := idAndRng(newOrganisationMemberRecordHashKey(organisationID), newOrganisationMemberRecordRangeKey(userID))
	_, err := store.Client.DeleteItem(&dynamodb.DeleteItemInput{
		TableName: store.TableName,
		Key:       key,
	})
	return err
}

// UpdateUserDetails updates a user's details within the Organisation.
func (store OrganisationStore) UpdateUserDetails(organisationID, userID, firstName, lastName, phone string) error {
	update := expression.
		Set(expression.Name("firstName"), expression.Value(firstName)).
		Set(expression.Name("lastName"), expression.Value(lastName)).
		Set(expression.Name("phone"), expression.Value(phone))
	expr, err := expression.NewBuilder().
		WithUpdate(update).
		Build()
	if err != nil {
		return err
	}
	_, err = store.Client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName:                 store.TableName,
		Key:                       idAndRng(newOrganisationMemberRecordHashKey(organisationID), newOrganisationMemberRecordRangeKey(userID)),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		UpdateExpression:          expr.Update(),
	})
	return err

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

// organisation member record.
const organisationMemberRecordName = "organisationGroupMember"

func newOrganisationMemberRecordHashKey(organisationID string) string {
	return newOrganisationRecordHashKey(organisationID)
}

func newOrganisationMemberRecordRangeKey(emailAddress string) string {
	return organisationMemberRecordName + "/" + emailAddress
}

func newOrganisationMemberRecord(org Organisation, groups []string, serviceGroups map[string][]string, u User) organisationMemberRecord {
	var record organisationMemberRecord
	record.ID = newOrganisationMemberRecordHashKey(org.ID)
	record.Range = newOrganisationMemberRecordRangeKey(u.ID)
	record.RecordType = organisationMemberRecordName
	record.Version = 0

	record.OrganisationID = org.ID

	record.Groups = newGroupSet(groups, serviceGroups)

	// userRecordFields
	record.Email = u.ID
	record.FirstName = u.FirstName
	record.LastName = u.LastName
	record.Phone = u.Phone
	record.CreatedAt = u.CreatedAt
	return record
}

type organisationMemberRecord struct {
	record
	OrganisationID string    `json:"organisationId"`
	Groups         *groupSet `json:"groups"`
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

func newOrganisationServiceRecord(organisationID, serviceID, name string) organisationServiceRecord {
	var record organisationServiceRecord
	record.ID = newOrganisationServiceRecordHashKey(organisationID)
	record.Range = newOrganisationServiceRecordRangeKey(serviceID)
	record.RecordType = organisationServiceRecordName
	record.Version = 0
	record.ServiceID = serviceID
	record.ServiceName = name
	return record
}

type organisationServiceRecord struct {
	record
	ServiceID   string `json:"serviceId"`
	ServiceName string `json:"serviceName"`
}
