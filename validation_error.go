// Copyright 2024 Teknologi Umum <opensource@teknologiumum.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package brassite

import (
	"encoding/json"
	"fmt"
)

type ValidationError struct {
	Issues []ValidationIssue `json:"issues"`
}

type ValidationIssue struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func NewValidationError() *ValidationError {
	return &ValidationError{
		Issues: []ValidationIssue{},
	}
}

func (v *ValidationError) AddIssue(field, message string) {
	v.Issues = append(v.Issues, ValidationIssue{
		Field:   field,
		Message: message,
	})
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %v", v.Issues)
}

func (v *ValidationError) HasIssues() bool {
	return len(v.Issues) > 0
}

func (v *ValidationError) String() string {
	// Export to an indented JSON string
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
