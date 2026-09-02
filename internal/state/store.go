package state

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const defaultDownloadURL = "https://github.com/XTLS/Xray-core/releases/download"

type Settings struct {
	XrayVersion  string `json:"xray_version"`
	XrayPath     string `json:"xray_path"`
	DownloadURL  string `json:"download_url"`
	GitHubMirror string `json:"github_mirror,omitempty"`
	TestURL      string `json:"test_url"`
	ActiveNodeID string `json:"active_node_id,omitempty"`
	ListenPort   int    `json:"listen_port"`
	GlobalProxy  bool   `json:"global_proxy"`
}

type Runtime struct {
	PID        int       `json:"pid,omitempty"`
	NodeID     string    `json:"node_id,omitempty"`
	ConfigPath string    `json:"config_path,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
}

type Subscription struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Node struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URI            string    `json:"uri"`
	SubscriptionID string    `json:"subscription_id,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Data struct {
	Settings      Settings       `json:"settings"`
	Runtime       Runtime        `json:"runtime"`
	Subscriptions []Subscription `json:"subscriptions"`
	Nodes         []Node         `json:"nodes"`
}

type Store struct {
	path string
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "x-cmd", "config.json"), nil
}

func New(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) RuntimeDir() string {
	return filepath.Join(filepath.Dir(s.path), "runtime")
}

func (s *Store) Load() (Data, error) {
	data := Data{Settings: Settings{DownloadURL: defaultDownloadURL, TestURL: "https://www.gstatic.com/generate_204", ListenPort: 1091}}
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return data, nil
	}
	if err != nil {
		return Data{}, err
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return Data{}, err
	}
	if data.Settings.DownloadURL == "" {
		data.Settings.DownloadURL = defaultDownloadURL
	}
	if data.Settings.TestURL == "" {
		data.Settings.TestURL = "https://www.gstatic.com/generate_204"
	}
	if data.Settings.ListenPort == 0 {
		data.Settings.ListenPort = 1091
	}
	return data, nil
}

func (s *Store) Save(data Data) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	temporary := s.path + ".tmp"
	if err := os.WriteFile(temporary, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, s.path)
}

func NewID() string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err == nil {
		return hex.EncodeToString(buffer)
	}
	return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
}
