/**
 * 基于浏览器原生 Web Crypto API 的端到端安全加解密工具
 * 采用现代椭圆曲线 ECDH (NIST P-256 / secp256r1) + AES-256-GCM
 * 零第三方依赖，纯内存协商，耗时 < 1ms，支持任意长度大文本机密
 */

export interface EcdhCipherPayload {
  server_public_key: string // 服务端临时 P-256 公钥 (Base64)
  ciphertext: string        // AES-256-GCM 加密后的密文 (Base64, 包含 16 字节 Auth Tag)
  nonce: string             // AES-256-GCM Nonce/IV (Base64, 12 字节)
}

// 辅助函数：Base64 字符串转 Uint8Array
export function base64ToUint8Array(base64: string): Uint8Array {
  const binaryString = window.atob(base64)
  const len = binaryString.length
  const buffer = new ArrayBuffer(len)
  const bytes = new Uint8Array(buffer)
  for (let i = 0; i < len; i++) {
    bytes[i] = binaryString.charCodeAt(i)
  }
  return bytes
}

// 辅助函数：ArrayBuffer 转 Base64
export function arrayBufferToBase64(buffer: ArrayBuffer): string {
  let binary = ''
  const bytes = new Uint8Array(buffer)
  const len = bytes.byteLength
  for (let i = 0; i < len; i++) {
    binary += String.fromCharCode(bytes[i] as number)
  }
  return window.btoa(binary)
}

/**
 * 检查当前浏览器环境是否支持安全加解密 (Web Crypto API)
 * 在局域网纯 HTTP IP 访问等非安全上下文中，浏览器会禁用 crypto.subtle
 */
export function isWebCryptoSupported(): boolean {
  return typeof window !== 'undefined' && !!window.crypto && !!window.crypto.subtle
}

/**
 * 生成临时的 ECDH 密钥对 (P-256 / secp256r1)
 * 相比 RSA 2048 生成需 3~8 秒，P-256 仅需 0.5ms，绝不卡死主线程
 */
export async function generateEcdhKeyPair(): Promise<CryptoKeyPair> {
  if (!isWebCryptoSupported()) {
    throw new Error('当前浏览器环境不支持安全加密 (Web Crypto API 不可用，请确保使用 HTTPS 或 localhost 访问)')
  }

  return await window.crypto.subtle.generateKey(
    {
      name: 'ECDH',
      namedCurve: 'P-256'
    },
    true, // extractable
    ['deriveKey', 'deriveBits']
  )
}

/**
 * 将 ECDH 公钥导出为 Raw 格式 Base64 (65 字节未压缩点 0x04 || X || Y)
 */
export async function exportEcdhPublicKey(publicKey: CryptoKey): Promise<string> {
  const rawBuffer = await window.crypto.subtle.exportKey('raw', publicKey)
  return arrayBufferToBase64(rawBuffer)
}

/**
 * 使用本地临时私钥与服务端公钥进行 ECDH 协商并派生 AES-256-GCM 密钥，解密明文内容
 */
export async function decryptEcdhPayload(
  privateKey: CryptoKey,
  payload: EcdhCipherPayload
): Promise<string> {
  if (!payload.server_public_key || !payload.ciphertext || !payload.nonce) {
    throw new Error('密文载荷不完整')
  }

  const serverPubBytes = base64ToUint8Array(payload.server_public_key)
  const nonceBytes = base64ToUint8Array(payload.nonce)
  const ciphertextBytes = base64ToUint8Array(payload.ciphertext)

  // 1. 将服务端临时公钥导入为 ECDH CryptoKey
  const serverPublicKey = await window.crypto.subtle.importKey(
    'raw',
    serverPubBytes as unknown as BufferSource,
    {
      name: 'ECDH',
      namedCurve: 'P-256'
    },
    false,
    []
  )

  // 2. 派生 256 位 AES-GCM 对称密钥 (根据 W3C 规范直接对齐 P-256 协商的 32 字节共享秘钥)
  const aesKey = await window.crypto.subtle.deriveKey(
    {
      name: 'ECDH',
      public: serverPublicKey
    },
    privateKey,
    {
      name: 'AES-GCM',
      length: 256
    },
    false,
    ['decrypt']
  )

  // 3. 使用 AES-GCM 解密密文获得明文
  const decryptedBuffer = await window.crypto.subtle.decrypt(
    {
      name: 'AES-GCM',
      iv: nonceBytes as unknown as BufferSource
    },
    aesKey,
    ciphertextBytes as unknown as BufferSource
  )

  const decoder = new TextDecoder('utf-8')
  return decoder.decode(decryptedBuffer)
}
