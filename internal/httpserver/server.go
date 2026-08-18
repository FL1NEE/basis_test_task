package httpserver

import (
	"embed"
	"net/http"

	"github.com/FL1NEE/basis_test_task/internal/auth"
	"github.com/FL1NEE/basis_test_task/internal/service"
)

//go:embed openapi.yaml swagger.html
var docsFS embed.FS

type Services struct {
	Auth     *service.AuthService
	Teams    *service.TeamService
	Tasks    *service.TaskService
	Comments *service.CommentService
	History  *service.HistoryService
	Stats    *service.StatsService
}

func NewRouter(svc Services, tokens *auth.TokenIssuer) http.Handler {
	mux := http.NewServeMux()

	authH := &authHandler{auth: svc.Auth}
	teamH := &teamHandler{teams: svc.Teams}
	taskH := &taskHandler{tasks: svc.Tasks}
	commentH := &commentHandler{comments: svc.Comments}
	historyH := &historyHandler{history: svc.History}
	statsH := &statsHandler{stats: svc.Stats}

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.Handle("GET /metrics", metricsHandler)

	mux.HandleFunc("GET /openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		http.ServeFileFS(w, r, docsFS, "openapi.yaml")
	})
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, docsFS, "swagger.html")
	})

	mux.HandleFunc("POST /api/v1/register", authH.register)
	mux.HandleFunc("POST /api/v1/login", authH.login)

	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/teams", teamH.create)
	protected.HandleFunc("GET /api/v1/teams", teamH.list)
	protected.HandleFunc("POST /api/v1/teams/{id}/invite", teamH.invite)
	protected.HandleFunc("GET /api/v1/teams/{team_id}/stats", statsH.get)

	protected.HandleFunc("POST /api/v1/tasks", taskH.create)
	protected.HandleFunc("GET /api/v1/tasks", taskH.list)
	protected.HandleFunc("PUT /api/v1/tasks/{id}", taskH.update)
	protected.HandleFunc("GET /api/v1/tasks/{id}/history", historyH.list)
	protected.HandleFunc("POST /api/v1/tasks/{id}/comments", commentH.create)
	protected.HandleFunc("GET /api/v1/tasks/{id}/comments", commentH.list)

	mux.Handle("/api/v1/", requireAuth(tokens)(protected))

	return prometheusMiddleware(requestID(requestLogger(mux)))
}
