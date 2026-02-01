<?php

if (!defined('ABSPATH')) {
	exit;
}

final class AAT_Chat_Token_JWT {
	public static function encode(array $payload, $secret) {
		$header = array('alg' => 'HS256', 'typ' => 'JWT');
		$segments = array(
			self::base64url_encode(json_encode($header)),
			self::base64url_encode(json_encode($payload)),
		);
		$signing_input = implode('.', $segments);
		$signature = hash_hmac('sha256', $signing_input, $secret, true);
		$segments[] = self::base64url_encode($signature);
		return implode('.', $segments);
	}

	public static function base64url_encode($data) {
		$encoded = base64_encode($data);
		$encoded = str_replace(array('+', '/', '='), array('-', '_', ''), $encoded);
		return $encoded;
	}
}
