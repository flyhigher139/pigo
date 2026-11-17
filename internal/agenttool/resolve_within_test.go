package agenttool

import (
	"os"
	"path/filepath"
	"testing"
)

// symlinkFixture builds target/a.txt plus a sibling symlink link → target, so
// tests can spell one workspace through two different path forms.
func symlinkFixture(t *testing.T) (target, link string) {
	t.Helper()
	parent := t.TempDir()
	target = filepath.Join(parent, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "a.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	link = filepath.Join(parent, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	return target, link
}

// TestResolveWithinAcceptsBothSymlinkSpellings: the guard compares physical
// paths, so a workspace spelled through a symlink and through its target is
// one workspace — the model may use either form (macOS's /tmp → /private/tmp
// is the canonical example, and os.Getwd always reports the physical form).
func TestResolveWithinAcceptsBothSymlinkSpellings(t *testing.T) {
	target, link := symlinkFixture(t)
	cases := []struct {
		name string
		root string
		path string
	}{
		{"root as symlink, relative path", link, "a.txt"},
		{"root as symlink, symlink-form abs", link, filepath.Join(link, "a.txt")},
		{"root as symlink, physical abs", link, filepath.Join(target, "a.txt")},
		{"root as target, symlink-form abs", target, filepath.Join(link, "a.txt")},
		{"root as symlink, new file via symlink", link, filepath.Join(link, "new.txt")},
		{"root as target, new file via symlink", target, filepath.Join(link, "new.txt")},
	}
	for _, tc := range cases {
		if _, err := resolveWithin(tc.root, tc.path); err != nil {
			t.Errorf("%s: resolveWithin(%q) = %v, want nil", tc.name, tc.path, err)
		}
	}
}

// TestResolveWithinStillRejectsEscape: the symlink-aware comparison must not
// widen the boundary — lexical traversal and absolute paths outside the root
// stay rejected, and so does a path that only escapes once symlinks resolve.
func TestResolveWithinStillRejectsEscape(t *testing.T) {
	target, link := symlinkFixture(t)
	if err := os.Symlink("/etc", filepath.Join(target, "evil")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	outside := filepath.Join(filepath.Dir(target), "outside.txt")
	if _, err := resolveWithin(link, outside); err == nil {
		t.Errorf("resolveWithin(%q) = nil, want outside-the-root error", outside)
	}
	if _, err := resolveWithin(link, "../../etc/passwd"); err == nil {
		t.Errorf("lexical traversal must stay rejected")
	}
	if _, err := resolveWithin(link, filepath.Join(link, "evil", "passwd")); err == nil {
		t.Errorf("symlink-assisted escape must be rejected")
	}
}
