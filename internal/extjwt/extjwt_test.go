package extjwt

import (
	"testing"
	"text/template"

	"github.com/modfin/epoxy/internal/cf"
)

func TestServiceAccountClaims(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("sa+{{.CommonName}}@modularfinance.se"))

	claims, err := serviceAccountClaims(cf.Claims{CommonName: "example.access"}, tmpl)
	if err != nil {
		t.Fatal(err)
	}

	wantEmail := "sa+example.access@modularfinance.se"
	if claims["email"] != wantEmail {
		t.Fatalf("email = %q, want %q", claims["email"], wantEmail)
	}
	if claims["sub"] != wantEmail {
		t.Fatalf("sub = %q, want %q", claims["sub"], wantEmail)
	}
	if claims["service_account"] != true {
		t.Fatalf("service_account = %v, want true", claims["service_account"])
	}
	if claims["cf_common_name"] != "example.access" {
		t.Fatalf("cf_common_name = %q, want %q", claims["cf_common_name"], "example.access")
	}
}

func TestServiceAccountClaimsRejectsInvalidEmail(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse("not an email"))

	_, err := serviceAccountClaims(cf.Claims{CommonName: "example.access"}, tmpl)
	if err == nil {
		t.Fatal("expected invalid email error")
	}
}

func TestIsServiceAccount(t *testing.T) {
	if !isServiceAccount(cf.Claims{CommonName: "example.access"}) {
		t.Fatal("expected common_name without email to be treated as service account")
	}
	if isServiceAccount(cf.Claims{Email: "user@example.com", CommonName: "example.access"}) {
		t.Fatal("did not expect claims with email to be treated as service account")
	}
	if isServiceAccount(cf.Claims{}) {
		t.Fatal("did not expect empty claims to be treated as service account")
	}
}
