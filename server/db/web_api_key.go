package db

import (
	"database/sql"
	"server/models"
)

// SaveWebAPIKey 保存WebAPIKey到数据库
func SaveWebAPIKey(key *models.WebAPIKey) error {
	_, err := db.Exec(`
		INSERT INTO web_api_keys (id, user_id, key, name, description, used, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, key.ID, key.UserID, key.Key, key.Name, key.Description, key.Used, key.ExpiresAt, key.CreatedAt)
	return err
}

// GetWebAPIKeyByKey 根据key获取WebAPIKey
func GetWebAPIKeyByKey(key string) (*models.WebAPIKey, error) {
	var apiKey models.WebAPIKey
	err := db.QueryRow(`
		SELECT id, user_id, key, name, description, used, expires_at, created_at
		FROM web_api_keys
		WHERE key = ?
	`, key).Scan(
		&apiKey.ID, &apiKey.UserID, &apiKey.Key, &apiKey.Name, &apiKey.Description,
		&apiKey.Used, &apiKey.ExpiresAt, &apiKey.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &apiKey, err
}

// MarkWebAPIKeyAsUsed 将WebAPIKey标记为已使用
func MarkWebAPIKeyAsUsed(id string) error {
	_, err := db.Exec(`
		UPDATE web_api_keys
		SET used = true
		WHERE key = ?
	`, id)
	return err
}
