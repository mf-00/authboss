package authboss

import (
	"net/http"
	"testing"
)

func TestMobileDetector(t *testing.T) {
	r, _ := http.NewRequest("GET", "http://localhost", nil)

	r.Header.Set("User-Agent", `Mozilla/5.0 (Linux; Android 4.4.2; Nexus 5 Build/KOT49H) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/33.0.1750.166 Mobile Safari/537.36`)
	if !basicMobileDetector(r) {
		t.Error("It should have detected as mobile")
	}

	r.Header.Set("User-Agent", `Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.101 Safari/537.36`)
	if basicMobileDetector(r) {
		t.Error("It should have detected as desktop")
	}
}
