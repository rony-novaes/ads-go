package ads

type AdTypeVariant struct {
	File      string `json:"file"`
	Extension string `json:"extension"`
}

type ResponseAd struct {
	Breackpoint int                       `json:"breackpoint"`
	Code        string                    `json:"code"`
	Description string                    `json:"description"`
	Types       map[int]AdTypeVariant     `json:"types"`
}

type AdsResponse struct {
	Ads      []ResponseAd `json:"ads"`
	Redirect string       `json:"redirect"`
	Static   string       `json:"static"`
}
