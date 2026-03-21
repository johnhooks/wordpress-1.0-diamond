package server

import (
	"log"
	"net/http"

	"press/internal/auth"
)

// LoginData is the template data for the admin-login template.
type LoginData struct {
	SiteName    string
	SiteURL     string
	Error       string
	RedirectTo  string
	CanRegister bool
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	name, _ := s.blogInfo(r.Context())
	redirectTo := auth.SafeRedirect(r.URL.Query().Get("redirect_to"), "/wp-admin/")

	s.renderAdmin(w, "admin-login", LoginData{
		SiteName:   name,
		SiteURL:    s.linker.SiteURL,
		RedirectTo: redirectTo,
	})
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.httpError(w, r, "Bad request", http.StatusBadRequest)
		return
	}

	login := r.FormValue("log")
	password := r.FormValue("pwd")
	redirectTo := auth.SafeRedirect(r.FormValue("redirect_to"), "/wp-admin/")

	token, err := s.auth.Login(r.Context(), login, password, r.RemoteAddr, r.UserAgent())
	if err != nil {
		name, _ := s.blogInfo(r.Context())
		s.renderAdmin(w, "admin-login", LoginData{
			SiteName:   name,
			SiteURL:    s.linker.SiteURL,
			Error:      "Invalid login or password.",
			RedirectTo: redirectTo,
		})
		return
	}

	s.auth.SetCookie(w, token)
	http.Redirect(w, r, redirectTo, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.CookieName); err == nil {
		s.auth.Logout(r.Context(), cookie.Value)
	}
	s.auth.ClearCookie(w)
	http.Redirect(w, r, "/wp-admin/login", http.StatusFound)
}

func (s *Server) handleAdminDashboard(w http.ResponseWriter, r *http.Request) {
	// WP 1.0 had no dashboard. It went straight to write.
	http.Redirect(w, r, "/wp-admin/post/new", http.StatusFound)
}

// csrf builds a CSRFHelper from the request context. Admin handlers
// include this in their template data so templates can use .CSRF.For.
func (s *Server) csrf(r *http.Request) auth.CSRFHelper {
	token := auth.SessionTokenFromContext(r.Context())
	return auth.NewCSRFHelper(token, s.cfg.SecretKey)
}

// renderAdmin executes an admin template.
func (s *Server) renderAdmin(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.adminTmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("admin template error: %v", err)
		http.Error(w, "An internal error occurred.", http.StatusInternalServerError)
	}
}
