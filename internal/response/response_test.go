package response

import (
	"html/template"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/authboss.v0"
	"gopkg.in/authboss.v0/internal/mocks"
)

var testViewTemplate = template.Must(template.New("").Parse(`{{.external}} {{.fun}} {{.flash_success}} {{.flash_error}} {{.xsrfName}} {{.xsrfToken}}`))
var testMobileViewTemplate = template.Must(template.New("").Parse(`{{.ismobile}}`))
var testEmailHTMLTemplate = template.Must(template.New("").Parse(`<h2>{{.}}</h2>`))
var testEmailPlainTemplate = template.Must(template.New("").Parse(`i am a {{.}}`))

func TestLoadTemplates(t *testing.T) {
	t.Parallel()

	dir := os.TempDir()
	file, err := ioutil.TempFile(dir, "authboss")
	if err != nil {
		t.Error("Unexpected error:", err)
	}
	tmpFileName := filepath.Base(file.Name())
	tmpFilePath := filepath.Dir(file.Name())
	fileMobile, err := os.Create(filepath.Join(tmpFilePath, "mobile_"+tmpFileName))
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	defer os.Remove(file.Name())
	defer os.Remove(fileMobile.Name())

	if _, err := file.Write([]byte("{{.Val}}")); err != nil {
		t.Error("Error writing to temp file", err)
	}
	if _, err := fileMobile.Write([]byte("{{.Val}}")); err != nil {
		t.Error("Error writing to temp file", err)
	}

	layout, err := template.New("").Parse(`<strong>{{template "authboss" .}}</strong>`)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	mobile, err := template.New("").Parse(`<strong>{{template "authboss" .}}</strong>`)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	filename := filepath.Base(file.Name())

	tpls, err := LoadTemplates(authboss.New(), layout, mobile, filepath.Dir(file.Name()), filename)
	if err != nil {
		t.Error("Unexpected error:", err)
	}

	if len(tpls) != 2 {
		t.Error("Expected 2 templates:", len(tpls))
	}

	if _, ok := tpls[filename]; !ok {
		t.Error("Expected tpl with name:", filename)
	}
}

func TestTemplates_Render(t *testing.T) {
	t.Parallel()

	cookies := mocks.NewMockClientStorer()
	ab := authboss.New()
	ab.LayoutDataMaker = func(_ http.ResponseWriter, _ *http.Request) authboss.HTMLData {
		return authboss.HTMLData{"fun": "is"}
	}
	ab.XSRFName = "do you think"
	ab.XSRFMaker = func(_ http.ResponseWriter, _ *http.Request) string {
		return "that's air you're breathing now?"
	}

	// Set up flashes
	cookies.Put(authboss.FlashSuccessKey, "no")
	cookies.Put(authboss.FlashErrorKey, "spoon")

	r, _ := http.NewRequest("GET", "http://localhost", nil)
	w := httptest.NewRecorder()
	ctx, _ := ab.ContextFromRequest(r)
	ctx.SessionStorer = cookies

	tpls := Templates{
		"hello": testViewTemplate,
	}

	err := tpls.Render(ctx, w, r, "hello", authboss.HTMLData{"external": "there"})
	if err != nil {
		t.Error(err)
	}

	if w.Body.String() != "there is no spoon do you think that's air you're breathing now?" {
		t.Error("Body was wrong:", w.Body.String())
	}
}

func TestTemplates_RenderMobile(t *testing.T) {
	t.Parallel()

	cookies := mocks.NewMockClientStorer()
	ab := authboss.New()
	ab.LayoutDataMaker = func(_ http.ResponseWriter, _ *http.Request) authboss.HTMLData {
		return nil
	}
	ab.XSRFName = "mobile"
	ab.XSRFMaker = func(_ http.ResponseWriter, _ *http.Request) string { return "mobile" }

	ab.MobileDetector = func(r *http.Request) bool {
		return true
	}

	r, _ := http.NewRequest("GET", "http://localhost", nil)
	r.Header.Set("User-Agent", `Mozilla/5.0 (Linux; Android 4.4.2; Nexus 5 Build/KOT49H) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/33.0.1750.166 Mobile Safari/537.36`)
	w := httptest.NewRecorder()
	ctx, _ := ab.ContextFromRequest(r)
	ctx.SessionStorer = cookies

	tpls := Templates{
		"mobile_hello": testMobileViewTemplate,
	}

	err := tpls.Render(ctx, w, r, "hello", authboss.HTMLData{"ismobile": "true"})
	if err != nil {
		t.Error(err)
	}

	if w.Body.String() != "true" {
		t.Error("Body was wrong:", w.Body.String())
	}
}

func Test_Email(t *testing.T) {
	t.Parallel()

	ab := authboss.New()
	mockMailer := &mocks.MockMailer{}
	ab.Mailer = mockMailer

	htmlTpls := Templates{"html": testEmailHTMLTemplate}
	textTpls := Templates{"plain": testEmailPlainTemplate}

	email := authboss.Email{
		To: []string{"a@b.c"},
	}

	err := Email(ab.Mailer, email, htmlTpls, "html", textTpls, "plain", "spoon")
	if err != nil {
		t.Error(err)
	}

	if len(mockMailer.Last.To) != 1 {
		t.Error("Expected 1 to addr")
	}
	if mockMailer.Last.To[0] != "a@b.c" {
		t.Error("Unexpected to addr @ 0:", mockMailer.Last.To[0])
	}

	if mockMailer.Last.HTMLBody != "<h2>spoon</h2>" {
		t.Error("Unexpected HTMLBody:", mockMailer.Last.HTMLBody)
	}

	if mockMailer.Last.TextBody != "i am a spoon" {
		t.Error("Unexpected TextBody:", mockMailer.Last.TextBody)
	}
}

func TestRedirect(t *testing.T) {
	t.Parallel()

	ab := authboss.New()
	cookies := mocks.NewMockClientStorer()

	r, _ := http.NewRequest("GET", "http://localhost", nil)
	w := httptest.NewRecorder()
	ctx, _ := ab.ContextFromRequest(r)
	ctx.SessionStorer = cookies

	Redirect(ctx, w, r, "/", "success", "failure", false)

	if w.Code != http.StatusFound {
		t.Error("Expected a redirect.")
	}

	if w.Header().Get("Location") != "/" {
		t.Error("Expected to be redirected to root.")
	}

	if val, _ := cookies.Get(authboss.FlashSuccessKey); val != "success" {
		t.Error("Flash success msg wrong:", val)
	}
	if val, _ := cookies.Get(authboss.FlashErrorKey); val != "failure" {
		t.Error("Flash failure msg wrong:", val)
	}
}

func TestRedirect_Override(t *testing.T) {
	t.Parallel()

	ab := authboss.New()
	cookies := mocks.NewMockClientStorer()

	r, _ := http.NewRequest("GET", "http://localhost?redir=foo/bar", nil)
	w := httptest.NewRecorder()
	ctx, _ := ab.ContextFromRequest(r)
	ctx.SessionStorer = cookies

	Redirect(ctx, w, r, "/shouldNotGo", "success", "failure", true)

	if w.Code != http.StatusFound {
		t.Error("Expected a redirect.")
	}

	if w.Header().Get("Location") != "/foo/bar" {
		t.Error("Expected to be redirected to root.")
	}

	if val, _ := cookies.Get(authboss.FlashSuccessKey); val != "success" {
		t.Error("Flash success msg wrong:", val)
	}
	if val, _ := cookies.Get(authboss.FlashErrorKey); val != "failure" {
		t.Error("Flash failure msg wrong:", val)
	}
}
