.PHONY: db-up db-down db-logs db-migrate db-reset db-shell wp-zip wp-zip-token wp-zip-widget

ifneq (,$(wildcard .env))
include .env
export
endif

GOOSE ?= $(shell command -v goose 2>/dev/null)
ifeq ($(GOOSE),)
GOOSE := $(HOME)/go/bin/goose
endif

db-up:
	docker compose up -d db

db-down:
	docker compose down

db-logs:
	docker compose logs -f db

db-migrate:
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" postgres "$(CHAT_DB_DSN)" up

db-reset:
	docker compose down -v

db-shell:
	docker compose exec db psql -U alicante -d alicanteabout

wp-zip: wp-zip-token wp-zip-widget

wp-zip-token:
	cd wordpress && zip -r alicanteabout-chat-token.zip alicanteabout-chat-token

wp-zip-widget:
	cd wordpress && zip -r alicanteabout-chat-widget.zip alicanteabout-chat-widget
