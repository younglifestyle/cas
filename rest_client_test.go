package cas

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestRequestGrantingTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cas/v1/tickets" || r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			w.WriteHeader(415)
			return
		}

		if r.FormValue("username") != "tricia" && r.FormValue("password") != "hitchhiker" {
			w.WriteHeader(400)
			return
		}

		w.Header().Set("Location", "/cas/v1/tickets/TGT-abc")
		w.WriteHeader(201)
	}))
	defer server.Close()

	casURL, err := url.Parse(server.URL + "/cas/")
	if err != nil {
		t.Error("failed to create cas url from test server")
	}

	restClient := NewRestClient(&RestOptions{
		CasURL: casURL,
		Client: server.Client(),
	})

	tgt, err := restClient.RequestGrantingTicket("tricia", "hitchhiker")
	if err != nil {
		t.Errorf("requesting granting ticket failed: %v", err)
	}

	if tgt != "TGT-abc" {
		t.Errorf("expected %s but received %v", "TGT-abc", tgt)
	}

	_, err = restClient.RequestGrantingTicket("arthur", "dent")
	if err == nil {
		t.Errorf("authentication should fail for arthur")
	}
}

func TestRequestServiceTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cas/v1/tickets/TGT-abc" || r.Method != "POST" {
			w.WriteHeader(404)
			return
		}

		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			w.WriteHeader(415)
			return
		}

		if r.FormValue("service") != "https://hitchhiker.com/heartOfGold" {
			w.WriteHeader(400)
			return
		}

		w.WriteHeader(200)
		w.Write([]byte("ST-123"))
	}))
	defer server.Close()

	casURL, err := url.Parse(server.URL + "/cas/")
	if err != nil {
		t.Error("failed to create cas url from test server")
	}

	serviceURL, err := url.Parse("https://hitchhiker.com/heartOfGold")
	if err != nil {
		t.Error("failed to create service url")
	}

	restClient := NewRestClient(&RestOptions{
		CasURL:     casURL,
		ServiceURL: serviceURL,
		Client:     server.Client(),
	})

	st, err := restClient.RequestServiceTicket(TicketGrantingTicket("TGT-abc"))
	if err != nil {
		t.Errorf("requesting service ticket failed: %v", err)
	}

	if st != "ST-123" {
		t.Errorf("expected %s but received %v", "ST-123", st)
	}

	_, err = restClient.RequestServiceTicket(TicketGrantingTicket("TGT-xyz"))
	if err == nil {
		t.Errorf("service ticket request should fail for TGT-xyz")
	}

	serviceURL, err = url.Parse("https://hitchhiker.com/restaurantAtTheEndOfTheUniverse")
	if err != nil {
		t.Error("failed to create service url")
	}

	restClient = NewRestClient(&RestOptions{
		CasURL:     casURL,
		ServiceURL: serviceURL,
		Client:     server.Client(),
	})

	_, err = restClient.RequestServiceTicket(TicketGrantingTicket("TGT-abc"))
	if err == nil {
		t.Errorf("service ticket request should fail for this service")
	}
}

func TestValidateService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cas/v1/tickets/TGT-abc" || r.Method != "DELETE" {
			w.WriteHeader(404)
			return
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	casURL, err := url.Parse(server.URL + "/cas/")
	if err != nil {
		t.Error("failed to create cas url from test server")
	}

	restClient := NewRestClient(&RestOptions{
		CasURL: casURL,
		Client: server.Client(),
	})

	err = restClient.Logout(TicketGrantingTicket("TGT-abc"))
	if err != nil {
		t.Errorf("logout failed %v", err)
	}

	err = restClient.Logout(TicketGrantingTicket("TGT-xyz"))
	if err == nil {
		t.Errorf("logout should failed for this TGT")
	}
}

func TestLogout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cas/v1/tickets/TGT-abc" || r.Method != "DELETE" {
			w.WriteHeader(404)
			return
		}

		w.WriteHeader(200)
	}))
	defer server.Close()

	casURL, err := url.Parse(server.URL + "/cas/")
	if err != nil {
		t.Error("failed to create cas url from test server")
	}

	restClient := NewRestClient(&RestOptions{
		CasURL: casURL,
		Client: server.Client(),
	})

	err = restClient.Logout(TicketGrantingTicket("TGT-abc"))
	if err != nil {
		t.Errorf("logout failed %v", err)
	}

	err = restClient.Logout(TicketGrantingTicket("TGT-xyz"))
	if err == nil {
		t.Errorf("logout should failed for this TGT")
	}
}
func TestRestClientValidateServiceTicketCas3(t *testing.T) {
	const response = `<?xml version="1.0"?>
<cas:serviceResponse xmlns:cas="http://www.yale.edu/tp/cas">
  <cas:authenticationSuccess>
    <cas:user>ford</cas:user>
    <cas:attributes>
      <cas:authenticationDate>2015-02-10T14:28:42Z</cas:authenticationDate>
      <cas:longTermAuthenticationRequestTokenUsed>true</cas:longTermAuthenticationRequestTokenUsed>
      <cas:isFromNewLogin>false</cas:isFromNewLogin>
    </cas:attributes>
  </cas:authenticationSuccess>
</cas:serviceResponse>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cas/p3/serviceValidate" {
			t.Fatalf("expected path /cas/p3/serviceValidate, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("ticket") != "ST-999" {
			t.Fatalf("expected ticket ST-999, got %s", r.URL.Query().Get("ticket"))
		}
		if r.URL.Query().Get("service") != "https://hitchhiker.com/heartOfGold" {
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

	serviceURL, err := url.Parse("https://hitchhiker.com/heartOfGold")
	if err != nil {
		t.Fatalf("failed to parse service url: %v", err)
	}

	client := NewRestClient(&RestOptions{
		CasURL:     casURL,
		ServiceURL: serviceURL,
		Client:     server.Client(),
		CasVersion: CASVERSION3,
	})

	auth, err := client.ValidateServiceTicket(ServiceTicket("ST-999"))
	if err != nil {
		t.Fatalf("unexpected error validating service ticket: %v", err)
	}

	if auth.User != "ford" {
		t.Fatalf("unexpected user: %s", auth.User)
	}

	if auth.IsRememberedLogin != true || auth.IsNewLogin != false {
		t.Fatalf("unexpected login flags: remembered=%v new=%v", auth.IsRememberedLogin, auth.IsNewLogin)
	}
}
