package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"

	"github.com/tydar/smallurl-ccask/ccask"
)

const MAX_KEY_SIZE = 1024

type Env struct {
	shortLinks ShortLinkRepository
	templates  map[string]*template.Template
}

func NewEnv(client *ccask.CCaskClient) *Env {
	return &Env{shortLinks: NewShortLinkModel(client)}
}

func (e *Env) AddTemplate(key string, files ...string) error {
	_, prs := e.templates[key]
	if prs {
		return fmt.Errorf("template with name %s already exists", key)
	}

	e.templates[key] = template.Must(template.ParseFiles(files...))
	return nil
}

func (e *Env) ExecuteTemplate(key string, w http.ResponseWriter, data interface{}) error {
	return e.templates[key].ExecuteTemplate(w, "base", data)
}

func (e *Env) SetURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		if err := e.ExecuteTemplate("set", w, nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	} else if r.Method == "POST" {
		keyStr := r.FormValue("key")
		urlStr := r.FormValue("url")
		url, err := url.ParseRequestURI(urlStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if keyStr == "" {
			http.Error(w, "no key provided", http.StatusBadRequest)
			return
		}

		if len(keyStr) > MAX_KEY_SIZE {
			http.Error(w, "key too long", http.StatusBadRequest)
			return
		}

		if err := e.shortLinks.SetLink(keyStr, url.String()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		successData := struct {
			Key string
			URL string
		}{
			Key: keyStr,
			URL: url.String(),
		}
		if err := e.ExecuteTemplate("set_success", w, successData); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		return
	} else {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
}

func (e *Env) GetURLHandler(w http.ResponseWriter, r *http.Request) {
	keySlice := r.URL.Path[len("/q/"):]
	if r.Method == "GET" {
		sl, err := e.shortLinks.GetLink(keySlice)
		if err != nil {
			http.Error(w, fmt.Sprintf("internal error %s", err.Error()), http.StatusInternalServerError)
			return
		}

		newUrl := sl.URL
		http.Redirect(w, r, newUrl, http.StatusFound)
		return
	} else {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
}
