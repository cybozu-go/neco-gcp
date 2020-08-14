package slack

import "fmt"

// UndefinedResourceTypeError arises when undefined resource type is given
type UndefinedResourceTypeError struct {
	ResourceType string
}

// Error implements error interface
func (e UndefinedResourceTypeError) Error() string {
	return fmt.Sprintf("undefined resource type: %s", e.ResourceType)
}
