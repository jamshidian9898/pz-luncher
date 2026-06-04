package contracts

type CacheEntryState string

const (
	CacheEntryStateValid     CacheEntryState = "valid"
	CacheEntryStateInvalid   CacheEntryState = "invalid"
	CacheEntryStateRepairing CacheEntryState = "repairing"
)

type CacheEntry struct {
	Hash           string          `json:"hash"`
	Size           int64           `json:"size"`
	Path           string          `json:"path"`
	ReferenceCount int             `json:"referenceCount"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	LastAccessedAt string          `json:"lastAccessedAt,omitempty"`
	State          CacheEntryState `json:"state"`
}
