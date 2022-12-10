//
// Copyright (C) 2022 Dmitry Kolesnikov
//
// This file may be modified and distributed under the terms
// of the MIT license.  See the LICENSE file for details.
// https://github.com/fogfish/dynamo
//

package ddb

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/fogfish/dynamo/v2"
	"github.com/fogfish/golem/pure/hseq"
)

type Schema[T dynamo.Thing, A any] string

func (schema Schema[T, A]) Updater() UpdateExpression[T, A] {
	return hseq.FMap1(
		generic[T](string(schema)),
		newUpdateExpression[T, A],
	)
}

func (schema Schema[T, A]) Condition() ConditionExpression[T, A] {
	return hseq.FMap1(
		generic[T](string(schema)),
		newConditionExpression[T, A],
	)
}

// generic[T] filters hseq.Generic[T] list with defined fields
func generic[T any](fs ...string) hseq.Seq[T] {
	seq := make(hseq.Seq[T], 0)
	for _, t := range hseq.Generic[T]() {
		for _, f := range fs {
			if t.Name == f {
				seq = append(seq, t)
			}
		}
	}
	return seq
}

//
// Internal data structure to manage type schema
//

// Schema is utility that decodes type into projection expression
type schema[T dynamo.Thing] struct {
	ExpectedAttributeNames map[string]string
	Projection             *string
}

func newSchema[T dynamo.Thing]() *schema[T] {
	seq := hseq.FMap(
		hseq.Generic[T](),
		func(t hseq.Type[T]) string {
			name := t.StructField.Tag.Get("dynamodbav")
			return strings.Split(name, ",")[0]
		},
	)

	names := make(map[string]string, len(seq))
	attrs := make([]string, len(seq))

	for i, x := range seq {
		name := "#__" + x + "__"
		names[name] = x
		attrs[i] = name
	}

	return &schema[T]{
		ExpectedAttributeNames: names,
		Projection:             aws.String(strings.Join(attrs, ", ")),
	}
}