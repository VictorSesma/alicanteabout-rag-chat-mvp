<?php

if (!defined('ABSPATH')) {
	exit;
}

final class AAT_Chat_Token_Rate_Limiter {
	public static function allow($ip, $limit, $window) {
		if ($ip === '') {
			return false;
		}
		$key = 'aat_chat_token_rl_' . md5($ip);
		$state = get_transient($key);
		$now = time();
		if (!is_array($state) || !isset($state['count'], $state['reset']) || $now > $state['reset']) {
			$state = array(
				'count' => 1,
				'reset' => $now + $window,
			);
			set_transient($key, $state, $window);
			return true;
		}
		if ($state['count'] >= $limit) {
			return false;
		}
		$state['count'] += 1;
		$ttl = max(1, $state['reset'] - $now);
		set_transient($key, $state, $ttl);
		return true;
	}
}
