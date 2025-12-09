package curseforge

import "time"

const (
	BaseURL   = "https://api.curseforge.com"
	GameIDWoW = 1
)

// SearchModsResponse is the response from /v1/mods/search
type SearchModsResponse struct {
	Data       []Mod      `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination info from API responses
type Pagination struct {
	Index       int `json:"index"`
	PageSize    int `json:"pageSize"`
	ResultCount int `json:"resultCount"`
	TotalCount  int `json:"totalCount"`
}

// Mod represents a CurseForge addon/mod
type Mod struct {
	ID                   int        `json:"id"`
	GameID               int        `json:"gameId"`
	Name                 string     `json:"name"`
	Slug                 string     `json:"slug"`
	Summary              string     `json:"summary"`
	DownloadCount        int64      `json:"downloadCount"`
	ThumbsUpCount        int        `json:"thumbsUpCount"`
	Rating               float64    `json:"rating"`
	PopularityRank       int        `json:"popularityRank"`
	DateCreated          time.Time  `json:"dateCreated"`
	DateModified         time.Time  `json:"dateModified"`
	DateReleased         time.Time  `json:"dateReleased"`
	Categories           []Category `json:"categories"`
	Authors              []Author   `json:"authors"`
	Logo                 *Logo      `json:"logo"`
	LatestFiles          []File     `json:"latestFiles"`
	LatestFilesIndexes   []FileIndex `json:"latestFilesIndexes"`
}

// Category represents an addon category
type Category struct {
	ID       int    `json:"id"`
	GameID   int    `json:"gameId"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	URL      string `json:"url"`
	IconURL  string `json:"iconUrl"`
	ParentID int    `json:"parentCategoryId"`
}

// Author represents an addon author
type Author struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Logo represents addon logo/thumbnail
type Logo struct {
	ID           int    `json:"id"`
	ModID        int    `json:"modId"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnailUrl"`
	URL          string `json:"url"`
}

// File represents an addon file/release
type File struct {
	ID           int       `json:"id"`
	GameID       int       `json:"gameId"`
	ModID        int       `json:"modId"`
	DisplayName  string    `json:"displayName"`
	FileName     string    `json:"fileName"`
	FileDate     time.Time `json:"fileDate"`
	GameVersions []string  `json:"gameVersions"`
}

// FileIndex for quick file lookup
type FileIndex struct {
	GameVersion       string    `json:"gameVersion"`
	FileID            int       `json:"fileId"`
	Filename          string    `json:"filename"`
	ReleaseType       int       `json:"releaseType"`
	GameVersionTypeID int       `json:"gameVersionTypeId"`
	ModLoader         *int      `json:"modLoader"`
}

// GetCategoriesResponse is the response from /v1/categories
type GetCategoriesResponse struct {
	Data []Category `json:"data"`
}
