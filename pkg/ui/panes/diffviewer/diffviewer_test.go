package diffviewer

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/dlvhdr/diffnav/pkg/ui/common"
)

func TestStaleRenderCannotReplaceRefreshedDiff(t *testing.T) {
	common.RegisterSupportedTints()
	styles := common.MakeStyles()
	m := New(false, &styles)
	m.SetSize(100, 40)
	key := cacheKey("/", false)
	oldNode := &cachedNode{path: "/"}
	newNode := &cachedNode{path: "/"}
	m.cache[key] = newNode
	m.dir = newNode
	m, _ = m.Update(
		diffContentMsg{cacheKey: key, node: newNode, lines: stringToDiffLines("fresh content")},
	)
	m, _ = m.Update(
		diffContentMsg{cacheKey: key, node: oldNode, lines: stringToDiffLines("stale content")},
	)
	if got := m.View(); !strings.Contains(got, "fresh content") ||
		strings.Contains(got, "stale content") {
		t.Fatalf("stale render replaced the refreshed diff: %s", got)
	}
}

func TestRenderPreamble_Empty(t *testing.T) {
	common.RegisterSupportedTints()
	m := New(false, &common.Styles{
		Tint:   common.Themes.Current(),
		Colors: common.Colors{},
	})
	if got := m.renderPreamble(""); got != "" {
		t.Fatalf("expected empty string for empty preamble, got %q", got)
	}
	if got := m.renderPreamble("   \n  \n  "); got != "" {
		t.Fatalf("expected empty string for whitespace-only preamble, got %q", got)
	}
}

func TestRenderPreamble_GitShow(t *testing.T) {
	common.RegisterSupportedTints()
	m := New(false, &common.Styles{
		Tint:   common.Themes.Current(),
		Colors: common.Colors{},
	})
	preamble := `commit abc123def456
Author: Jane Doe <jane@example.com>
Date:   Mon Jan 1 00:00:00 2026 +0000

    feat: add new feature

    This is the body of the commit message.`

	got := m.renderPreamble(preamble)
	plain := ansi.Strip(got)

	// All original content lines should be preserved in the output.
	for _, want := range []string{
		"commit abc123def456",
		"Author: Jane Doe <jane@example.com>",
		"Date:   Mon Jan 1 00:00:00 2026 +0000",
		"feat: add new feature",
		"This is the body of the commit message.",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, plain)
		}
	}
}

func TestRenderPreamble_MergeCommit(t *testing.T) {
	preamble := `commit abc123def456
Merge: aaa111 bbb222
Author: Jane Doe <jane@example.com>
Date:   Mon Jan 1 00:00:00 2026 +0000

    Merge branch 'feature' into main`

	common.RegisterSupportedTints()
	m := New(false, &common.Styles{
		Tint:   common.Themes.Current(),
		Colors: common.Colors{},
	})
	got := m.renderPreamble(preamble)
	plain := ansi.Strip(got)

	for _, want := range []string{
		"Merge: aaa111 bbb222",
		"Merge branch 'feature' into main",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, plain)
		}
	}
}
