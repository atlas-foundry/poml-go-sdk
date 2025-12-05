package langgraph

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecuteGraphWithDummyPlugin(t *testing.T) {
	tmp := t.TempDir()
	pluginPath := filepath.Join(tmp, "plugin-dummy")
	if runtime.GOOS == "windows" {
		pluginPath += ".exe"
	}
	if out, err := runCmd(t, "go", "build", "-o", pluginPath, "./cmd/plugin-dummy"); err != nil {
		t.Fatalf("build plugin: %v\n%s", err, string(out))
	}
	exec, err := LoadTransport(pluginPath)
	if err != nil {
		t.Fatalf("load transport: %v", err)
	}
	defer exec.Close()
	if err := exec.LoadMeshLogger(pluginPath); err != nil {
		t.Fatalf("load mesh: %v", err)
	}
	g := Graph{
		Nodes: []Node{
			{Type: "mesh_log", Msg: "start"},
			{Type: "transport_dial", Addr: "127.0.0.1:443"},
			{Type: "mesh_log", Msg: "done"},
		},
	}
	if err := exec.ExecuteGraph(g); err != nil {
		t.Fatalf("execute graph: %v", err)
	}
}

func runCmd(t *testing.T, cmd string, args ...string) ([]byte, error) {
	t.Helper()
	c := exec.Command(cmd, args...)
	c.Env = append(os.Environ(), "GO111MODULE=on")
	return c.CombinedOutput()
}

func TestExecuteGraphEmpty(t *testing.T) {
	e := &Executor{}
	g := Graph{Nodes: []Node{}}
	if err := e.ExecuteGraph(g); err != nil {
		t.Errorf("empty graph should succeed: %v", err)
	}
}

func TestExecuteGraphUnknownNodeType(t *testing.T) {
	e := &Executor{}
	g := Graph{Nodes: []Node{{Type: "unknown"}}}
	if err := e.ExecuteGraph(g); err != nil {
		t.Errorf("unknown node type should be ignored: %v", err)
	}
}

func TestExecuteTransportPingNoTransport(t *testing.T) {
	e := &Executor{}
	if err := e.ExecuteTransportPing("localhost:8080"); err != nil {
		t.Errorf("should succeed when no transport: %v", err)
	}
}

func TestExecuteGraphMeshLogNoMesh(t *testing.T) {
	e := &Executor{}
	g := Graph{Nodes: []Node{{Type: "mesh_log", Msg: "test"}}}
	if err := e.ExecuteGraph(g); err != nil {
		t.Errorf("mesh_log without mesh should succeed: %v", err)
	}
}

func TestExecutorClose(t *testing.T) {
	e := &Executor{clients: nil}
	e.Close() // Should not panic with nil clients
}

func TestLoadTransportInvalidPath(t *testing.T) {
	_, err := LoadTransport("/nonexistent/plugin")
	if err == nil {
		t.Error("expected error for invalid plugin path")
	}
}
