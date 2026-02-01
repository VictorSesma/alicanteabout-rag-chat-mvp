package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"content-rag-chat/internal/chat"
	appconfig "content-rag-chat/internal/config"
)

func main() {
	if err := appconfig.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}

	secret := envString("CHAT_JWT_SECRET", "")
	issuer := envString("CHAT_JWT_ISSUER", "alicanteabout.com")
	audience := envString("CHAT_JWT_AUDIENCE", "alicanteabout-chat")
	ttl := envDuration("CHAT_JWT_TTL", 120*time.Second)

	flag.StringVar(&secret, "secret", secret, "JWT secret (or CHAT_JWT_SECRET)")
	flag.StringVar(&issuer, "issuer", issuer, "JWT issuer")
	flag.StringVar(&audience, "audience", audience, "JWT audience")
	flag.DurationVar(&ttl, "ttl", ttl, "JWT time-to-live")
	flag.Parse()

	if secret == "" {
		log.Fatal("CHAT_JWT_SECRET is not set")
	}

	now := time.Now()
	token, err := chat.BuildJWT(secret, issuer, audience, now, ttl)
	if err != nil {
		log.Fatal(err)
	}
	expiresAt := now.Add(ttl)
	fmt.Println(token)
	fmt.Printf("expires_at=%s\n", expiresAt.Format(time.RFC3339))
	fmt.Printf("expires_in=%s\n", ttl.String())
}

func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return def
}
