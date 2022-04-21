//
// Copyright (C) 2022 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/dynamo
//

package ddb_test

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/fogfish/curie"
	"github.com/fogfish/dynamo"
	"github.com/fogfish/dynamo/internal/ddb/ddbtest"
	"github.com/fogfish/dynamo/internal/dynamotest"
	"github.com/fogfish/it"
)

type person struct {
	Prefix  dynamo.IRI `dynamodbav:"prefix,omitempty"`
	Suffix  dynamo.IRI `dynamodbav:"suffix,omitempty"`
	Name    string     `dynamodbav:"name,omitempty"`
	Age     int        `dynamodbav:"age,omitempty"`
	Address string     `dynamodbav:"address,omitempty"`
}

func (p person) HashKey() string { return curie.IRI(p.Prefix).String() }
func (p person) SortKey() string { return curie.IRI(p.Suffix).String() }

func entityStruct() person {
	return person{
		Prefix:  dynamo.NewIRI("dead:beef"),
		Suffix:  dynamo.NewIRI("1"),
		Name:    "Verner Pleishner",
		Age:     64,
		Address: "Blumenstrasse 14, Berne, 3013",
	}
}

func entityDynamo() map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		"prefix":  {S: aws.String("dead:beef")},
		"suffix":  {S: aws.String("1")},
		"address": {S: aws.String("Blumenstrasse 14, Berne, 3013")},
		"name":    {S: aws.String("Verner Pleishner")},
		"age":     {N: aws.String("64")},
	}
}

func codec(p dynamotest.Person) (map[string]*dynamodb.AttributeValue, error) {
	return dynamodbattribute.MarshalMap(p)
}

func TestDynamoDB(t *testing.T) {
	dynamotest.TestGet(t, codec, ddbtest.GetItem[dynamotest.Person])
	dynamotest.TestPut(t, codec, ddbtest.PutItem[dynamotest.Person])
	dynamotest.TestRemove(t, codec, ddbtest.DeleteItem[dynamotest.Person])
	dynamotest.TestUpdate(t, codec, ddbtest.UpdateItem[dynamotest.Person])
	dynamotest.TestMatch(t, codec, ddbtest.Query[dynamotest.Person])
}

func TestDdbPutWithConstrain(t *testing.T) {
	name := dynamo.Schema1[person, string]("Name")
	ddb := ddbtest.Constrains[person](nil)

	success := ddb.Put(entityStruct(), name.Eq("xxx"))
	failure := ddb.Put(entityStruct(), name.Eq("yyy"))

	it.Ok(t).
		If(success).Should().Equal(nil).
		If(failure).Should().Be().Like(dynamo.PreConditionFailed{})
}

func TestDdbRemoveWithConstrain(t *testing.T) {
	name := dynamo.Schema1[person, string]("Name")
	ddb := ddbtest.Constrains[person](nil)

	success := ddb.Remove(entityStruct(), name.Eq("xxx"))
	failure := ddb.Remove(entityStruct(), name.Eq("yyy"))

	it.Ok(t).
		If(success).Should().Equal(nil).
		If(failure).Should().Be().Like(dynamo.PreConditionFailed{})
}

func TestDdbUpdateWithConstrain(t *testing.T) {
	name := dynamo.Schema1[person, string]("Name")
	ddb := ddbtest.Constrains[person](entityDynamo())
	patch := person{
		Prefix: dynamo.NewIRI("dead:beef"),
		Suffix: dynamo.NewIRI("1"),
		Age:    65,
	}

	_, success := ddb.Update(patch, name.Eq("xxx"))
	_, failure := ddb.Update(patch, name.Eq("yyy"))

	it.Ok(t).
		If(success).Should().Equal(nil).
		If(failure).Should().Be().Like(dynamo.PreConditionFailed{})
}