# wordpress/DESIGN

Design guidelines for the WordPress layer.

- Keep PHP minimal; use core WP APIs only.
- Prefer simple, static assets for reliability.
- Token endpoint must be cache-disabled (no-store).
- Widget should be lazy-loaded and non-blocking.
- Keep UI text editable via WP settings.
- Avoid collecting personal data in the frontend.
