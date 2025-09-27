package cas

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestServiceValidateUrlCas3(t *testing.T) {
	casURL, err := url.Parse("https://cas.example.com/cas")
	if err != nil {
		t.Fatalf("failed to parse cas url: %v", err)
	}

	validator := NewServiceTicketValidator(http.DefaultClient, casURL)
	serviceURL, err := url.Parse("https://app.example.com/")
	if err != nil {
		t.Fatalf("failed to parse service url: %v", err)
	}

	validateURL, err := validator.ServiceValidateUrl(serviceURL, "ST-123", CASVERSION3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(validateURL, "/cas/p3/serviceValidate") {
		t.Fatalf("expected CAS 3.0 service validation endpoint, got %s", validateURL)
	}

	if !strings.Contains(validateURL, "ticket=ST-123") {
		t.Fatalf("expected ticket parameter in url, got %s", validateURL)
	}

	if !strings.Contains(validateURL, "service=https%3A%2F%2Fapp.example.com%2F") {
		t.Fatalf("expected service parameter in url, got %s", validateURL)
	}
}

func TestValidateTicketCas3(t *testing.T) {
	const response = `<?xml version="1.0"?>
<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
  <cas:authenticationSuccess>
    <cas:user>arthur</cas:user>
    <cas:attributes>
      <cas:authenticationDate>2024-09-27T10:11:12.123+08:00[Asia/Shanghai]</cas:authenticationDate>
      <cas:longTermAuthenticationRequestTokenUsed>false</cas:longTermAuthenticationRequestTokenUsed>
      <cas:isFromNewLogin>true</cas:isFromNewLogin>
      <cas:memberOf>hitchhiker</cas:memberOf>
    </cas:attributes>
  </cas:authenticationSuccess>
</cas:serviceResponse>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cas/p3/serviceValidate" {
			t.Fatalf("expected path /cas/p3/serviceValidate, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("ticket") != "ST-42" {
			t.Fatalf("expected ticket ST-42, got %s", r.URL.Query().Get("ticket"))
		}
		if r.URL.Query().Get("service") != "https://app.example.com/" {
			t.Fatalf("unexpected service value: %s", r.URL.Query().Get("service"))
		}

		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, response)
	}))
	defer server.Close()

	casURL, err := url.Parse(server.URL + "/cas")
	if err != nil {
		t.Fatalf("failed to parse cas url: %v", err)
	}

	validator := NewServiceTicketValidator(server.Client(), casURL)
	serviceURL, err := url.Parse("https://app.example.com/")
	if err != nil {
		t.Fatalf("failed to parse service url: %v", err)
	}

	auth, err := validator.ValidateTicket(serviceURL, "ST-42", CASVERSION3)
	if err != nil {
		t.Fatalf("unexpected error validating ticket: %v", err)
	}

	if auth.User != "arthur" {
		t.Fatalf("unexpected user: %s", auth.User)
	}

	if len(auth.MemberOf) != 1 || auth.MemberOf[0] != "hitchhiker" {
		t.Fatalf("unexpected memberOf: %#v", auth.MemberOf)
	}

	if auth.AuthenticationDate.IsZero() {
		t.Fatalf("authentication date should be parsed for CAS 3 response")
	}

	if auth.IsNewLogin != true || auth.IsRememberedLogin != false {
		t.Fatalf("unexpected login flags: new=%v remembered=%v", auth.IsNewLogin, auth.IsRememberedLogin)
	}
}
