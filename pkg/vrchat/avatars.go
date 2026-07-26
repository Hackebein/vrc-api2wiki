package vrchat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (c *Client) FetchPaidMarketplaceAvatars(snapshotDir string, limit int) ([]map[string]any, error) {
	var all []map[string]any
	offset := 0
	const n = 100
	for {
		pageSize := n
		if limit > 0 {
			remaining := limit - len(all)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}
		u := fmt.Sprintf("%s/avatars?n=%d&offset=%d&marketplace=paid&releaseStatus=all", apiBase, pageSize, offset)
		body, err := c.AuthedGet(u)
		if err != nil {
			return nil, err
		}
		var page []map[string]any
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("parse avatars offset %d: %w", offset, err)
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if snapshotDir != "" {
			dir := filepath.Join(snapshotDir, "Avatars")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, err
			}
			for _, av := range page {
				id, _ := av["id"].(string)
				if id == "" {
					continue
				}
				if err := writeJSON(filepath.Join(dir, id+".json"), av); err != nil {
					return nil, err
				}
			}
		}
		if limit > 0 && len(all) >= limit {
			all = all[:limit]
			break
		}
		if len(page) < pageSize {
			break
		}
		offset += len(page)
	}
	return all, nil
}
