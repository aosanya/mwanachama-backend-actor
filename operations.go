package mwanachamaactor

import _ "embed"

//go:embed actor.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
