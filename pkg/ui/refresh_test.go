package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func refreshKey() tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: "r", Code: 'r'})
}

func TestManualRefreshUpdatesContentAndRetainsSelection(t *testing.T) {
	stubRefreshDelta(t)
	m := newTestMainModel(t)
	m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.fileTree.SetCursorByPath("yarn.lock")
	m, render := m.setNodeDiff(m.fileTree.GetCurrNode())
	m = runRefreshRender(t, m, render)
	updatedDiff := "diff --git a/yarn.lock b/yarn.lock\n--- a/yarn.lock\n+++ b/yarn.lock\n@@ -1 +1 @@\n-old\n+refreshedcontent\n"
	updatedDiff += strings.ReplaceAll(updatedDiff, "yarn.lock", "fresh.lock")
	path := filepath.Join(t.TempDir(), "changes.diff")
	if err := os.WriteFile(path, []byte(updatedDiff), 0o600); err != nil {
		t.Fatal(err)
	}
	m.refreshCmd = fmt.Sprintf("cat '%s'", path)
	updated, fetch := m.Update(refreshKey())
	m = updated.(mainModel)
	if fetch == nil || !m.refreshInFlight {
		t.Fatal("r did not start refresh")
	}
	if _, duplicate := m.Update(refreshKey()); duplicate != nil {
		t.Fatal("repeated r started an overlapping refresh")
	}
	result := fetch().(refreshResultMsg)
	if result.err != nil {
		t.Fatal(result.err)
	}
	updated, render = m.Update(result)
	m = updated.(mainModel)
	if m.refreshInFlight || m.diff != updatedDiff {
		t.Fatal("refresh did not apply new diff")
	}
	if got := m.fileTree.CurrNodePath(); got != "yarn.lock" {
		t.Fatalf("selected file = %q, want yarn.lock", got)
	}
	if len(m.files) != len(result.parsed.files) ||
		!strings.Contains(m.fileTree.View(), "fresh.lock") {
		t.Fatal("file tree did not reflect the new diff")
	}
	m = runRefreshRender(t, m, render)
	if !strings.Contains(m.diffViewer.View(), "refreshedcontent") {
		t.Fatal("selected file still shows the cached diff")
	}
	m.fileTree.GoToTop()
	m, render = m.setNodeDiff(m.fileTree.GetCurrNode())
	m = runRefreshRender(t, m, render)
	if !strings.Contains(m.diffViewer.View(), "fresh.lock") {
		t.Fatal("root diff was not rendered after leaving the selected file")
	}
}

func TestRefreshEmptyDiffKeepsViewerOpenAndCanRecover(t *testing.T) {
	stubRefreshDelta(t)
	m := newTestMainModel(t)
	m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.fileTree.SetCursorByPath("yarn.lock")
	oldDiff := m.diff
	for _, input := range []string{"", oldDiff} {
		parsed, err := parseDiffInput(input)
		if err != nil {
			t.Fatal(err)
		}
		m.refreshCmd = "git diff"
		m.refreshInFlight = true
		updated, render := m.Update(refreshResultMsg{output: input, parsed: parsed})
		m = runRefreshRender(t, updated.(mainModel), render)
		if m.diff != input || m.refreshInFlight || len(m.files) != len(parsed.files) {
			t.Fatal("failed to apply empty/non-empty diff")
		}
	}
}

func TestRefreshFailurePreservesDiffAndAllowsRetry(t *testing.T) {
	for _, command := range []string{"false", "printf 'diff --git a/x b/x\\n--- a/x\\n+++ b/x\\n@@ -1 +1 @@\\n'"} {
		t.Run(command, func(t *testing.T) {
			m := newTestMainModel(t)
			oldDiff := m.diff
			m.refreshCmd = command
			updated, fetch := m.Update(refreshKey())
			m = updated.(mainModel)
			result := fetch().(refreshResultMsg)
			if result.err == nil {
				t.Fatal("expected refresh error")
			}
			updated, cmd := m.Update(result)
			m = updated.(mainModel)
			if cmd != nil || m.refreshInFlight || m.diff != oldDiff || m.refreshError == "" {
				t.Fatal("failed refresh must preserve the old diff and show an error")
			}
			if _, retry := m.Update(refreshKey()); retry == nil {
				t.Fatal("cannot retry failed refresh")
			}
		})
	}
}

func TestRefreshKeyRespectsSearchAndOverlays(t *testing.T) {
	for _, mode := range []string{"stdin", "file search", "diff search", "help", "message", "theme"} {
		t.Run(mode, func(t *testing.T) {
			m := newTestMainModel(t)
			m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
			m.refreshCmd = "git diff"
			switch mode {
			case "stdin":
				m.refreshCmd = ""
			case "file search":
				m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "t", Code: 't'}))
			case "diff search":
				m = updateMainModel(t, m, tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}))
			case "help":
				m.helpOpen = true
			case "message":
				m.messageOpen = true
			case "theme":
				m.themePickerOpen = true
			}
			m = updateMainModel(t, m, refreshKey())
			if m.refreshInFlight {
				t.Fatal("r should not trigger refresh in this context")
			}
			if mode == "file search" && m.filesSearch.Value() != "r" {
				t.Fatal("r was not entered in the search field")
			}
		})
	}
}

func TestRefreshResultIsHandledWhileSearching(t *testing.T) {
	for _, searchKey := range []rune{'t', '/'} {
		t.Run(string(searchKey), func(t *testing.T) {
			m := newTestMainModel(t)
			m = updateMainModel(t, m, tea.WindowSizeMsg{Width: 100, Height: 40})
			m.refreshCmd = "git diff"
			m.refreshInFlight = true
			m = updateMainModel(
				t,
				m,
				tea.KeyPressMsg(tea.Key{Text: string(searchKey), Code: searchKey}),
			)
			m = updateMainModel(t, m, refreshResultMsg{})
			if m.refreshInFlight || m.diff != "" || len(m.files) != 0 {
				t.Fatal("search swallowed refresh result")
			}
			if searchKey == 't' && len(m.filtered) != 0 {
				t.Fatal("file search results were not updated")
			}
		})
	}
}

func TestWatchTimerDoesNotDuplicateOrOverlapManualRefresh(t *testing.T) {
	m := newTestMainModel(t)
	m.refreshCmd = "git diff"
	m.watchEnabled = true
	m.watchInterval = time.Millisecond
	updated, _ := m.Update(refreshKey())
	m = updated.(mainModel)
	updated, tickCmd := m.Update(watchTickMsg{})
	m = updated.(mainModel)
	if tickCmd == nil || !m.refreshInFlight {
		t.Fatal("watch tick must keep the timer alive during manual refresh")
	}
	if _, ok := tickCmd().(watchTickMsg); !ok {
		t.Fatal("watch tick started an overlapping fetch instead of only scheduling a timer")
	}
	updated, cmd := m.Update(refreshResultMsg{output: m.diff})
	m = updated.(mainModel)
	if cmd != nil || m.refreshInFlight {
		t.Fatal("unchanged refresh must finish without adding a timer")
	}
	_, cmd = m.Update(watchTickMsg{})
	if batch, ok := cmd().(tea.BatchMsg); !ok || len(batch) != 2 {
		t.Fatal("next watch tick must schedule a timer and fetch")
	}
}

// Keep model tests independent of a locally installed delta binary.
func stubRefreshDelta(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, "delta"),
		[]byte("#!/bin/sh\ncat\n"),
		0o700,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// Run only the rendering commands returned after a refresh (no watch timers).
func runRefreshRender(t *testing.T, m mainModel, cmd tea.Cmd) mainModel {
	t.Helper()
	if cmd == nil {
		return m
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		for _, next := range batch {
			m = runRefreshRender(t, m, next)
		}
		return m
	}
	if _, quit := msg.(tea.QuitMsg); quit {
		t.Fatal("refresh unexpectedly quit the viewer")
	}
	updated, next := m.Update(msg)
	return runRefreshRender(t, updated.(mainModel), next)
}
