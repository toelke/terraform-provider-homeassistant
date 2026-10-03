package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestProviderSchemaIsValid(t *testing.T) {
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}

	resp, err := server.GetProviderSchema(context.Background(), &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range resp.Diagnostics {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			t.Errorf("schema diagnostic: %s: %s", d.Summary, d.Detail)
		}
	}
	if resp.Provider == nil || resp.Provider.Block == nil {
		t.Fatal("provider schema is missing")
	}
	sensitive := map[string]bool{}
	for _, a := range resp.Provider.Block.Attributes {
		sensitive[a.Name] = a.Sensitive
	}
	for _, name := range []string{"url", "token", "insecure", "timeout"} {
		if _, ok := sensitive[name]; !ok {
			t.Errorf("attribute %q is missing", name)
		}
	}
	if !sensitive["token"] {
		t.Error("token must be sensitive")
	}
}

func TestConfigureWithoutURLOrTokenNamesArgumentAndEnv(t *testing.T) {
	t.Setenv(envURL, "")
	t.Setenv(envToken, "")
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}

	config, err := tfprotov6.NewDynamicValue(configType, tftypes.NewValue(configType, map[string]tftypes.Value{
		"url":      tftypes.NewValue(tftypes.String, nil),
		"token":    tftypes.NewValue(tftypes.String, nil),
		"insecure": tftypes.NewValue(tftypes.Bool, nil),
		"timeout":  tftypes.NewValue(tftypes.String, nil),
	}))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.ConfigureProvider(context.Background(), &tfprotov6.ConfigureProviderRequest{Config: &config})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range [][]string{{"`url`", envURL}, {"`token`", envToken}} {
		found := false
		for _, d := range resp.Diagnostics {
			if d.Severity == tfprotov6.DiagnosticSeverityError &&
				strings.Contains(d.Detail, want[0]) && strings.Contains(d.Detail, want[1]) {
				found = true
			}
		}
		if !found {
			t.Errorf("no error diagnostic names %s and %s", want[0], want[1])
		}
	}
}

var configType = tftypes.Object{AttributeTypes: map[string]tftypes.Type{
	"url":      tftypes.String,
	"token":    tftypes.String,
	"insecure": tftypes.Bool,
	"timeout":  tftypes.String,
}}
