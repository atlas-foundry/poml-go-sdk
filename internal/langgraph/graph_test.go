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
