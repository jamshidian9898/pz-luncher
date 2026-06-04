package session

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// SteamAPIClient handles Steam Web API interactions for Workshop content
type SteamAPIClient struct {
	APIKey     string
	HTTPClient *http.Client
}

// NewSteamAPIClient creates a new Steam API client
func NewSteamAPIClient(apiKey string) *SteamAPIClient {
	return &SteamAPIClient{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// WorkshopItem represents a Steam Workshop item metadata
type WorkshopItem struct {
	PublishedFileID string `json:"publishedfileid"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	FileURL         string `json:"file_url"`         // Direct download URL (if public)
	PreviewURL      string `json:"preview_url"`
	FileSize        int64  `json:"file_size"`
	TimeCreated     int64  `json:"time_created"`
	TimeUpdated     int64  `json:"time_updated"`
	Visibility      int    `json:"visibility"` // 0=public, 1=friends, 2=private
	Banned          bool   `json:"banned"`
	Accepted        bool   `json:"accepted"`
}

// ResolveWorkshopItem fetches metadata for a Workshop item
// Returns the item details including download URL if available
func (c *SteamAPIClient) ResolveWorkshopItem(workshopID string) (*WorkshopItem, error) {
	// Steam Web API endpoint for published file details
	endpoint := "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/"

	// Build form data
	data := url.Values{}
	data.Set("itemcount", "1")
	data.Set("publishedfileids[0]", workshopID)
	if c.APIKey != "" {
		data.Set("key", c.APIKey)
	}

	resp, err := c.HTTPClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("steam api request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("steam api error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Response struct {
			Result         int            `json:"result"`
			ResultCount    int            `json:"resultcount"`
			PublishedFiles []WorkshopItem `json:"publishedfiledetails"`
		} `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode steam api response: %w", err)
	}

	if result.Response.Result != 1 {
		return nil, fmt.Errorf("steam api returned error code: %d", result.Response.Result)
	}

	if result.Response.ResultCount == 0 {
		return nil, fmt.Errorf("workshop item not found: %s", workshopID)
	}

	item := &result.Response.PublishedFiles[0]

	// Validate the item
	if item.Banned {
		return nil, fmt.Errorf("workshop item %s is banned", workshopID)
	}

	if item.Visibility != 0 {
		return nil, fmt.Errorf("workshop item %s is not public (visibility: %d)", workshopID, item.Visibility)
	}

	return item, nil
}

// GetDownloadURL returns the direct download URL for a workshop item
// Some items require SteamCMD if file_url is empty
func (c *SteamAPIClient) GetDownloadURL(workshopID string) (string, int64, error) {
	item, err := c.ResolveWorkshopItem(workshopID)
	if err != nil {
		return "", 0, err
	}

	// If file_url is empty, this item requires SteamCMD
	if item.FileURL == "" {
		return "", item.FileSize, fmt.Errorf("item requires steamcmd: no direct download url")
	}

	return item.FileURL, item.FileSize, nil
}

// ResolveWorkshopID attempts to resolve a mod ID to Steam Workshop ID
// In production, this queries a mapping database or registry
func (c *SteamAPIClient) ResolveWorkshopID(modID, version string) (string, error) {
	// Placeholder: In real implementation, this queries a database
	// e.g., "Brita" → "123456789"
	// For now, assume modID is the workshop ID if it's numeric
	
	if _, err := strconv.ParseInt(modID, 10, 64); err == nil {
		return modID, nil // Already a workshop ID
	}

	// TODO: Query mapping service
	// This would call a registry service that maps mod names to Workshop IDs
	return "", fmt.Errorf("cannot resolve mod %s to workshop id: mapping service not implemented", modID)
}

// IsAvailable checks if Steam API is accessible
func (c *SteamAPIClient) IsAvailable() bool {
	resp, err := c.HTTPClient.Get("https://api.steampowered.com/ISteamWebAPIUtil/GetSupportedAPIList/v1/")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
