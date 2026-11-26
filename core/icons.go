package core

import (
	"html/template"
	"net/url"
	"os"
	"path/filepath"
)

func RenderIcon(icon, label, webroot string) template.HTML {
	if icon != "" && filepath.Base(icon) == icon && icon != "." && icon != ".." {
		info, err := os.Lstat(filepath.Join(webroot, "assets", "icons", icon))
		if err == nil && info.Mode().IsRegular() {
			return template.HTML(`<img class="icon" src="` + template.HTMLEscapeString("/assets/icons/"+url.PathEscape(icon)) + `" alt="` + template.HTMLEscapeString(label) + `">`)
		}
	}
	if icon == "" {
		icon = label
	}
	return template.HTML(template.HTMLEscapeString(icon))
}
