package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const version = "0.1.0"

type Health struct {
	Ok      bool   `json:"ok"`
	Version string `json:"version"`
	Time    string `json:"time"`
}

type ContainerDTO struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Image          string `json:"image"`
	State          string `json:"state"`
	Status         string `json:"status"`
	Created        string `json:"created"`
	ComposeProject string `json:"composeProject,omitempty"`
}

func main() {
	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, Health{
			Ok:      true,
			Version: version,
			Time:    time.Now().Format(time.RFC3339),
		})
	})

	// Docker
	r.Route("/api/docker", func(r chi.Router) {
		r.Get("/containers", handleDockerContainers)
		r.Post("/containers/{id}/start", handleDockerAction("start"))
		r.Post("/containers/{id}/stop", handleDockerAction("stop"))
		r.Post("/containers/{id}/restart", handleDockerAction("restart"))
		r.Get("/containers/{id}/logs/stream", handleDockerLogsStream)
	})

	// Production (Compose Projekte in /home/l/Production)
	r.Route("/api/production", func(r chi.Router) {
		r.Get("/projects", handleProductionProjects)
		r.Get("/projects/{name}/compose", handleProductionComposeFile)
		r.Get("/projects/{name}/env", handleProductionEnvFile)

		r.Post("/projects/{name}/pull", handleProductionAction("pull"))
		r.Post("/projects/{name}/up", handleProductionAction("up"))
		r.Post("/projects/{name}/down", handleProductionAction("down"))
		r.Post("/projects/{name}/restart", handleProductionAction("restart"))
	})

	addr := "127.0.0.1:9069"
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen %s: %v", addr, err)
	}

	log.Printf("API listening on http://%s", addr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func dockerClient(r *http.Request) (*client.Client, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	return cli, ctx, cancel, nil
}

func handleDockerContainers(w http.ResponseWriter, r *http.Request) {
	cli, ctx, cancel, err := dockerClient(r)
	if err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	defer cancel()
	defer cli.Close()

	// all containers
	args := filters.NewArgs()
	_ = args

	list, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}

	out := make([]ContainerDTO, 0, len(list))
	for _, c := range list {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		dto := ContainerDTO{
			ID:      c.ID,
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Created: time.Unix(c.Created, 0).Format(time.RFC3339),
		}
		if p, ok := c.Labels["com.docker.compose.project"]; ok && p != "" {
			dto.ComposeProject = p
		}
		out = append(out, dto)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ComposeProject == out[j].ComposeProject {
			return out[i].Name < out[j].Name
		}
		return out[i].ComposeProject < out[j].ComposeProject
	})

	writeJSON(w, http.StatusOK, out)
}

func handleDockerAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		cli, ctx, cancel, err := dockerClient(r)
		if err != nil {
			httpError(w, err, http.StatusInternalServerError)
			return
		}
		defer cancel()
		defer cli.Close()

		switch action {
		case "start":
			if err := cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
				httpError(w, err, http.StatusInternalServerError)
				return
			}
		case "stop":
			// default timeout nil => daemon default; we can set a small one
			t := 10
			if err := cli.ContainerStop(ctx, id, container.StopOptions{Timeout: &t}); err != nil {
				httpError(w, err, http.StatusInternalServerError)
				return
			}
		case "restart":
			t := 10
			if err := cli.ContainerRestart(ctx, id, container.StopOptions{Timeout: &t}); err != nil {
				httpError(w, err, http.StatusInternalServerError)
				return
			}
		default:
			httpError(w, fmt.Errorf("unknown action: %s", action), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func handleDockerLogsStream(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		tail = "200"
	}

	cli, ctx, cancel, err := dockerClient(r)
	if err != nil {
		httpError(w, err, http.StatusInternalServerError)
		return
	}
	// NOTE: do not defer cancel/cli.Close until after stream ends

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: false,
		Details:    false,
		Tail:       tail,
	}

	rc, err := cli.ContainerLogs(ctx, id, opts)
	if err != nil {
		cancel()
		cli.Close()
		httpError(w, err, http.StatusInternalServerError)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		rc.Close()
		cancel()
		cli.Close()
		httpError(w, fmt.Errorf("streaming unsupported"), http.StatusInternalServerError)
		return
	}

	// Docker multiplexes stdout/stderr in a special format by default.
	// The simplest robust approach is to use StdCopy, but that writes to io.Writer not per-line.
	// Here we do a pragmatic line scan and strip potential binary header bytes.
	// Good enough for typical text logs.
	defer rc.Close()
	defer cancel()
	defer cli.Close()

	scanner := bufio.NewScanner(rc)
	// increase buffer for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Initial comment to open the stream
	fmt.Fprintf(w, ": ok\n\n")
	flusher.Flush()

	for scanner.Scan() {
		line := sanitizeDockerLogLine(scanner.Bytes())
		// SSE event
		fmt.Fprintf(w, "data: %s\n\n", escapeSSE(line))
		flusher.Flush()
	}

	// client disconnected or container stopped
}

// ===== Production / Compose manager =====

const productionRoot = "/home/l/Production"

type ProductionProjectDTO struct {
	Name           string `json:"name"`
	Path           string `json:"path"`
	HasEnv         bool   `json:"hasEnv"`
	ContainerTotal int    `json:"containerTotal"`
	ContainerRun   int    `json:"containerRun"`
	ContainerStop  int    `json:"containerStop"`
}

func handleProductionProjects(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(productionRoot)
	if err != nil {
		httpError(w, fmt.Errorf("read production root: %w", err), http.StatusInternalServerError)
		return
	}

	// get docker containers once and aggregate by compose project label
	composeStats := map[string]struct {
		total int
		run   int
		stop  int
	}{}

	cli, ctx, cancel, err := dockerClient(r)
	if err == nil {
		defer cancel()
		defer cli.Close()
		list, derr := cli.ContainerList(ctx, container.ListOptions{All: true})
		if derr == nil {
			for _, c := range list {
				p := c.Labels["com.docker.compose.project"]
				if p == "" {
					continue
				}
				s := composeStats[p]
				s.total++
				if c.State == "running" {
					s.run++
				} else {
					s.stop++
				}
				composeStats[p] = s
			}
		}
	}

	out := make([]ProductionProjectDTO, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// hide dot-prefixed folders (your rule)
		if strings.HasPrefix(name, ".") {
			continue
		}

		dir := productionRoot + "/" + name
		composePath := dir + "/docker-compose.yml"

		if _, err := os.Stat(composePath); err != nil {
			// only include dirs with docker-compose.yml
			continue
		}

		_, envErr := os.Stat(dir + "/.env")
		stats := composeStats[name]

		out = append(out, ProductionProjectDTO{
			Name:           name,
			Path:           dir,
			HasEnv:         envErr == nil,
			ContainerTotal: stats.total,
			ContainerRun:   stats.run,
			ContainerStop:  stats.stop,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

func handleProductionComposeFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	dir := productionRoot + "/" + name

	if strings.HasPrefix(name, ".") {
		httpError(w, fmt.Errorf("not found"), http.StatusNotFound)
		return
	}

	path := dir + "/docker-compose.yml"
	b, err := os.ReadFile(path)
	if err != nil {
		httpError(w, fmt.Errorf("read compose file: %w", err), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name": name,
		"path": path,
		"text": string(b),
	})
}

func handleProductionEnvFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	dir := productionRoot + "/" + name

	if strings.HasPrefix(name, ".") {
		httpError(w, fmt.Errorf("not found"), http.StatusNotFound)
		return
	}

	path := dir + "/.env"
	b, err := os.ReadFile(path)
	if err != nil {
		httpError(w, fmt.Errorf("read .env: %w", err), http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name": name,
		"path": path,
		"text": string(b),
	})
}

func handleProductionAction(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		if name == "" || strings.HasPrefix(name, ".") {
			httpError(w, fmt.Errorf("invalid project"), http.StatusBadRequest)
			return
		}
		dir := productionRoot + "/" + name
		composePath := dir + "/docker-compose.yml"

		if _, err := os.Stat(composePath); err != nil {
			httpError(w, fmt.Errorf("compose not found: %s", composePath), http.StatusNotFound)
			return
		}

		// Keep actions bounded (avoid hanging forever)
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
		defer cancel()

		var args []string
		switch action {
		case "pull":
			// Step 1: pull latest images
			pullCmd := execCommandContext(ctx, "docker",
				"compose", "-f", "docker-compose.yml", "pull",
			)
			pullCmd.Dir = dir

			pullOut, err := pullCmd.CombinedOutput()
			if err != nil {
				httpError(w, fmt.Errorf("pull failed: %v\n%s", err, string(pullOut)), http.StatusInternalServerError)
				return
			}

			// Step 2: recreate containers with latest images
			upCmd := execCommandContext(ctx, "docker",
				"compose", "-f", "docker-compose.yml", "up",
				"-d", "--force-recreate",
			)
			upCmd.Dir = dir

			upOut, err := upCmd.CombinedOutput()
			if err != nil {
				httpError(w, fmt.Errorf("recreate failed: %v\n%s", err, string(upOut)), http.StatusInternalServerError)
				return
			}

			writeJSON(w, http.StatusOK, map[string]any{
				"ok":      true,
				"action":  "pull-recreate",
				"project": name,
				"output":  string(pullOut) + "\n---\n" + string(upOut),
			})
			return
		case "up":
			args = []string{"compose", "-f", "docker-compose.yml", "up", "-d"}
		case "down":
			args = []string{"compose", "-f", "docker-compose.yml", "down"}
		case "restart":
			args = []string{"compose", "-f", "docker-compose.yml", "restart"}
		default:
			httpError(w, fmt.Errorf("unknown action"), http.StatusBadRequest)
			return
		}

		// Run in project dir so relative paths + env files behave like you expect
		cmd := execCommandContext(ctx, "docker", args...)
		cmd.Dir = dir

		out, err := cmd.CombinedOutput()
		if err != nil {
			httpError(w, fmt.Errorf("%s failed: %v\n%s", action, err, string(out)), http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"action":  action,
			"project": name,
			"output":  string(out),
		})
	}
}

func sanitizeDockerLogLine(b []byte) string {
	// Docker mux header can appear as: 8-byte header then payload
	// We do a cheap heuristic: if first bytes look like mux header, skip 8.
	if len(b) >= 8 && (b[0] == 1 || b[0] == 2) && b[1] == 0 && b[2] == 0 && b[3] == 0 {
		return string(b[8:])
	}
	return string(b)
}

func escapeSSE(s string) string {
	// Prevent breaking SSE framing by replacing CR and ensuring no raw newlines.
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, err error, status int) {
	writeJSON(w, status, map[string]any{
		"ok":    false,
		"error": err.Error(),
	})
}

var execCommandContext = exec.CommandContext
