//
// Copyright (C) 2022 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/dynamo
//

//
// The file implements dynamodb specific constraints
//

package ddb

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/fogfish/dynamo/v2"
	"github.com/fogfish/golem/pure/hseq"
)

// See DynamoDB Conditional Expressions
//
//	https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/Expressions.ConditionExpressions.html
//
// Schema declares type descriptor to express Storage I/O Constrains.
//
// Let's consider a following example:
//
//	type Person struct {
//	  curie.ID
//	  Name    string `dynamodbav:"anothername,omitempty"`
//	}
//
// How to define a condition expression on the field Name? Golang struct defines
// and refers the field by `Name` but DynamoDB stores it under the attribute
// `anothername`. Struct field dynamodbav tag specifies serialization rules.
// Golang does not support a typesafe approach to build a correspondence between
// `Name` ⟷ `anothername`. Developers have to utilize dynamodb attribute
// name(s) in conditional expression and Golang struct name in rest of the code.
// It becomes confusing and hard to maintain.
//
// The Schema is helpers to declare builders for conditional
// expressions. Just declare a global variables next to type definition and
// use them across the application.
//
//	var name = dynamo.Schema[Person, string]("Name").Condition()
//
//	name.Eq("Joe Doe")
//	name.NotExists()
func ClauseFor[T dynamo.Thing, A any](schema string) ConditionExpression[T, A] {
	return hseq.FMap1(
		generic[T](string(schema)),
		newConditionExpression[T, A],
	)
}

type ConditionExpression[T dynamo.Thing, A any] struct{ key string }

func newConditionExpression[T dynamo.Thing, A any](t hseq.Type[T]) ConditionExpression[T, A] {
	tag := t.Tag.Get("dynamodbav")
	if tag == "" {
		panic(fmt.Errorf("field %s of type %T do not have `dynamodbav` tag", t.Name, *new(T)))
	}

	return ConditionExpression[T, A]{strings.Split(tag, ",")[0]}
}

// Internal implementation of Constrain effects for storage
// type Constraints[T dynamo.Thing, A any] struct{ key string }

// Eq is equal condition
//
//	name.Eq(x) ⟼ Field = :value
func (ce ConditionExpression[T, A]) Eq(val A) interface{ ConditionExpression(T) } {
	return &dyadicCondition[T, A]{op: "=", key: ce.key, val: val}
}

// Ne is non equal condition
//
//	name.Ne(x) ⟼ Field <> :value
func (ce ConditionExpression[T, A]) Ne(val A) interface{ ConditionExpression(T) } {
	return &dyadicCondition[T, A]{op: "<>", key: ce.key, val: val}
}

// Lt is less than constraint
//
//	name.Lt(x) ⟼ Field < :value
func (ce ConditionExpression[T, A]) Lt(val A) interface{ ConditionExpression(T) } {
	return &dyadicCondition[T, A]{op: "<", key: ce.key, val: val}
}

// Le is less or equal constain
//
//	name.Le(x) ⟼ Field <= :value
func (ce ConditionExpression[T, A]) Le(val A) interface{ ConditionExpression(T) } {
	return &dyadicCondition[T, A]{op: "<=", key: ce.key, val: val}
}

// Gt is greater than constrain
//
//	name.Le(x) ⟼ Field > :value
func (ce ConditionExpression[T, A]) Gt(val A) interface{ ConditionExpression(T) } {
	return &dyadicCondition[T, A]{op: ">", key: ce.key, val: val}
}

// Ge is greater or equal constrain
//
//	name.Le(x) ⟼ Field >= :value
func (ce ConditionExpression[T, A]) Ge(val A) interface{ ConditionExpression(T) } {
	return &dyadicCondition[T, A]{op: ">=", key: ce.key, val: val}
}

// dyadic condition implementation
type dyadicCondition[T any, A any] struct {
	op  string
	key string
	val A
}

func (op dyadicCondition[T, A]) ConditionExpression(T) {}

func (op dyadicCondition[T, A]) Apply(
	conditionExpression **string,
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
) {
	if op.key == "" {
		return
	}

	lit, err := attributevalue.Marshal(op.val)
	if err != nil {
		return
	}

	key := "#__c_" + op.key + "__"
	let := ":__c_" + op.key + "__"
	expressionAttributeValues[let] = lit
	expressionAttributeNames[key] = op.key
	expr := "(" + key + " " + op.op + " " + let + ")"

	if *conditionExpression == nil {
		*conditionExpression = aws.String(expr)
	} else {
		*conditionExpression = aws.String(**conditionExpression + " and " + expr)
	}
}

// Exists attribute constrain
//
//	name.Exists(x) ⟼ attribute_exists(name)
func (ce ConditionExpression[T, A]) Exists() interface{ ConditionExpression(T) } {
	return &unaryCondition[T]{op: "attribute_exists", key: ce.key}
}

// NotExists attribute constrain
//
//	name.NotExists(x) ⟼ attribute_not_exists(name)
func (ce ConditionExpression[T, A]) NotExists() interface{ ConditionExpression(T) } {
	return &unaryCondition[T]{op: "attribute_not_exists", key: ce.key}
}

// unary condition implementation
type unaryCondition[T any] struct {
	op  string
	key string
}

func (op unaryCondition[T]) ConditionExpression(T) {}

func (op unaryCondition[T]) Apply(
	conditionExpression **string,
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
) {
	if op.key == "" {
		return
	}

	key := "#__c_" + op.key + "__"
	expressionAttributeNames[key] = op.key
	expr := "(" + op.op + "(" + key + ")" + ")"

	if *conditionExpression == nil {
		*conditionExpression = aws.String(expr)
	} else {
		*conditionExpression = aws.String(**conditionExpression + " and " + expr)
	}
}

// Is matches either Eq or NotExists if value is not defined
func (ce ConditionExpression[T, A]) Is(val string) interface{ ConditionExpression(T) } {
	if val == "_" {
		return ce.NotExists()
	}

	return ce.Eq(any(val).(A))
}

// Between attribute condition
//
//	name.Between(a, b) ⟼ Field BETWEEN :a AND :b
func (ce ConditionExpression[T, A]) Between(a, b A) interface{ ConditionExpression(T) } {
	return &betweenCondition[T, A]{key: ce.key, a: a, b: b}
}

// between condition implementation
type betweenCondition[T any, A any] struct {
	key  string
	a, b A
}

func (op betweenCondition[T, A]) ConditionExpression(T) {}

func (op betweenCondition[T, A]) Apply(
	conditionExpression **string,
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
) {
	if op.key == "" {
		return
	}

	litA, err := attributevalue.Marshal(op.a)
	if err != nil {
		return
	}

	litB, err := attributevalue.Marshal(op.b)
	if err != nil {
		return
	}

	key := "#__c_" + op.key + "__"
	letA := ":__c_" + op.key + "_a__"
	letB := ":__c_" + op.key + "_b__"
	expressionAttributeValues[letA] = litA
	expressionAttributeValues[letB] = litB
	expressionAttributeNames[key] = op.key
	expr := "(" + key + " BETWEEN " + letA + " AND " + letB + ")"

	if *conditionExpression == nil {
		*conditionExpression = aws.String(expr)
	} else {
		*conditionExpression = aws.String(**conditionExpression + " and " + expr)
	}
}

// In attribute condition
//
//	name.Between(a, b, c) ⟼ Field IN (:a, :b, :c)
func (ce ConditionExpression[T, A]) In(seq ...A) interface{ ConditionExpression(T) } {
	return &inCondition[T, A]{key: ce.key, seq: seq}
}

// between condition implementation
type inCondition[T any, A any] struct {
	key string
	seq []A
}

func (op inCondition[T, A]) ConditionExpression(T) {}

func (op inCondition[T, A]) Apply(
	conditionExpression **string,
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
) {
	if op.key == "" {
		return
	}

	key := "#__c_" + op.key + "__"
	expressionAttributeNames[key] = op.key

	lits := make([]types.AttributeValue, len(op.seq))
	lets := make([]string, len((op.seq)))
	for i := 0; i < len(op.seq); i++ {
		lit, err := attributevalue.Marshal(op.seq[i])
		if err != nil {
			return
		}
		lits[i] = lit
		lets[i] = ":__c_" + op.key + "_" + strconv.Itoa(i) + "__"
		expressionAttributeValues[lets[i]] = lits[i]
	}

	expr := "(" + key + " IN (" + strings.Join(lets, ",") + "))"

	if *conditionExpression == nil {
		*conditionExpression = aws.String(expr)
	} else {
		*conditionExpression = aws.String(**conditionExpression + " and " + expr)
	}
}

// HasPrefix attribute condition
//
// name.HasPrefix(x) ⟼ begins_with(Field, :value)
func (ce ConditionExpression[T, A]) HasPrefix(val A) interface{ ConditionExpression(T) } {
	return &functionalCondition[T, A]{fun: "begins_with", key: ce.key, val: val}
}

// Contains attribute condition
//
// name.Contains(x) ⟼ contains(Field, :value)
func (ce ConditionExpression[T, A]) Contains(val A) interface{ ConditionExpression(T) } {
	return &functionalCondition[T, A]{fun: "contains", key: ce.key, val: val}
}

// functional condition implementation
type functionalCondition[T any, A any] struct {
	fun string
	key string
	val A
}

func (op functionalCondition[T, A]) ConditionExpression(T) {}

func (op functionalCondition[T, A]) Apply(
	conditionExpression **string,
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
) {
	if op.key == "" {
		return
	}

	lit, err := attributevalue.Marshal(op.val)
	if err != nil {
		return
	}

	key := "#__c_" + op.key + "__"
	let := ":__c_" + op.key + "__"
	expressionAttributeValues[let] = lit
	expressionAttributeNames[key] = op.key
	expr := "(" + op.fun + "(" + key + "," + let + "))"

	if *conditionExpression == nil {
		*conditionExpression = aws.String(expr)
	} else {
		*conditionExpression = aws.String(**conditionExpression + " and " + expr)
	}
}

/*
Internal implementation of conditional expressions for dynamo db
*/
func maybeConditionExpression[T dynamo.Thing](
	conditionExpression **string,
	opts []interface{ ConditionExpression(T) },
) (
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
) {
	if len(opts) > 0 {
		expressionAttributeNames = map[string]string{}
		expressionAttributeValues = map[string]types.AttributeValue{}

		for _, opt := range opts {
			if ap, ok := opt.(interface {
				Apply(**string, map[string]string, map[string]types.AttributeValue)
			}); ok {
				ap.Apply(conditionExpression, expressionAttributeNames, expressionAttributeValues)
			}
		}

		// Unfortunately empty maps are not accepted by DynamoDB
		if len(expressionAttributeNames) == 0 {
			expressionAttributeNames = nil
		}
		if len(expressionAttributeValues) == 0 {
			expressionAttributeValues = nil
		}
	}
	return
}

/*
Internal implementation of conditional expressions for dynamo db in the case of
update.
*/
func maybeUpdateConditionExpression[T dynamo.Thing](
	conditionExpression **string,
	expressionAttributeNames map[string]string,
	expressionAttributeValues map[string]types.AttributeValue,
	opts []interface{ ConditionExpression(T) },
) {
	for _, opt := range opts {
		if ap, ok := opt.(interface {
			Apply(**string, map[string]string, map[string]types.AttributeValue)
		}); ok {
			ap.Apply(conditionExpression, expressionAttributeNames, expressionAttributeValues)
		}
	}
}
