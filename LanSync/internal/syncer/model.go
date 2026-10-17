package syncer

import (
	"encoding/json"
	"fmt"
	"time"
)

const MarkdownLimit = 2 << 20

type Settings struct {
	Name      string `json:"name"`
	BackupDir string `json:"backupDir"`
	Port      int    `json:"port"`
	AutoStart bool   `json:"autoStart"`
}
type Peer struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Certificate []byte `json:"certificate,omitempty"`
	Online      bool   `json:"online"`
}
type Task struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PeerID      string `json:"peerId"`
	Root        string `json:"root"`
	RemoteRoot  string `json:"remoteRoot"`
	Coordinator string `json:"coordinator"`
	Paused      bool   `json:"paused"`
	Started     bool   `json:"started"`
}
type File struct {
	Hash     string `json:"hash"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified"`
}
type Manifest struct {
	Files    map[string]File   `json:"files"`
	Ignored  []string          `json:"ignored"`
	Problems map[string]string `json:"problems"`
}
type Baseline struct {
	File    File   `json:"file"`
	Content []byte `json:"content,omitempty"`
}
type Issue struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Local   File   `json:"local"`
	Remote  File   `json:"remote"`
	Copy    string `json:"copy,omitempty"`
}
type Status struct {
	TaskID   string   `json:"taskId"`
	State    string   `json:"state"`
	Message  string   `json:"message"`
	LastSync int64    `json:"lastSync"`
	Done     int      `json:"done"`
	Total    int      `json:"total"`
	Issues   []Issue  `json:"issues"`
	Skipped  []string `json:"skipped"`
}
type Version struct {
	ID        string `json:"id"`
	TaskID    string `json:"taskId"`
	Path      string `json:"path"`
	Hash      string `json:"hash"`
	Size      int64  `json:"size"`
	Time      int64  `json:"time"`
	Reason    string `json:"reason"`
	BackupDir string `json:"backupDir,omitempty"`
}
type Invitation struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PeerID      string `json:"peerId"`
	RemoteRoot  string `json:"remoteRoot"`
	Coordinator string `json:"coordinator"`
}
type Pairing struct {
	ID             string `json:"id"`
	PeerID         string `json:"peerId"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	Code           string `json:"code"`
	LocalApproved  bool   `json:"localApproved"`
	RemoteApproved bool   `json:"remoteApproved"`
	Expires        int64  `json:"expires"`
	Certificate    []byte `json:"-"`
}
type Device struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Seen    int64  `json:"seen"`
}
type TaskView struct {
	Task
	Status   Status `json:"status"`
	PeerName string `json:"peerName"`
}
type Snapshot struct {
	ID           string       `json:"id"`
	Settings     Settings     `json:"settings"`
	Addresses    []string     `json:"addresses"`
	Devices      []Device     `json:"devices"`
	Peers        []Peer       `json:"peers"`
	Pairings     []Pairing    `json:"pairings"`
	Invitations  []Invitation `json:"invitations"`
	Tasks        []TaskView   `json:"tasks"`
	NetworkError string       `json:"networkError"`
}
type Preview struct {
	AddedLocal  int `json:"addedLocal"`
	AddedRemote int `json:"addedRemote"`
	Changed     int `json:"changed"`
	Decisions   int `json:"decisions"`
}
type Comparison struct {
	Path       string `json:"path"`
	Local      string `json:"local"`
	Remote     string `json:"remote"`
	LocalHash  string `json:"localHash"`
	RemoteHash string `json:"remoteHash"`
}

func now() int64 { return time.Now().UnixMilli() }
func encode(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
func wrap(action string, err error) error { return fmt.Errorf("%s：%w", action, err) }
