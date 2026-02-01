package main

import (
	"time"

	"content-rag-chat/internal/chat"
)

type tokenConfig struct {
	Secret   string
	Issuer   string
	Audience string
	TTL      time.Duration
}

func buildToken(cfg tokenConfig, now time.Time) (string, error) {
	return chat.BuildJWT(cfg.Secret, cfg.Issuer, cfg.Audience, now, cfg.TTL)
}
