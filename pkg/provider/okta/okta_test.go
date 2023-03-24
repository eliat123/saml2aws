package okta

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/versent/saml2aws/v2/pkg/cfg"
	"github.com/versent/saml2aws/v2/pkg/creds"
)

type stateTokenTests struct {
	title      string
	body       string
	stateToken string
	err        error
}

type parseMfaIdentifierTests struct {
	title      string
	identifier string
	authName   string
	index      int
}

func TestGetStateTokenFromOktaPageBody(t *testing.T) {
	tests := []stateTokenTests{
		{
			title:      "State token in body gets returned",
			body:       "someJavascriptCode();\nvar stateToken = '123456789';\nsomeOtherJavaScriptCode();",
			stateToken: "123456789",
			err:        nil,
		},
		{
			title:      "State token not in body causes error",
			body:       "someJavascriptCode();\nsomeOtherJavaScriptCode();",
			stateToken: "",
			err:        errors.New("cannot find state token"),
		},
		{
			title:      "State token with hyphen handled correctly",
			body:       "someJavascriptCode();\nvar stateToken = '12345\x2D6789';\nsomeOtherJavaScriptCode();",
			stateToken: "12345-6789",
			err:        nil,
		},
	}
	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			stateToken, err := getStateTokenFromOktaPageBody(test.body)
			assert.Equal(t, test.stateToken, stateToken)
			if test.err != nil {
				assert.NotNil(t, err)
				assert.Equal(t, test.err.Error(), err.Error())
			}

		})
	}
}

func TestSetDeviceTokenCookie(t *testing.T) {
	idpAccount := cfg.NewIDPAccount()
	idpAccount.URL = "https://idp.example.com/abcd"
	idpAccount.Username = "user@example.com"

	loginDetails := &creds.LoginDetails{
		Username: "user@example.com",
		Password: "abc123",
		URL:      "https://idp.example.com/abcd",
	}

	oc, err := New(idpAccount)
	assert.Nil(t, err)

	err = oc.setDeviceTokenCookie(loginDetails)
	assert.Nil(t, err)

	expectedDT := fmt.Sprintf("okta_%s_saml2aws", loginDetails.Username)
	actualDT := ""
	for _, c := range oc.client.Jar.Cookies(&url.URL{Scheme: "https", Host: "idp.example.com", Path: "/abc"}) {
		if c.Name == "DT" {
			actualDT = c.Value
		}
	}
	assert.NotEqual(t, actualDT, "")
	assert.Equal(t, expectedDT, actualDT)

}

func TestOktaCfgFlagsDefaultState(t *testing.T) {
	idpAccount := cfg.NewIDPAccount()
	idpAccount.URL = "https://idp.example.com/abcd"
	idpAccount.Username = "user@example.com"

	oc, err := New(idpAccount)
	assert.Nil(t, err)

	assert.False(t, oc.disableSessions, fmt.Errorf("disableSessions should be false by default"))
	assert.True(t, oc.rememberDevice, fmt.Errorf("rememberDevice should be true by default"))
}

func TestOktaCfgFlagsCustomState(t *testing.T) {
	idpAccount := cfg.NewIDPAccount()
	idpAccount.URL = "https://idp.example.com/abcd"
	idpAccount.Username = "user@example.com"

	idpAccount.DisableRememberDevice = true
	oc, err := New(idpAccount)
	assert.Nil(t, err)

	assert.False(t, oc.disableSessions, fmt.Errorf("disableSessions should be false by default"))
	assert.False(t, oc.rememberDevice, fmt.Errorf("DisableRememberDevice was set to true, so rememberDevice should be false"))

	idpAccount.DisableSessions = true

	oc, err = New(idpAccount)
	assert.Nil(t, err)

	assert.True(t, oc.disableSessions, fmt.Errorf("DisableSessions was set to true so disableSessions should be true"))
	assert.False(t, oc.rememberDevice, fmt.Errorf("DisablDisableSessionseRememberDevice was set to true, so rememberDevice should be false"))

}

func TestOktaParseMfaIdentifer(t *testing.T) {
	resp := `{
		"_embedded": {
			"factors": [
				{
					"factorType": "token:software:totp",
					"provider": "GOOGLE",
					"profile": {
						"credentialId": "dade.murphy@example.com"
					}
				},
				{
					"factorType":"webauthn",
					"provider":"FIDO",
					"profile":{
						"authenticatorName":"MacBook Touch ID"
					}
				},
				{
					"factorType":"webauthn",
					"provider":"FIDO",
					"profile":{
						"authenticatorName":"Yubikey 5"
					}
				}
			]
		}
	}`

	tests := []parseMfaIdentifierTests{
		{
			title:      "Google TOTP doesn't have a name",
			identifier: "GOOGLE TOKEN:SOFTWARE:TOTP",
			authName:   "",
			index:      0,
		},
		{
			title:      "WebAuthn tokens have names",
			identifier: "FIDO WEBAUTHN",
			authName:   "MacBook Touch ID",
			index:      1,
		},
		{
			title:      "A second webauthn token with a different name",
			identifier: "FIDO WEBAUTHN",
			authName:   "Yubikey 5",
			index:      2,
		},
	}

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			identifier, authName := parseMfaIdentifer(resp, test.index)
			assert.Equal(t, test.identifier, identifier)
			assert.Equal(t, test.authName, authName)
		})
	}
}
