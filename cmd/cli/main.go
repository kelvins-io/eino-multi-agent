package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	base := getenv("EINO_API_BASE", "http://127.0.0.1:8180")
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: cli <create|list|show|cancel|confirm> [flags]\n")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		goal := fs.String("goal", "", "task goal")
		policy := fs.String("policy", "on_risk", "confirm policy")
		skills := fs.String("skills", "", "comma-separated skill names")
		file := fs.String("file", "", "upload file")
		_ = fs.Parse(os.Args[2:])
		if *goal == "" {
			fatal("goal is required")
		}
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		_ = w.WriteField("goal", *goal)
		_ = w.WriteField("confirm_policy", *policy)
		_ = w.WriteField("skills", *skills)
		if *file != "" {
			f, err := os.Open(*file)
			if err != nil {
				fatal(err.Error())
			}
			defer f.Close()
			part, err := w.CreateFormFile("files", filepath.Base(*file))
			if err != nil {
				fatal(err.Error())
			}
			if _, err := io.Copy(part, f); err != nil {
				fatal(err.Error())
			}
		}
		_ = w.Close()
		resp := do(http.MethodPost, base+"/api/v1/tasks", w.FormDataContentType(), body)
		fmt.Println(resp)
	case "list":
		fmt.Println(do(http.MethodGet, base+"/api/v1/tasks", "", nil))
	case "show":
		if len(os.Args) < 3 {
			fatal("id is required")
		}
		fmt.Println(do(http.MethodGet, base+"/api/v1/tasks/"+os.Args[2], "", nil))
	case "cancel":
		if len(os.Args) < 3 {
			fatal("id is required")
		}
		fmt.Println(do(http.MethodPost, base+"/api/v1/tasks/"+os.Args[2]+"/cancel", "application/json", strings.NewReader("{}")))
	case "confirm":
		if len(os.Args) < 3 {
			fatal("id is required")
		}
		id := os.Args[2]
		fs := flag.NewFlagSet("confirm", flag.ExitOnError)
		approved := fs.Bool("approved", true, "approve or reject")
		_ = fs.Parse(os.Args[3:])
		payload, _ := json.Marshal(map[string]bool{"approved": *approved})
		fmt.Println(do(http.MethodPost, base+"/api/v1/tasks/"+id+"/confirm", "application/json", bytes.NewReader(payload)))
	default:
		fatal("unknown command")
	}
}

func do(method, url, contentType string, body io.Reader) string {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		fatal(err.Error())
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fatal(err.Error())
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		fatal(fmt.Sprintf("%s: %s", resp.Status, b))
	}
	return string(b)
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
