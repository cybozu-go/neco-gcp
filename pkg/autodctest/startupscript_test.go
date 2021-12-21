package autodctest

import (
	"strings"
	"testing"
)

func TestNecoStartupScriptBuilder(t *testing.T) {
	_, err := NewStartupScriptBuilder().WithNecoApps("this-is-neco-apps")
	if err == nil {
		t.Errorf("should fail when neco-apps is enabled without neco")
	}

	builder, err := NewStartupScriptBuilder().
		WithFluentd().
		WithNeco("this-is-neco").
		WithNecoApps("this-is-neco-apps")
	if err != nil {
		t.Errorf("should not fail because neco-apps is enabled after neco is enabled")
	}

	s := builder.Build()
	shouldContain := []string{
		"service google-fluentd restart",           // check for .WithFluentd
		"git clone --depth 1 -b this-is-neco",      // check for .WithNeco
		"git clone --depth 1 -b this-is-neco-apps", // check for .WithNecoApps
	}
	for _, v := range shouldContain {
		if !strings.Contains(s, v) {
			t.Errorf("should contain %q", v)
		}
	}
}
