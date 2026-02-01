<?php

use PHPUnit\Framework\TestCase;

final class JwtTest extends TestCase {
	public function testEncodeProducesValidSignature() {
		$payload = array(
			'iss' => 'alicanteabout.com',
			'aud' => 'alicanteabout-chat',
			'iat' => 1700000000,
			'exp' => 1700000123,
		);
		$secret = 'test-secret';

		$token = AAT_Chat_Token_JWT::encode($payload, $secret);
		$parts = explode('.', $token);
		$this->assertCount(3, $parts);

		$header = json_decode($this->base64url_decode($parts[0]), true);
		$body = json_decode($this->base64url_decode($parts[1]), true);
		$this->assertSame('HS256', $header['alg']);
		$this->assertSame('JWT', $header['typ']);
		$this->assertSame($payload, $body);

		$expected_signature = hash_hmac('sha256', $parts[0] . '.' . $parts[1], $secret, true);
		$actual_signature = $this->base64url_decode($parts[2]);
		$this->assertSame($expected_signature, $actual_signature);
	}

	private function base64url_decode($data) {
		$remainder = strlen($data) % 4;
		if ($remainder) {
			$data .= str_repeat('=', 4 - $remainder);
		}
		$data = str_replace(array('-', '_'), array('+', '/'), $data);
		return base64_decode($data);
	}
}
