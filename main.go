package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/go-playground/webhooks/v6/github"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	Name     string `yaml:"name"`
	Branch   string `yaml:"branch"`
	Location string `yaml:"location"`
	Remote   string `yaml:"remote"`
}

type HttpServer struct {
	Services []Service
	Hook     *github.Webhook
}

func pullRepo(remote string, path string) error {
	cmd := exec.Command("git", "-C", path, "pull", remote)
	var b bytes.Buffer
	cmd.Stdout = &b
	if err := cmd.Run(); err != nil {
		return err
	}
	log.Printf("git pull: %s", b.String())
	return nil
}

func (s *HttpServer) pullRepoHandler(w http.ResponseWriter, r *http.Request) {
	payloadI, err := s.Hook.Parse(r, github.PushEvent)
	if err != nil {
		log.Println("Hook parse error:", err)
		return
	}

	payload := payloadI.(github.PushPayload)

	for _, service := range s.Services {
		if payload.Repository.Name == service.Name {
			refParts := strings.Split(payload.Ref, "/")
			branch := refParts[len(refParts)-1]
			if branch == service.Branch {
				if err := pullRepo(service.Remote, service.Location); err != nil {
					log.Println("Pull repo error:", err)
					return
				}
				return
			}
		}
	}
	log.Println("nothing happened")
}

func (s *HttpServer) Run() error {
	http.HandleFunc("/", s.pullRepoHandler)
	return http.ListenAndServe(":8502", nil)
}

func main() {
	confPath := os.Getenv("GHOOK_CONFIG")
	if confPath == "" {
		panic("env GHOOK_CONFIG is empty")
	}

	f, err := os.Open(confPath)
	if err != nil {
		panic(err)
	}
	var config Config
	if err := yaml.NewDecoder(f).Decode(&config); err != nil {
		panic(err)
	}

	hook, _ := github.New()

	s := HttpServer{Services: config.Services, Hook: hook}
	if err := s.Run(); err != nil {
		panic(err)
	}
}
