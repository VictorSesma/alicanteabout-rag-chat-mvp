<?php
/**
 * Plugin Name: AlicanteAbout Chat Token
 * Description: Issues short-lived JWTs for the AlicanteAbout chat API.
 * Version: 0.1.0
 * Author: AlicanteAbout
 */

if (!defined('ABSPATH')) {
	exit;
}

define('AAT_CHAT_TOKEN_VERSION', '0.1.0');
define('AAT_CHAT_TOKEN_OPTION', 'alicanteabout_chat_token_settings');
define('AAT_CHAT_TOKEN_DEFAULT_ISSUER', 'alicanteabout.com');
define('AAT_CHAT_TOKEN_DEFAULT_AUDIENCE', 'alicanteabout-chat');
define('AAT_CHAT_TOKEN_DEFAULT_TTL', 120);
define('AAT_CHAT_TOKEN_DEFAULT_RATE_LIMIT', 30);
define('AAT_CHAT_TOKEN_RATE_WINDOW', 60);

require_once __DIR__ . '/includes/class-aat-jwt.php';
require_once __DIR__ . '/includes/class-aat-rate-limiter.php';
require_once __DIR__ . '/includes/class-aat-settings.php';
require_once __DIR__ . '/includes/class-aat-token-endpoint.php';

final class AAT_Chat_Token_Plugin {
	public function __construct() {
		add_action('admin_menu', array($this, 'register_menu'));
		add_action('admin_init', array($this, 'register_settings'));
		add_action('rest_api_init', array($this, 'register_routes'));
	}

	public function register_menu() {
		AAT_Chat_Token_Settings::register_menu();
	}

	public function register_settings() {
		AAT_Chat_Token_Settings::register_settings();
	}

	public function register_routes() {
		AAT_Chat_Token_Endpoint::register_routes();
	}
}

new AAT_Chat_Token_Plugin();
