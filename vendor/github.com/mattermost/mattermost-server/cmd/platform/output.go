// Copyright (c) 2016-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.
package main

import (
	"fmt"
	"os"
)

func CommandPrintln(a ...interface{}) (int, error) {
	return fmt.Println(a...)
}

func CommandPrintErrorln(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}

func CommandPrettyPrintln(a ...interface{}) (int, error) {
	return fmt.Fprintln(os.Stderr, a...)
}
