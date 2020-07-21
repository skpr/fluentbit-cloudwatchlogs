package flush

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupName(t *testing.T) {
	// Test an override.
	actual, err := groupName("prefix", "example", "container", map[string]string{
		AnnotationGroupOverride: "/skpr/example/foo/bar",
	})
	assert.Nil(t, err)
	assert.Equal(t, "/skpr/example/foo/bar", actual)

	// Test a generated group name.
	actual, err = groupName("prefix", "example", "container", map[string]string{
		AnnotationProject: "project",
		AnnotationEnvironment: "environment",
	})
	assert.Nil(t, err)
	assert.Equal(t, "/prefix/example/project/environment/container", actual)
}
