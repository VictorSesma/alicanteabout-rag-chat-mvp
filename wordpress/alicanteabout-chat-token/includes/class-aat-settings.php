<?php

if (!defined('ABSPATH')) {
	exit;
}

final class AAT_Chat_Token_Settings {
	public static function register_menu() {
		add_options_page(
			'AlicanteAbout Chat Token',
			'Chat Token',
			'manage_options',
			'aat-chat-token',
			array(__CLASS__, 'render_page')
		);
	}

	public static function register_settings() {
		register_setting('aat_chat_token', AAT_CHAT_TOKEN_OPTION, array(__CLASS__, 'sanitize'));

		add_settings_section(
			'aat_chat_token_main',
			'Chat Token Settings',
			'__return_null',
			'aat-chat-token'
		);

		add_settings_field(
			'aat_chat_token_secret',
			'JWT Secret',
			array(__CLASS__, 'render_secret_field'),
			'aat-chat-token',
			'aat_chat_token_main'
		);

		add_settings_field(
			'aat_chat_token_rate_limit',
			'Rate Limit (per minute)',
			array(__CLASS__, 'render_rate_limit_field'),
			'aat-chat-token',
			'aat_chat_token_main'
		);
	}

	public static function sanitize($input) {
		$output = array();
		$output['secret'] = isset($input['secret']) ? sanitize_text_field($input['secret']) : '';
		$output['rate_limit'] = isset($input['rate_limit']) ? absint($input['rate_limit']) : AAT_CHAT_TOKEN_DEFAULT_RATE_LIMIT;
		if ($output['rate_limit'] < 1) {
			$output['rate_limit'] = AAT_CHAT_TOKEN_DEFAULT_RATE_LIMIT;
		}
		return $output;
	}

	public static function get_settings() {
		$defaults = array(
			'secret' => '',
			'rate_limit' => AAT_CHAT_TOKEN_DEFAULT_RATE_LIMIT,
		);
		$settings = get_option(AAT_CHAT_TOKEN_OPTION, array());
		if (!is_array($settings)) {
			$settings = array();
		}
		return wp_parse_args($settings, $defaults);
	}

	public static function render_secret_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[secret]" value="%2$s" class="regular-text" autocomplete="off" />',
			esc_attr(AAT_CHAT_TOKEN_OPTION),
			esc_attr($settings['secret'])
		);
		echo '<p class="description">Keep this secret private. Used to sign JWTs.</p>';
	}

	public static function render_rate_limit_field() {
		$settings = self::get_settings();
		printf(
			'<input type="number" min="1" name="%1$s[rate_limit]" value="%2$d" class="small-text" />',
			esc_attr(AAT_CHAT_TOKEN_OPTION),
			intval($settings['rate_limit'])
		);
		echo '<p class="description">Applies per IP per minute.</p>';
	}

	public static function render_page() {
		if (!current_user_can('manage_options')) {
			return;
		}
		echo '<div class="wrap">';
		echo '<h1>AlicanteAbout Chat Token</h1>';
		echo '<form method="post" action="options.php">';
		settings_fields('aat_chat_token');
		do_settings_sections('aat-chat-token');
		submit_button();
		echo '</form>';
		echo '<p><strong>Token endpoint:</strong> /wp-json/alicanteabout/v1/chat-token</p>';
		echo '</div>';
	}
}
