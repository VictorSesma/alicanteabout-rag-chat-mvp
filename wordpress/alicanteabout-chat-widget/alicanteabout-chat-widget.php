<?php
/**
 * Plugin Name: AlicanteAbout Chat Widget
 * Description: Floating chat widget for AlicanteAbout.
 * Version: 0.1.5
 * Author: AlicanteAbout
 */

if (!defined('ABSPATH')) {
	exit;
}

define('AAT_CHAT_WIDGET_VERSION', '0.1.5');
define('AAT_CHAT_WIDGET_OPTION', 'alicanteabout_chat_widget_settings');

require_once __DIR__ . '/includes/class-aat-chat-widget-settings.php';

final class AAT_Chat_Widget_Plugin {
	public function __construct() {
		add_action('admin_menu', array($this, 'register_menu'));
		add_action('admin_init', array($this, 'register_settings'));
		add_action('wp_enqueue_scripts', array($this, 'enqueue_assets'));
	}

	public function register_menu() {
		AAT_Chat_Widget_Settings::register_menu();
	}

	public function register_settings() {
		AAT_Chat_Widget_Settings::register_settings();
	}

	public function enqueue_assets() {
		$settings = AAT_Chat_Widget_Settings::get_settings();
		$handle = 'aat-chat-widget-bootstrap';
		$bootstrap_path = __DIR__ . '/assets/bootstrap.js';
		$widget_js_path = __DIR__ . '/assets/widget.js';
		$widget_css_path = __DIR__ . '/assets/widget.css';
		wp_enqueue_script(
			$handle,
			plugins_url('assets/bootstrap.js', __FILE__),
			array(),
			file_exists($bootstrap_path) ? filemtime($bootstrap_path) : AAT_CHAT_WIDGET_VERSION,
			true
		);
		wp_localize_script(
			$handle,
			'AATChatWidgetConfig',
			array(
				'apiUrl' => $settings['api_url'],
				'tokenUrl' => $settings['token_url'],
				'buttonLabel' => $settings['button_label'],
				'accentColor' => $settings['accent_color'],
				'title' => $settings['title'],
				'disclaimer' => $settings['disclaimer'],
				'gdprDisclaimer' => $settings['gdpr_disclaimer'],
				'allowedLanguages' => $settings['allowed_languages'],
				'buttonPosition' => $settings['button_position'],
				'offsetX' => $settings['offset_x'],
				'offsetY' => $settings['offset_y'],
				'jsUrl' => plugins_url('assets/widget.js', __FILE__) . $this->asset_version($widget_js_path),
				'cssUrl' => plugins_url('assets/widget.css', __FILE__) . $this->asset_version($widget_css_path),
			)
		);
	}

	private function asset_version($path) {
		if (file_exists($path)) {
			return '?v=' . filemtime($path);
		}
		return '';
	}
}

new AAT_Chat_Widget_Plugin();
