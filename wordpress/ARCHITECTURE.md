# wordpress/ARCHITECTURE

Plugins

alicanteabout-chat-token
- Purpose: issue short-lived JWTs for the chat API.
- Settings: JWT secret + per-IP rate limit.
- Endpoint: /wp-json/alicanteabout/v1/chat-token
- Storage: option key alicanteabout_chat_token_settings.

alicanteabout-chat-widget
- Purpose: render floating chat UI and stream responses.
- Settings: API URL, token URL, labels, colors, allowed languages.
- Bootstrap: loads minimal button first, then lazy-loads widget.js/css on click.
- Language gate: checks document.documentElement lang against allowed list.

Runtime flow
- Widget requests token from WP endpoint.
- Token plugin signs JWT (HS256) with short TTL.
- Widget calls Go chat API with Authorization: Bearer <token>.
