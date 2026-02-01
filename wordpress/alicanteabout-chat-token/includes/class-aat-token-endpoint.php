<?php

if (!defined('ABSPATH')) {
	exit;
}

final class AAT_Chat_Token_Endpoint {
	public static function register_routes() {
		register_rest_route(
			'alicanteabout/v1',
			'/chat-token',
			array(
				'methods' => WP_REST_Server::READABLE,
				'callback' => array(__CLASS__, 'handle_request'),
				'permission_callback' => '__return_true',
			)
		);
	}

	public static function handle_request($request) {
		$settings = AAT_Chat_Token_Settings::get_settings();
		$secret = $settings['secret'];
		if ($secret === '') {
			return new WP_REST_Response(
				array('error' => 'chat_token_not_configured'),
				500
			);
		}

		$ip = self::get_client_ip();
		$limit = intval($settings['rate_limit']);
		if (!AAT_Chat_Token_Rate_Limiter::allow($ip, $limit, AAT_CHAT_TOKEN_RATE_WINDOW)) {
			return new WP_REST_Response(
				array('error' => 'rate_limit_exceeded'),
				429
			);
		}

		$now = time();
		$payload = array(
			'iss' => apply_filters('alicanteabout_chat_jwt_issuer', AAT_CHAT_TOKEN_DEFAULT_ISSUER),
			'aud' => apply_filters('alicanteabout_chat_jwt_audience', AAT_CHAT_TOKEN_DEFAULT_AUDIENCE),
			'iat' => $now,
			'exp' => $now + AAT_CHAT_TOKEN_DEFAULT_TTL,
		);
		$token = AAT_Chat_Token_JWT::encode($payload, $secret);
		$response = new WP_REST_Response(
			array(
				'token' => $token,
				'expires_in' => AAT_CHAT_TOKEN_DEFAULT_TTL,
			),
			200
		);
		$response->header('Cache-Control', 'no-store');
		return $response;
	}

	private static function get_client_ip() {
		if (!empty($_SERVER['HTTP_X_FORWARDED_FOR'])) {
			$parts = explode(',', $_SERVER['HTTP_X_FORWARDED_FOR']);
			if (count($parts) > 0) {
				return trim($parts[0]);
			}
		}
		if (!empty($_SERVER['HTTP_X_REAL_IP'])) {
			return trim($_SERVER['HTTP_X_REAL_IP']);
		}
		if (!empty($_SERVER['REMOTE_ADDR'])) {
			return $_SERVER['REMOTE_ADDR'];
		}
		return '';
	}
}
