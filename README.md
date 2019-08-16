[![GoDoc](https://godoc.org/github.com/gqlc/compiler?status.svg)](https://godoc.org/github.com/gqlc/compiler)
[![Go Report Card](https://goreportcard.com/badge/github.com/gqlc/compiler)](https://goreportcard.com/report/github.com/gqlc/compiler)
[![Build Status](https://travis-ci.org/gqlc/compiler.svg?branch=master)](https://travis-ci.org/gqlc/compiler)
[![codecov](https://codecov.io/gh/gqlc/compiler/branch/master/graph/badge.svg)](https://codecov.io/gh/gqlc/compiler)

# GraphQL Compiler Internals

Package `compiler` provides types and interfaces for interacting with or implementing
a compiler for the GraphQL IDL.

## Features

- Import Tree Reduction
- Type Validation
- Type Merging

### Import Tree Reduction
GraphQL documents can import one another with the following directive:
```graphql
directive @import(paths: [String]!) on DOCUMENT
```

### Type Validation
Type Validation/Checking is provided by implementing the `TypeChecker` interface. The
`Validate` function is a `TypeChecker` that enforces type validation, per the GraphQL spec.

### Type Merging
Type merging handles merging type extensions with their original type definition.