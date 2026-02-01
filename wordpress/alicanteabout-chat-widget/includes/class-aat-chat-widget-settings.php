<?php

if (!defined('ABSPATH')) {
	exit;
}

final class AAT_Chat_Widget_Settings {
	public static function register_menu() {
		add_options_page(
			'AlicanteAbout Chat Widget',
			'Chat Widget',
			'manage_options',
			'aat-chat-widget',
			array(__CLASS__, 'render_page')
		);
	}

	public static function register_settings() {
		register_setting('aat_chat_widget', AAT_CHAT_WIDGET_OPTION, array(__CLASS__, 'sanitize'));

		add_settings_section(
			'aat_chat_widget_main',
			'Chat Widget Settings',
			'__return_null',
			'aat-chat-widget'
		);

		add_settings_field(
			'aat_chat_widget_api_url',
			'Chat API URL',
			array(__CLASS__, 'render_api_url_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_token_url',
			'Token Endpoint URL',
			array(__CLASS__, 'render_token_url_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_button_label',
			'Button Label',
			array(__CLASS__, 'render_button_label_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_accent_color',
			'Accent Color',
			array(__CLASS__, 'render_accent_color_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_title',
			'Widget Title',
			array(__CLASS__, 'render_title_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_disclaimer',
			'Content Disclaimer',
			array(__CLASS__, 'render_disclaimer_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_gdpr_disclaimer',
			'GDPR Disclaimer',
			array(__CLASS__, 'render_gdpr_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_allowed_languages',
			'Allowed Languages',
			array(__CLASS__, 'render_allowed_languages_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_button_position',
			'Button Position',
			array(__CLASS__, 'render_button_position_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_offset_x',
			'Button Offset X (px)',
			array(__CLASS__, 'render_offset_x_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);

		add_settings_field(
			'aat_chat_widget_offset_y',
			'Button Offset Y (px)',
			array(__CLASS__, 'render_offset_y_field'),
			'aat-chat-widget',
			'aat_chat_widget_main'
		);
	}

	public static function sanitize($input) {
		$output = array();
		$output['api_url'] = isset($input['api_url']) ? esc_url_raw($input['api_url']) : '';
		$output['token_url'] = isset($input['token_url']) ? esc_url_raw($input['token_url']) : '';
		$output['button_label'] = isset($input['button_label']) ? sanitize_text_field($input['button_label']) : '';
		$output['accent_color'] = isset($input['accent_color']) ? sanitize_text_field($input['accent_color']) : '';
		$output['title'] = isset($input['title']) ? sanitize_text_field($input['title']) : '';
		$output['disclaimer'] = isset($input['disclaimer']) ? sanitize_text_field($input['disclaimer']) : '';
		$output['gdpr_disclaimer'] = isset($input['gdpr_disclaimer']) ? sanitize_text_field($input['gdpr_disclaimer']) : '';
		$output['allowed_languages'] = isset($input['allowed_languages']) ? sanitize_text_field($input['allowed_languages']) : '';
		$output['button_position'] = isset($input['button_position']) ? sanitize_text_field($input['button_position']) : 'bottom-right';
		if (!in_array($output['button_position'], self::position_options(), true)) {
			$output['button_position'] = 'bottom-right';
		}
		$output['offset_x'] = isset($input['offset_x']) ? absint($input['offset_x']) : 20;
		$output['offset_y'] = isset($input['offset_y']) ? absint($input['offset_y']) : 20;
		return $output;
	}

	public static function get_settings() {
		$defaults = array(
			'api_url' => 'https://api.alicanteabout.com/chat',
			'token_url' => home_url('/wp-json/alicanteabout/v1/chat-token'),
			'button_label' => 'Ask Alicante',
			'accent_color' => '#0f4c5c',
			'title' => 'Ask Alicante',
			'disclaimer' => 'Answers are based on AlicanteAbout content.',
			'gdpr_disclaimer' => 'We log questions for quality. Please avoid personal data.',
			'allowed_languages' => 'en',
			'button_position' => 'bottom-right',
			'offset_x' => 20,
			'offset_y' => 20,
		);
		$settings = get_option(AAT_CHAT_WIDGET_OPTION, array());
		if (!is_array($settings)) {
			$settings = array();
		}
		return wp_parse_args($settings, $defaults);
	}

	public static function render_api_url_field() {
		$settings = self::get_settings();
		printf(
			'<input type="url" name="%1$s[api_url]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['api_url'])
		);
	}

	public static function render_token_url_field() {
		$settings = self::get_settings();
		printf(
			'<input type="url" name="%1$s[token_url]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['token_url'])
		);
	}

	public static function render_button_label_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[button_label]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['button_label'])
		);
	}

	public static function render_accent_color_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[accent_color]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['accent_color'])
		);
		echo '<p class="description">Hex color used for the button and highlights.</p>';
	}

	public static function render_title_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[title]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['title'])
		);
	}

	public static function render_disclaimer_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[disclaimer]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['disclaimer'])
		);
	}

	public static function render_gdpr_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[gdpr_disclaimer]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['gdpr_disclaimer'])
		);
	}

	public static function render_allowed_languages_field() {
		$settings = self::get_settings();
		printf(
			'<input type="text" name="%1$s[allowed_languages]" value="%2$s" class="regular-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			esc_attr($settings['allowed_languages'])
		);
		echo '<p class="description">Comma-separated language codes (e.g., "en"). Uses the page lang attribute.</p>';
	}

	public static function render_button_position_field() {
		$settings = self::get_settings();
		$current = $settings['button_position'];
		echo '<select name="' . esc_attr(AAT_CHAT_WIDGET_OPTION) . '[button_position]">';
		foreach (self::position_options() as $value) {
			$label = ucwords(str_replace('-', ' ', $value));
			printf(
				'<option value="%1$s"%2$s>%3$s</option>',
				esc_attr($value),
				selected($current, $value, false),
				esc_html($label)
			);
		}
		echo '</select>';
		echo '<p class="description">Choose where the floating button appears.</p>';
	}

	public static function render_offset_x_field() {
		$settings = self::get_settings();
		printf(
			'<input type="number" min="0" name="%1$s[offset_x]" value="%2$d" class="small-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			intval($settings['offset_x'])
		);
		echo '<p class="description">Horizontal distance from the chosen edge.</p>';
	}

	public static function render_offset_y_field() {
		$settings = self::get_settings();
		printf(
			'<input type="number" min="0" name="%1$s[offset_y]" value="%2$d" class="small-text" />',
			esc_attr(AAT_CHAT_WIDGET_OPTION),
			intval($settings['offset_y'])
		);
		echo '<p class="description">Vertical distance from the chosen edge.</p>';
	}

	private static function position_options() {
		return array('bottom-right', 'bottom-left', 'top-right', 'top-left');
	}

	public static function render_page() {
		if (!current_user_can('manage_options')) {
			return;
		}
		echo '<div class="wrap">';
		echo '<h1>AlicanteAbout Chat Widget</h1>';
		echo '<form method="post" action="options.php">';
		settings_fields('aat_chat_widget');
		do_settings_sections('aat-chat-widget');
		submit_button();
		echo '</form>';
		echo '</div>';
	}
}
