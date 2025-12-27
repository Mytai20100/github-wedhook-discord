package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`
	Discord struct {
		WebhookURL string `yaml:"webhook_url"`
	} `yaml:"discord"`
}

type DiscordMessage struct {
	Content string                   `json:"content,omitempty"`
	Embeds  []map[string]interface{} `json:"embeds,omitempty"`
}

var config Config

func main() {
	loadConfig()

	http.HandleFunc("/webhook", handleWebhook)
	http.HandleFunc("/", handleRoot)

	addr := fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port)
	log.Printf("Server starting on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func loadConfig() {
	if _, err := os.Stat("config.yml"); os.IsNotExist(err) {
		createDefaultConfig()
	}

	data, err := os.ReadFile("config.yml")
	if err != nil {
		log.Fatal("Error reading config:", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("Error parsing config:", err)
	}

	if config.Discord.WebhookURL == "" {
		log.Fatal("Discord webhook URL not configured")
	}
}

func createDefaultConfig() {
	defaultConfig := `server:
  host: "0.0.0.0"
  port: 8080

discord:
  webhook_url: "https://discord.com/api/webhooks/YOUR_WEBHOOK_URL"
`
	err := os.WriteFile("config.yml", []byte(defaultConfig), 0644)
	if err != nil {
		log.Fatal("Error creating config:", err)
	}
	log.Println("Created config.yml - Please update with your Discord webhook URL")
	os.Exit(0)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("GitHub Webhook Discord Bridge"))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	event := r.Header.Get("X-GitHub-Event")
	delivery := r.Header.Get("X-GitHub-Delivery")
	log.Printf("Received event: %s, delivery: %s", event, delivery)

	if event == "" {
		log.Println("Missing X-GitHub-Event header")
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if event == "ping" {
		log.Println("Ping event received")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
		return
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("Error parsing JSON: %v, body: %s", err, string(body))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	discordMsg := convertToDiscord(event, payload)

	if err := sendToDiscord(discordMsg); err != nil {
		log.Printf("Error sending to Discord: %v", err)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	log.Printf("Event %s processed successfully", event)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func convertToDiscord(event string, payload map[string]interface{}) DiscordMessage {
	embed := make(map[string]interface{})

	switch event {
	case "push":
		handlePushEvent(embed, payload)
	case "pull_request":
		handlePullRequestEvent(embed, payload)
	case "issues":
		handleIssuesEvent(embed, payload)
	case "issue_comment":
		handleIssueCommentEvent(embed, payload)
	case "pull_request_review_comment":
		handlePRReviewCommentEvent(embed, payload)
	case "pull_request_review":
		handlePRReviewEvent(embed, payload)
	case "star":
		handleStarEvent(embed, payload)
	case "fork":
		handleForkEvent(embed, payload)
	case "create":
		handleCreateEvent(embed, payload)
	case "delete":
		handleDeleteEvent(embed, payload)
	default:
		handleDefaultEvent(embed, event, payload)
	}

	return DiscordMessage{Embeds: []map[string]interface{}{embed}}
}

func handlePushEvent(embed map[string]interface{}, payload map[string]interface{}) {
	embed["color"] = 0x7289DA
	
	ref := getStr(payload, "ref")
	repo := getStr(payload, "repository", "full_name")
	getStr(payload, "repository", "html_url")
	compareURL := getStr(payload, "compare")
	
	pusher := getStr(payload, "pusher", "name")
	pusherURL := fmt.Sprintf("https://github.com/%s", pusher)
	pusherAvatar := getStr(payload, "sender", "avatar_url")

	branch := ref
	if len(ref) > 11 && ref[:11] == "refs/heads/" {
		branch = ref[11:]
	}

	commitCount := 0
	commitText := ""
	if commits, ok := payload["commits"].([]interface{}); ok {
		commitCount = len(commits)
		if commitCount == 1 {
			commitText = "1 new commit"
		} else {
			commitText = fmt.Sprintf("%d new commits", commitCount)
		}
	}

	embed["author"] = map[string]string{
		"name":     pusher,
		"url":      pusherURL,
		"icon_url": pusherAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s:%s] %s", repo, branch, commitText)
	embed["url"] = compareURL
	
	if commits, ok := payload["commits"].([]interface{}); ok && len(commits) > 0 {
		description := ""
		maxCommits := 5
		for i, commit := range commits {
			if i >= maxCommits {
				break
			}
			c := commit.(map[string]interface{})
			sha := getStr(c, "id")
			if len(sha) > 7 {
				sha = sha[:7]
			}
			msg := getStr(c, "message")
			author := getStr(c, "author", "name")
			description += fmt.Sprintf("`%s` %s - %s\n", sha, msg, author)
		}
		if len(commits) > maxCommits {
			description += fmt.Sprintf("... and %d more commits", len(commits)-maxCommits)
		}
		embed["description"] = description
	}
}

func handlePullRequestEvent(embed map[string]interface{}, payload map[string]interface{}) {
	action := getStr(payload, "action")
	
	if action == "opened" || action == "reopened" {
		embed["color"] = 0x28A745
	} else if action == "closed" {
		embed["color"] = 0x6E7681
	} else {
		embed["color"] = 0xFF69B4
	}
	
	pr := payload["pull_request"].(map[string]interface{})
	number := int(getFloat(pr, "number"))
	title := getStr(pr, "title")
	url := getStr(pr, "html_url")
	user := getStr(pr, "user", "login")
	userURL := getStr(pr, "user", "html_url")
	userAvatar := getStr(pr, "user", "avatar_url")
	repo := getStr(payload, "repository", "full_name")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] Pull request #%d %s: %s", repo, number, action, title)
	embed["url"] = url
}

func handleIssuesEvent(embed map[string]interface{}, payload map[string]interface{}) {
	embed["color"] = 0xDC143C
	
	action := getStr(payload, "action")
	issue := payload["issue"].(map[string]interface{})
	number := int(getFloat(issue, "number"))
	title := getStr(issue, "title")
	url := getStr(issue, "html_url")
	user := getStr(issue, "user", "login")
	userURL := getStr(issue, "user", "html_url")
	userAvatar := getStr(issue, "user", "avatar_url")
	repo := getStr(payload, "repository", "full_name")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] Issue #%d %s: %s", repo, number, action, title)
	embed["url"] = url
}

func handleIssueCommentEvent(embed map[string]interface{}, payload map[string]interface{}) {
	embed["color"] = 0xFF69B4
	
	getStr(payload, "action")
	issue := payload["issue"].(map[string]interface{})
	comment := payload["comment"].(map[string]interface{})
	
	number := int(getFloat(issue, "number"))
	title := getStr(issue, "title")
	url := getStr(comment, "html_url")
	user := getStr(comment, "user", "login")
	userURL := getStr(comment, "user", "html_url")
	userAvatar := getStr(comment, "user", "avatar_url")
	repo := getStr(payload, "repository", "full_name")
	body := getStr(comment, "body")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] New comment on issue #%d: %s", repo, number, title)
	embed["url"] = url
	
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	embed["description"] = body
}

func handlePRReviewCommentEvent(embed map[string]interface{}, payload map[string]interface{}) {
	embed["color"] = 0xFFFFFF
	
	getStr(payload, "action")
	pr := payload["pull_request"].(map[string]interface{})
	comment := payload["comment"].(map[string]interface{})
	
	number := int(getFloat(pr, "number"))
	title := getStr(pr, "title")
	url := getStr(comment, "html_url")
	user := getStr(comment, "user", "login")
	userURL := getStr(comment, "user", "html_url")
	userAvatar := getStr(comment, "user", "avatar_url")
	repo := getStr(payload, "repository", "full_name")
	body := getStr(comment, "body")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] New comment on pull request #%d: %s", repo, number, title)
	embed["url"] = url
	
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	embed["description"] = body
}

func handlePRReviewEvent(embed map[string]interface{}, payload map[string]interface{}) {
	embed["color"] = 0x90EE90
	
	getStr(payload, "action")
	pr := payload["pull_request"].(map[string]interface{})
	review := payload["review"].(map[string]interface{})
	
	number := int(getFloat(pr, "number"))
	title := getStr(pr, "title")
	url := getStr(review, "html_url")
	user := getStr(review, "user", "login")
	userURL := getStr(review, "user", "html_url")
	userAvatar := getStr(review, "user", "avatar_url")
	repo := getStr(payload, "repository", "full_name")
	state := getStr(review, "state")
	body := getStr(review, "body")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] Pull request review %s on #%d: %s", repo, state, number, title)
	embed["url"] = url
	
	if body != "" {
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		embed["description"] = body
	}
}

func handleStarEvent(embed map[string]interface{}, payload map[string]interface{}) {
	action := getStr(payload, "action")
	repo := getStr(payload, "repository", "full_name")
	repoURL := getStr(payload, "repository", "html_url")
	user := getStr(payload, "sender", "login")
	userURL := getStr(payload, "sender", "html_url")
	userAvatar := getStr(payload, "sender", "avatar_url")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	
	if action == "created" {
		embed["title"] = fmt.Sprintf("[%s] New star", repo)
	} else {
		embed["title"] = fmt.Sprintf("[%s] Star removed", repo)
	}
	embed["url"] = repoURL
}

func handleForkEvent(embed map[string]interface{}, payload map[string]interface{}) {
	repo := getStr(payload, "repository", "full_name")
	getStr(payload, "repository", "html_url")
	user := getStr(payload, "sender", "login")
	userURL := getStr(payload, "sender", "html_url")
	userAvatar := getStr(payload, "sender", "avatar_url")
	forkee := payload["forkee"].(map[string]interface{})
	forkURL := getStr(forkee, "html_url")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] Forked", repo)
	embed["url"] = forkURL
	embed["description"] = fmt.Sprintf("Fork: %s", forkURL)
}

func handleCreateEvent(embed map[string]interface{}, payload map[string]interface{}) {
	refType := getStr(payload, "ref_type")
	ref := getStr(payload, "ref")
	repo := getStr(payload, "repository", "full_name")
	repoURL := getStr(payload, "repository", "html_url")
	user := getStr(payload, "sender", "login")
	userURL := getStr(payload, "sender", "html_url")
	userAvatar := getStr(payload, "sender", "avatar_url")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] Created %s: %s", repo, refType, ref)
	embed["url"] = repoURL
}

func handleDeleteEvent(embed map[string]interface{}, payload map[string]interface{}) {
	refType := getStr(payload, "ref_type")
	ref := getStr(payload, "ref")
	repo := getStr(payload, "repository", "full_name")
	repoURL := getStr(payload, "repository", "html_url")
	user := getStr(payload, "sender", "login")
	userURL := getStr(payload, "sender", "html_url")
	userAvatar := getStr(payload, "sender", "avatar_url")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] Deleted %s: %s", repo, refType, ref)
	embed["url"] = repoURL
}

func handleDefaultEvent(embed map[string]interface{}, event string, payload map[string]interface{}) {
	repo := getStr(payload, "repository", "full_name")
	repoURL := getStr(payload, "repository", "html_url")
	user := getStr(payload, "sender", "login")
	userURL := getStr(payload, "sender", "html_url")
	userAvatar := getStr(payload, "sender", "avatar_url")

	embed["author"] = map[string]string{
		"name":     user,
		"url":      userURL,
		"icon_url": userAvatar,
	}
	embed["title"] = fmt.Sprintf("[%s] %s event", repo, event)
	embed["url"] = repoURL
}

func getStr(data map[string]interface{}, keys ...string) string {
	var current interface{} = data
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return ""
		}
	}
	if s, ok := current.(string); ok {
		return s
	}
	return ""
}

func getFloat(data map[string]interface{}, keys ...string) float64 {
	var current interface{} = data
	for _, key := range keys {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[key]
		} else {
			return 0
		}
	}
	if f, ok := current.(float64); ok {
		return f
	}
	return 0
}

func sendToDiscord(msg DiscordMessage) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal error: %v", err)
	}

	log.Printf("Sending to Discord: %s", string(jsonData))

	resp, err := http.Post(config.Discord.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Discord response: %d", resp.StatusCode)
	return nil
}
