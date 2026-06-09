// Command supervisor is a tiny PID-1 process manager for the hardened
// (distroless, shell-less) Homebox image. It launches the Go API, the Next.js
// standalone server and Caddy, forwards termination signals to all of them, and
// exits as soon as any child exits — so the container is restarted by the
// orchestrator rather than silently running degraded.
//
// The alpine-based images use s6-overlay instead; this exists only because
// distroless has no shell or supervisor to run an s6 install.
package main

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

type service struct {
	name string
	argv []string
}

func main() {
	// node and caddy live at different paths across the alpine and distroless
	// bases, so resolve them via PATH; the API binary is always at /app/api.
	node := lookup("node", "/nodejs/bin/node")
	caddyBin := lookup("caddy", "/usr/bin/caddy")

	services := []service{
		{
			name: "api",
			argv: []string{"/app/api", "/data/config.yml"},
		},
		{
			name: "ui",
			argv: []string{node, "/app/ui/server.js"},
		},
		{
			name: "caddy",
			argv: []string{caddyBin, "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"},
		},
	}

	type exited struct {
		name string
		err  error
	}
	done := make(chan exited, len(services))
	cmds := make([]*exec.Cmd, 0, len(services))

	for _, svc := range services {
		cmd := exec.Command(svc.argv[0], svc.argv[1:]...)
		cmd.Env = os.Environ()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			// Failing to start any service is fatal; tear down the rest.
			os.Stderr.WriteString("supervisor: failed to start " + svc.name + ": " + err.Error() + "\n")
			terminate(cmds)
			os.Exit(1)
		}
		cmds = append(cmds, cmd)
		name := svc.name
		go func() {
			done <- exited{name: name, err: cmd.Wait()}
		}()
	}

	// Forward signals to children so `docker stop` performs a graceful shutdown.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case sig := <-sigs:
			for _, cmd := range cmds {
				if cmd.Process != nil {
					_ = cmd.Process.Signal(sig)
				}
			}
		case e := <-done:
			// Any child exiting takes the container down so it can be restarted
			// cleanly rather than serving with a missing component.
			os.Stderr.WriteString("supervisor: service " + e.name + " exited; shutting down\n")
			terminate(cmds)
			if e.err != nil {
				os.Exit(1)
			}
			os.Exit(0)
		}
	}
}

// lookup resolves name via PATH, falling back to fallback when not found so the
// supervisor still works on minimal images with an empty or unusual PATH.
func lookup(name, fallback string) string {
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return fallback
}

func terminate(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
		}
	}
}
