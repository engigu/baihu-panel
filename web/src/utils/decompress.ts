import pako from 'pako'
import { decompress } from 'fzstd'

export function decompressFromBase64(compressed: string): string {
  if (!compressed) return ''
  try {
    let isZstd = false
    let dataToDecode = compressed

    if (compressed.startsWith('zstd:')) {
      isZstd = true
      dataToDecode = compressed.slice(5)
    }

    // Decode base64 to binary
    const binaryString = atob(dataToDecode)
    const bytes = new Uint8Array(binaryString.length)
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }

    if (isZstd) {
      // Decompress Zstd
      const decompressedBytes = decompress(bytes)
      return new TextDecoder().decode(decompressedBytes)
    }

    // Decompress zlib
    const decompressed = pako.inflate(bytes, { to: 'string' })
    return decompressed
  } catch (e) {
    console.error('Failed to decompress data:', e)
    return compressed
  }
}

