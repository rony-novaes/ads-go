package ads

type Ad struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	ImageURL  string `json:"image_url"`
	TargetURL string `json:"target_url"`
	Weight    int    `json:"weight"`
	Active    bool   `json:"active"`
}
