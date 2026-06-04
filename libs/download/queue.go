package download

import (
	"time"

	"pzlauncher/libs/modplan"
)

type State string

const (
	StatePending     State = "Pending"
	StateDownloading State = "Downloading"
	StatePaused      State = "Paused"
	StateFailed      State = "Failed"
	StateCompleted   State = "Completed"
)

type Item struct {
	ModID            string `json:"modId"`
	State            State  `json:"state"`
	BytesDone        int64  `json:"bytesDone"`
	BytesTotal       int64  `json:"bytesTotal"`
	SpeedBps         int64  `json:"speedBps,omitempty"`
	ETASeconds       int    `json:"etaSeconds,omitempty"`
	Attempt          int    `json:"attempt"`
	LastError        string `json:"lastError,omitempty"`
	ChecksumExpected string `json:"checksumExpected"`
	ChecksumActual   string `json:"checksumActual,omitempty"`
	LocalPath        string `json:"localPath,omitempty"`
}

type Queue struct {
	SessionID string    `json:"sessionId"`
	ServerID  string    `json:"serverId"`
	Items     []Item    `json:"items"`
	StartedAt time.Time `json:"startedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func NewQueueFromPlan(sessionID, serverID string, plan *modplan.ResolvedModPlan) *Queue {
	now := time.Now()
	items := make([]Item, len(plan.OrderedMods))
	for i, m := range plan.OrderedMods {
		total := m.SizeBytes
		if total == 0 {
			total = 1
		}
		items[i] = Item{
			ModID:            m.ID,
			State:            StatePending,
			BytesTotal:       total,
			ChecksumExpected: m.SHA256,
		}
	}
	return &Queue{
		SessionID: sessionID,
		ServerID:  serverID,
		Items:     items,
		StartedAt: now,
		UpdatedAt: now,
	}
}

func (q *Queue) Find(modID string) *Item {
	for i := range q.Items {
		if q.Items[i].ModID == modID {
			return &q.Items[i]
		}
	}
	return nil
}

func (q *Queue) CompletedCount() int {
	n := 0
	for _, it := range q.Items {
		if it.State == StateCompleted {
			n++
		}
	}
	return n
}

func (q *Queue) OverallPercent() int {
	if len(q.Items) == 0 {
		return 100
	}
	var done, total int64
	for _, it := range q.Items {
		if it.BytesTotal > 0 {
			total += it.BytesTotal
		} else {
			total += 1
		}
		if it.State == StateCompleted {
			done += it.BytesTotal
			if done == 0 {
				done = 1
			}
		} else {
			done += it.BytesDone
		}
	}
	if total == 0 {
		return 0
	}
	return int(done * 100 / total)
}
