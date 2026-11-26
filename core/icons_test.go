package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIconMarkupIsEscaped(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "assets", "icons"), 0700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", "icons", "ac.png"), []byte("image"), 0600))
	payload := `<img src=x onerror="alert(1)">`
	require.NotContains(t, string(RenderIcon(payload, "label", root)), "<img")
	require.Contains(t, string(RenderIcon(payload, "label", root)), "&lt;img")
	markup := string(RenderIcon("ac.png", `" onload="alert(1)`, root))
	require.Contains(t, markup, `src="/assets/icons/ac.png"`)
	require.Contains(t, markup, `alt="&#34; onload=&#34;alert(1)"`)
	require.NotContains(t, string(RenderIcon("../ac.png", "label", root)), "<img")
	require.Equal(t, "💡", string(RenderIcon("💡", "label", root)))
	require.NoError(t, os.Symlink(filepath.Join(root, "assets", "icons", "ac.png"), filepath.Join(root, "assets", "icons", "link.png")))
	require.NotContains(t, string(RenderIcon("link.png", "label", root)), "<img")
}
