package dynamo_test

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fogfish/iri"
	"github.com/fogfish/it"
)

type Item struct {
	iri.IRI
	Ref *iri.Compact `json:"ref,omitempty"  dynamodbav:"ref,omitempty"`
	Tag string       `json:"tag,omitempty"  dynamodbav:"tag,omitempty"`
}

var fixtureItem Item = Item{
	IRI: iri.New("foo:prefix:suffix"),
	Ref: iri.New("foo:a:suffix").Compact(),
	Tag: "tag",
}
var fixtureJson string = "{\"id\":\"foo:prefix:suffix\",\"ref\":\"foo:a:suffix\",\"tag\":\"tag\"}"

var fixtureEmptyItem Item = Item{
	IRI: iri.New("foo:prefix:suffix"),
}
var fixtureEmptyJson string = "{\"id\":\"foo:prefix:suffix\"}"

var fixtureDdb map[string]*dynamodb.AttributeValue = map[string]*dynamodb.AttributeValue{
	"id":  &dynamodb.AttributeValue{S: aws.String("foo:prefix:suffix")},
	"ref": &dynamodb.AttributeValue{S: aws.String("foo:a:suffix")},
	"tag": &dynamodb.AttributeValue{S: aws.String("tag")},
}

func TestMarshalJSON(t *testing.T) {
	bytes, err := json.Marshal(fixtureItem)

	it.Ok(t).
		If(err).Should().Equal(nil).
		If(string(bytes)).Should().Equal(fixtureJson)
}

func TestMarshalEmptyJSON(t *testing.T) {
	bytes, err := json.Marshal(fixtureEmptyItem)

	it.Ok(t).
		If(err).Should().Equal(nil).
		If(string(bytes)).Should().Equal(fixtureEmptyJson)
}

func TestUnmarshalJSON(t *testing.T) {
	var item Item

	it.Ok(t).
		If(json.Unmarshal([]byte(fixtureJson), &item)).Should().Equal(nil).
		If(item).Should().Equal(fixtureItem)
}

func TestUnmarshalEmptyJSON(t *testing.T) {
	var item Item

	it.Ok(t).
		If(json.Unmarshal([]byte(fixtureEmptyJson), &item)).Should().Equal(nil).
		If(item).Should().Equal(fixtureEmptyItem)
}

func TestMarshalDynamo(t *testing.T) {
	gen, err := dynamodbattribute.MarshalMap(fixtureItem)

	it.Ok(t).
		If(err).Should().Equal(nil).
		If(gen).Should().Equal(fixtureDdb)
}

func TestUnmarshalDynamo(t *testing.T) {
	var item Item

	it.Ok(t).
		If(dynamodbattribute.UnmarshalMap(fixtureDdb, &item)).Should().Equal(nil).
		If(item).Should().Equal(fixtureItem)
}
