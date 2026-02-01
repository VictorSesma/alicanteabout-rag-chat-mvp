# wordpress/AGENTS

Guidance for WordPress plugins.

- Treat each plugin as a standalone WP package.
- If plugin code changes, bump the version and regenerate:
  - wordpress/alicanteabout-chat-token.zip
  - wordpress/alicanteabout-chat-widget.zip
- Do not edit the zip files directly; regenerate from source folders.
- Zip structure must have the plugin folder at archive root:
  - Run: `(cd wordpress && zip -r alicanteabout-chat-widget.zip alicanteabout-chat-widget)`
  - Run: `(cd wordpress && zip -r alicanteabout-chat-token.zip alicanteabout-chat-token)`
- Keep settings keys stable; they are stored in wp_options.
- Language gating must use the page <html lang> attribute.
