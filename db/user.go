package db

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
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

type userOrganisationRecord struct {
	record
	Email string `json:"email"`
	organisationRecordFields
	InvitedAt  time.Time  `json:"invitedAt"`
	AcceptedAt *time.Time `json:"acceptedAt"`
}
