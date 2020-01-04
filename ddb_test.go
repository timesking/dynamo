package dynamo_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/fogfish/dynamo"
	"github.com/fogfish/it"
)

type person struct {
	dynamo.IRI
	Name    string `dynamodbav:"name,omitempty"`
	Age     int    `dynamodbav:"age,omitempty"`
	Address string `dynamodbav:"address,omitempty"`
}

func entity() person {
	return person{
		IRI:     dynamo.IRI{"dead", "beef"},
		Name:    "Verner Pleishner",
		Age:     64,
		Address: "Blumenstrasse 14, Berne, 3013",
	}
}

func TestDdbGet(t *testing.T) {
	val := person{IRI: dynamo.IRI{"dead", "beef"}}
	err := ddb().Get(&val)

	it.Ok(t).
		If(err).Should().Equal(nil).
		If(val).Should().Equal(entity())
}

func TestDdbPut(t *testing.T) {
	it.Ok(t).If(ddb().Put(entity())).Should().Equal(nil)
}

func TestDdbRemove(t *testing.T) {
	it.Ok(t).If(ddb().Remove(entity())).Should().Equal(nil)
}

func TestDdbUpdate(t *testing.T) {
	val := person{IRI: dynamo.IRI{"dead", "beef"}}
	err := ddb().Update(&val)

	it.Ok(t).
		If(err).Should().Equal(nil).
		If(val).Should().Equal(entity())
}

func TestDdbMatch(t *testing.T) {
	cnt := 0
	seq := ddb().Match(dynamo.IRI{Prefix: "dead"})

	for seq.Tail() {
		cnt++
		val := person{}
		err := seq.Head(&val)

		it.Ok(t).
			If(err).Should().Equal(nil).
			If(val).Should().Equal(entity())
	}

	it.Ok(t).
		If(seq.Error()).Should().Equal(nil).
		If(cnt).Should().Equal(2)
}

//-----------------------------------------------------------------------------
//
// Mock Dynamo DB
//
//-----------------------------------------------------------------------------

func ddb() *dynamo.DB {
	client := &dynamo.DB{}
	client.Mock(&mockDDB{})
	return client
}

type mockDDB struct {
	dynamodbiface.DynamoDBAPI
}

func (m mockDDB) GetItem(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {
	return &dynamodb.GetItemOutput{
		Item: map[string]*dynamodb.AttributeValue{
			"prefix":  {S: aws.String("dead")},
			"address": {S: aws.String("Blumenstrasse 14, Berne, 3013")},
			"name":    {S: aws.String("Verner Pleishner")},
			"suffix":  {S: aws.String("beef")},
			"age":     {N: aws.String("64")},
		},
	}, nil
}

func (m mockDDB) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {
	expect := map[string]*dynamodb.AttributeValue{
		"prefix":  {S: aws.String("dead")},
		"address": {S: aws.String("Blumenstrasse 14, Berne, 3013")},
		"name":    {S: aws.String("Verner Pleishner")},
		"suffix":  {S: aws.String("beef")},
		"age":     {N: aws.String("64")},
	}

	if !reflect.DeepEqual(expect, input.Item) {
		return nil, errors.New("Unexpected entity.")
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m mockDDB) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {
	prefix := input.Key["prefix"]
	suffix := input.Key["suffix"]

	if !reflect.DeepEqual(prefix, &dynamodb.AttributeValue{S: aws.String("dead")}) {
		return nil, errors.New("Unexpected entity.")
	}

	if !reflect.DeepEqual(suffix, &dynamodb.AttributeValue{S: aws.String("beef")}) {
		return nil, errors.New("Unexpected entity.")
	}

	return &dynamodb.DeleteItemOutput{}, nil
}

func (m mockDDB) UpdateItem(*dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	return &dynamodb.UpdateItemOutput{
		Attributes: map[string]*dynamodb.AttributeValue{
			"prefix":  {S: aws.String("dead")},
			"address": {S: aws.String("Blumenstrasse 14, Berne, 3013")},
			"name":    {S: aws.String("Verner Pleishner")},
			"suffix":  {S: aws.String("beef")},
			"age":     {N: aws.String("64")},
		},
	}, nil
}

func (m mockDDB) Query(*dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{
		ScannedCount: aws.Int64(2),
		Count:        aws.Int64(2),
		Items: []map[string]*dynamodb.AttributeValue{
			{
				"prefix":  {S: aws.String("dead")},
				"address": {S: aws.String("Blumenstrasse 14, Berne, 3013")},
				"name":    {S: aws.String("Verner Pleishner")},
				"suffix":  {S: aws.String("beef")},
				"age":     {N: aws.String("64")},
			},
			{
				"prefix":  {S: aws.String("dead")},
				"address": {S: aws.String("Blumenstrasse 14, Berne, 3013")},
				"name":    {S: aws.String("Verner Pleishner")},
				"suffix":  {S: aws.String("beef")},
				"age":     {N: aws.String("64")},
			},
		},
	}, nil
}
