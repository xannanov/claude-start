package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const unsubscribeTTL = 30 * 24 * time.Hour // 30 дней

type unsubscribePayload struct {
	UserID     string `json:"u"`
	ScheduleID int    `json:"s"`
	ExpiresAt  int64  `json:"e"`
}

// GenerateUnsubscribeToken создаёт подписанный токен для отписки.
func GenerateUnsubscribeToken(secret []byte, userID string, scheduleID int) (string, error) {
	payload := unsubscribePayload{
		UserID:     userID,
		ScheduleID: scheduleID,
		ExpiresAt:  time.Now().Add(unsubscribeTTL).Unix(),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("ошибка кодирования токена: %w", err)
	}

	payloadB64 := base64.URLEncoding.EncodeToString(data)
	sig := signHMAC(secret, payloadB64)

	return payloadB64 + "." + sig, nil
}

// ValidateUnsubscribeToken проверяет и декодирует токен отписки.
// Возвращает userID и scheduleID.
func ValidateUnsubscribeToken(secret []byte, token string) (string, int, error) {
	// Разделяем payload.signature
	dotIdx := -1
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			dotIdx = i
			break
		}
	}
	if dotIdx == -1 {
		return "", 0, fmt.Errorf("невалидный токен отписки")
	}

	payloadB64 := token[:dotIdx]
	sig := token[dotIdx+1:]

	// Проверяем подпись
	expectedSig := signHMAC(secret, payloadB64)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", 0, fmt.Errorf("невалидная подпись токена")
	}

	// Декодируем payload
	data, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", 0, fmt.Errorf("ошибка декодирования токена: %w", err)
	}

	var payload unsubscribePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", 0, fmt.Errorf("ошибка парсинга токена: %w", err)
	}

	// Проверяем срок действия
	if time.Now().Unix() > payload.ExpiresAt {
		return "", 0, fmt.Errorf("токен отписки истёк")
	}

	return payload.UserID, payload.ScheduleID, nil
}

func signHMAC(secret []byte, data string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}
