package api

import "github.com/xhanio/zen/pkg/types/entity"

type SearchResponse struct {
	Query    string              `json:"query"`
	Scope    string              `json:"scope"`
	Cards    []*entity.SearchHit `json:"cards"`
	Messages []*entity.SearchHit `json:"messages"`
}
