package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644)
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "init")
	return dir
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %s: %v", name, args, string(out), err)
	}
}

func TestRepoRoot(t *testing.T) {
	dir := initTestRepo(t)
	root, err := RepoRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestCurrentBranch(t *testing.T) {
	dir := initTestRepo(t)
	branch, err := CurrentBranch(dir)
	if err != nil {
		t.Fatal(err)
	}
	if branch != "main" && branch != "master" {
		t.Errorf("branch = %q, want main or master", branch)
	}
}

func TestHasUncommittedChanges(t *testing.T) {
	dir := initTestRepo(t)

	has, err := HasUncommittedChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Error("expected no uncommitted changes")
	}

	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("new"), 0644)
	run(t, dir, "git", "add", ".")
	has, err = HasUncommittedChanges(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Error("expected uncommitted changes")
	}
}

func TestAddAndCommit(t *testing.T) {
	dir := initTestRepo(t)

	// Nothing to commit
	sha, err := AddAndCommit(dir, "empty")
	if err != nil {
		t.Fatal(err)
	}
	if sha != "" {
		t.Error("expected empty sha for no changes")
	}

	// With changes
	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	sha, err = AddAndCommit(dir, "add file")
	if err != nil {
		t.Fatal(err)
	}
	if sha == "" {
		t.Error("expected non-empty sha")
	}
}

func TestConflictError(t *testing.T) {
	e := &ConflictError{Files: []string{"a.go", "b.go"}}
	if e.Error() == "" {
		t.Error("expected non-empty error message")
	}
}
