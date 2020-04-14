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

// idAndRng creates a DynamoDB key.
func idAndRng(id, rng string) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"id":  {S: aws.String(id)},
		"rng": {S: aws.String(rng)},
	}
}
